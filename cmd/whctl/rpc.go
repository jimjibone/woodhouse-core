package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// apiPackage is the proto package of the services core serves.
const apiPackage = "woodhouse.api.v2.clients"

var heartbeatName = (&clientsapi.Heartbeat{}).ProtoReflect().Descriptor().FullName()

// Access is who may call a method.
type Access string

const (
	AccessNone   Access = "none"   // No session needed (login, refresh, ping).
	AccessUser   Access = "user"   // A user session, which whctl holds.
	AccessClient Access = "client" // A paired client (bridge) identity, which whctl does not hold.
)

// methodAccess mirrors core's auth policy (cmd/woodhouse-core/internal/auth):
// the auth services need no session, UserService and SystemService need a
// user, and everything else needs a paired client.
func methodAccess(md protoreflect.MethodDescriptor) Access {
	switch md.Parent().Name() {
	case "UserAuthService", "AuthService":
		return AccessNone
	case "UserService", "SystemService":
		return AccessUser
	}
	return AccessClient
}

// services returns core's API services, sorted by name.
func services() []protoreflect.ServiceDescriptor {
	var out []protoreflect.ServiceDescriptor
	protoregistry.GlobalFiles.RangeFilesByPackage(apiPackage, func(fd protoreflect.FileDescriptor) bool {
		for i := 0; i < fd.Services().Len(); i++ {
			out = append(out, fd.Services().Get(i))
		}
		return true
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// findService resolves a service by short or full name, case-insensitively.
func findService(name string) (protoreflect.ServiceDescriptor, error) {
	want := strings.ToLower(strings.Trim(name, "/."))
	for _, sd := range services() {
		if strings.ToLower(string(sd.Name())) == want || strings.ToLower(string(sd.FullName())) == want {
			return sd, nil
		}
	}
	return nil, fmt.Errorf("unknown service %q; run 'whctl list' to see them", name)
}

// findMethod resolves a method given as "Service/Method", "Service.Method",
// a full name (with or without a leading slash) or a bare method name that is
// unique among the methods whctl can call. Matching is case-insensitive.
func findMethod(name string) (protoreflect.MethodDescriptor, error) {
	want := strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(name, "/"), "/", "."))
	var matches []protoreflect.MethodDescriptor
	for _, sd := range services() {
		for i := 0; i < sd.Methods().Len(); i++ {
			md := sd.Methods().Get(i)
			full := strings.ToLower(string(md.FullName()))
			if full == want || strings.HasSuffix(full, "."+want) {
				matches = append(matches, md)
			}
		}
	}
	// A bare name such as SendAction exists on both UserService and
	// ClientService; prefer the one whctl can call.
	if len(matches) > 1 {
		var callable []protoreflect.MethodDescriptor
		for _, md := range matches {
			if methodAccess(md) != AccessClient {
				callable = append(callable, md)
			}
		}
		if len(callable) > 0 {
			matches = callable
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("unknown method %q; run 'whctl list' to see them", name)
	case 1:
		return matches[0], nil
	}
	names := make([]string, len(matches))
	for i, md := range matches {
		names[i] = methodName(md)
	}
	return nil, fmt.Errorf("method %q is ambiguous: %s", name, strings.Join(names, ", "))
}

// methodName is "Service/Method".
func methodName(md protoreflect.MethodDescriptor) string {
	return string(md.Parent().Name()) + "/" + string(md.Name())
}

// grpcPath is "/package.Service/Method".
func grpcPath(md protoreflect.MethodDescriptor) string {
	return "/" + string(md.Parent().FullName()) + "/" + string(md.Name())
}

func newMessage(desc protoreflect.MessageDescriptor) (proto.Message, error) {
	mt, err := protoregistry.GlobalTypes.FindMessageByName(desc.FullName())
	if err != nil {
		return nil, fmt.Errorf("message type %s: %w", desc.FullName(), err)
	}
	return mt.New().Interface(), nil
}

// parseRequest decodes a JSON request for md. Empty input is an empty request.
func parseRequest(md protoreflect.MethodDescriptor, data []byte) (proto.Message, error) {
	req, err := newMessage(md.Input())
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return req, nil
	}
	if err := protojson.Unmarshal(data, req); err != nil {
		return nil, fmt.Errorf("parse %s: %w", md.Input().Name(), err)
	}
	return req, nil
}

// isHeartbeat reports whether a stream response is a keepalive: either an
// entirely empty message (the older streams) or one whose only set field is a
// Heartbeat (the oneof streams). Either kind sent first after the initial
// state also marks the end of that state.
func isHeartbeat(m proto.Message) bool {
	set, heartbeat := 0, false
	m.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		set++
		if fd.Message() != nil && fd.Message().FullName() == heartbeatName {
			heartbeat = true
		}
		return true
	})
	return set == 0 || (set == 1 && heartbeat)
}

// Printer writes messages as JSON, one per line, or indented.
type Printer struct {
	w        *bufio.Writer
	pretty   bool
	defaults bool
}

func NewPrinter(w io.Writer, pretty, defaults bool) *Printer {
	return &Printer{w: bufio.NewWriter(w), pretty: pretty, defaults: defaults}
}

func (p *Printer) Print(m proto.Message) error {
	raw, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: p.defaults}.Marshal(m)
	if err != nil {
		return err
	}
	return p.PrintJSON(raw)
}

// PrintJSON re-encodes raw JSON so the output is stable: protojson
// deliberately varies its whitespace between runs.
func (p *Printer) PrintJSON(raw []byte) error {
	var buf bytes.Buffer
	var err error
	if p.pretty {
		err = json.Indent(&buf, raw, "", "  ")
	} else {
		err = json.Compact(&buf, raw)
	}
	if err != nil {
		return err
	}
	buf.WriteByte('\n')
	if _, err := p.w.Write(buf.Bytes()); err != nil {
		return err
	}
	return p.w.Flush()
}

// StreamOpts limits how much of a server stream is read. Zero values mean no
// limit; a stream with no limits runs until the server ends it or the user
// interrupts.
type StreamOpts struct {
	Count      int           // Stop after this many responses (heartbeats not counted).
	Duration   time.Duration // Stop after this long.
	Idle       time.Duration // Stop when no response arrives for this long.
	Snapshot   bool          // Stop at the first heartbeat: the end of the initial state.
	Heartbeats bool          // Also output heartbeats.
}

// Call invokes md with req and passes each response to emit. Unary calls are
// bounded by the session timeout; server streams by opts. Stopping a stream
// because a limit was reached, or because ctx was cancelled (Ctrl-C), is not
// an error.
func (s *Session) Call(ctx context.Context, md protoreflect.MethodDescriptor, req proto.Message, opts StreamOpts, emit func(proto.Message) error) error {
	access := methodAccess(md)
	if access == AccessClient {
		return fmt.Errorf("%s needs a paired client (bridge) identity; whctl logs in as a user", methodName(md))
	}
	if md.IsStreamingClient() {
		return fmt.Errorf("%s is a client-streaming RPC, which whctl does not support", methodName(md))
	}
	if access == AccessUser {
		if err := s.EnsureAccess(ctx, false); err != nil {
			return err
		}
	}

	emitted := false
	err := s.call(ctx, md, req, opts, func(m proto.Message) error {
		emitted = true
		return emit(m)
	})
	// The access token can be rejected before it expires, e.g. after core
	// restarts with a new signing key. Refresh once and retry, but only if
	// nothing was output, so a retry never duplicates output.
	if access == AccessUser && !emitted && status.Code(err) == codes.Unauthenticated {
		if rerr := s.EnsureAccess(ctx, true); rerr != nil {
			return rerr
		}
		err = s.call(ctx, md, req, opts, emit)
	}
	return err
}

func (s *Session) call(parent context.Context, md protoreflect.MethodDescriptor, req proto.Message, opts StreamOpts, emit func(proto.Message) error) error {
	streaming := md.IsStreamingServer()

	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	if !streaming {
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}

	// stopped records that we ended the stream on purpose, so the resulting
	// Canceled error is reported as success.
	var stopMu sync.Mutex
	stopped := false
	stop := func() {
		stopMu.Lock()
		stopped = true
		stopMu.Unlock()
		cancel()
	}
	isStopped := func() bool {
		stopMu.Lock()
		defer stopMu.Unlock()
		return stopped || parent.Err() != nil
	}

	if streaming && opts.Duration > 0 {
		t := time.AfterFunc(opts.Duration, stop)
		defer t.Stop()
	}
	var idle *time.Timer
	if streaming && opts.Idle > 0 {
		idle = time.AfterFunc(opts.Idle, stop)
		defer idle.Stop()
	}

	desc := &grpc.StreamDesc{StreamName: string(md.Name()), ServerStreams: streaming}
	stream, err := s.conn.NewStream(ctx, desc, grpcPath(md))
	if err != nil {
		if streaming && isStopped() {
			return nil
		}
		return err
	}
	if err := stream.SendMsg(req); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if err := stream.CloseSend(); err != nil {
		return err
	}

	count := 0
	for {
		res, err := newMessage(md.Output())
		if err != nil {
			return err
		}
		if err := stream.RecvMsg(res); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			if streaming && isStopped() {
				return nil
			}
			return err
		}
		if idle != nil {
			idle.Reset(opts.Idle)
		}

		if streaming && isHeartbeat(res) {
			if opts.Snapshot {
				return nil
			}
			if !opts.Heartbeats {
				continue
			}
			if err := emit(res); err != nil {
				return err
			}
			continue
		}

		if err := emit(res); err != nil {
			return err
		}
		count++
		if !streaming {
			return nil
		}
		if opts.Count > 0 && count >= opts.Count {
			stop()
			return nil
		}
	}
}
