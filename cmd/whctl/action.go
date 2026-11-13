package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// assignment is one "id=value" argument.
type assignment struct {
	id, value string
}

func parseAssignments(args []string) ([]assignment, error) {
	out := make([]assignment, 0, len(args))
	for _, arg := range args {
		id, value, ok := strings.Cut(arg, "=")
		if !ok || id == "" {
			return nil, fmt.Errorf("expected ID=VALUE, got %q", arg)
		}
		out = append(out, assignment{id: id, value: value})
	}
	return out, nil
}

// isRawValue reports whether s is a Value given as JSON, e.g.
// {"color":{"hue_sat":{"hue":120,"sat":100}}}.
func isRawValue(s string) bool {
	return strings.HasPrefix(strings.TrimSpace(s), "{")
}

func allRaw(as []assignment) bool {
	for _, a := range as {
		if !isRawValue(a.value) {
			return false
		}
	}
	return true
}

// parseValue converts a command-line value to a Value of the kind spec
// declares. A value written as JSON is decoded as a Value as is, which is the
// only way to give the structured kinds (color, list, struct, image).
func parseValue(spec *clientsapi.Spec, s string) (*clientsapi.Value, error) {
	if isRawValue(s) {
		v := &clientsapi.Value{}
		if err := protojson.Unmarshal([]byte(s), v); err != nil {
			return nil, fmt.Errorf("parse Value JSON: %w", err)
		}
		return v, nil
	}
	if spec == nil || spec.Kind == nil {
		return nil, fmt.Errorf("type unknown; give the value as Value JSON, e.g. '{\"text\":\"abc\"}'")
	}
	switch spec.Kind.(type) {
	case *clientsapi.Spec_Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return nil, fmt.Errorf("expected a bool: %w", err)
		}
		return &clientsapi.Value{Kind: &clientsapi.Value_Bool{Bool: b}}, nil
	case *clientsapi.Spec_Int:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("expected an int: %w", err)
		}
		return &clientsapi.Value{Kind: &clientsapi.Value_Int{Int: n}}, nil
	case *clientsapi.Spec_Float:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("expected a float: %w", err)
		}
		return &clientsapi.Value{Kind: &clientsapi.Value_Float{Float: f}}, nil
	case *clientsapi.Spec_Text:
		return &clientsapi.Value{Kind: &clientsapi.Value_Text{Text: s}}, nil
	case *clientsapi.Spec_Enum:
		return &clientsapi.Value{Kind: &clientsapi.Value_Enum{Enum: s}}, nil
	case *clientsapi.Spec_Duration:
		// A Go duration ("1m30s") or plain milliseconds.
		if ms, err := strconv.ParseInt(s, 10, 64); err == nil {
			return &clientsapi.Value{Kind: &clientsapi.Value_Duration{Duration: ms}}, nil
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("expected a duration such as 1m30s or milliseconds: %w", err)
		}
		return &clientsapi.Value{Kind: &clientsapi.Value_Duration{Duration: d.Milliseconds()}}, nil
	case *clientsapi.Spec_Time:
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return nil, fmt.Errorf("expected an RFC 3339 time: %w", err)
		}
		return &clientsapi.Value{Kind: &clientsapi.Value_Time{Time: &clientsapi.TimeValue{
			Seconds: t.Unix(),
			Nanos:   int32(t.Nanosecond()),
		}}}, nil
	}
	return nil, fmt.Errorf("structured value; give it as Value JSON, e.g. '{\"color\":{\"hue_sat\":{\"hue\":120,\"sat\":100}}}'")
}

// findDeviceService fetches the device and returns the named service, so
// values can be typed from its specs.
func (s *Session) findDeviceService(ctx context.Context, deviceID, endpointID, serviceID string) (*clientsapi.Service, error) {
	md, err := findMethod("UserService/GetDevices")
	if err != nil {
		return nil, err
	}
	var device *clientsapi.Device
	err = s.Call(ctx, md, &clientsapi.GetDevicesRequest{}, StreamOpts{}, func(m proto.Message) error {
		if d := m.(*clientsapi.Device); d.Id == deviceID {
			device = d
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get devices: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device %q not found", deviceID)
	}
	for _, ep := range device.Endpoints {
		if ep.Id != endpointID {
			continue
		}
		for _, srv := range ep.Services {
			if srv.Id == serviceID {
				return srv, nil
			}
		}
		return nil, fmt.Errorf("device %q endpoint %q has no service %q", deviceID, endpointID, serviceID)
	}
	return nil, fmt.Errorf("device %q has no endpoint %q", deviceID, endpointID)
}

// buildWrite builds a WriteAction, typing each value from the attribute spec.
func buildWrite(srv *clientsapi.Service, sets []assignment) (*clientsapi.WriteAction, error) {
	specs := map[string]*clientsapi.Spec{}
	if srv != nil {
		for _, a := range srv.Attrs {
			specs[a.Id] = a.Spec
		}
	}
	w := &clientsapi.WriteAction{}
	for _, set := range sets {
		spec, ok := specs[set.id]
		if srv != nil && !ok && !isRawValue(set.value) {
			return nil, fmt.Errorf("service %q has no attribute %q", srv.Id, set.id)
		}
		v, err := parseValue(spec, set.value)
		if err != nil {
			return nil, fmt.Errorf("attribute %q: %w", set.id, err)
		}
		w.Values = append(w.Values, &clientsapi.FieldValue{Id: set.id, Value: v})
	}
	return w, nil
}

// buildCommand builds a CommandAction, typing each param from the command
// spec.
func buildCommand(srv *clientsapi.Service, id string, params []assignment) (*clientsapi.CommandAction, error) {
	specs := map[string]*clientsapi.Spec{}
	if srv != nil {
		var cmd *clientsapi.CommandSpec
		for _, c := range srv.Commands {
			if c.Id == id {
				cmd = c
			}
		}
		if cmd == nil {
			return nil, fmt.Errorf("service %q has no command %q", srv.Id, id)
		}
		for _, p := range cmd.Params {
			specs[p.Id] = p.Spec
		}
	}
	c := &clientsapi.CommandAction{Id: id}
	for _, p := range params {
		spec, ok := specs[p.id]
		if srv != nil && !ok && !isRawValue(p.value) {
			return nil, fmt.Errorf("command %q has no param %q", id, p.id)
		}
		v, err := parseValue(spec, p.value)
		if err != nil {
			return nil, fmt.Errorf("param %q: %w", p.id, err)
		}
		c.Params = append(c.Params, &clientsapi.FieldValue{Id: p.id, Value: v})
	}
	return c, nil
}
