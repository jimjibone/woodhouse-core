package main

import (
	"bytes"
	"path/filepath"
	"slices"
	"testing"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"google.golang.org/protobuf/proto"
)

func TestFindMethod(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"UserService/GetDevices", "UserService/GetDevices"},
		{"userservice.getdevices", "UserService/GetDevices"},
		{"/woodhouse.api.v2.clients.UserService/GetDevices", "UserService/GetDevices"},
		{"GetDevices", "UserService/GetDevices"},
		// Also on ClientService, which whctl cannot call.
		{"SendAction", "UserService/SendAction"},
		{"ClientService/SendAction", "ClientService/SendAction"},
		{"Ping", "AuthService/Ping"},
	}
	for _, tt := range tests {
		md, err := findMethod(tt.in)
		if err != nil {
			t.Errorf("findMethod(%q): %v", tt.in, err)
			continue
		}
		if got := methodName(md); got != tt.want {
			t.Errorf("findMethod(%q) = %s, want %s", tt.in, got, tt.want)
		}
	}

	// Refresh and Logout exist on both auth services, neither of them
	// client-only, so the bare name stays ambiguous.
	for _, in := range []string{"Refresh", "NoSuchMethod"} {
		if _, err := findMethod(in); err == nil {
			t.Errorf("findMethod(%q) succeeded, want error", in)
		}
	}
}

func TestMethodAccess(t *testing.T) {
	tests := map[string]Access{
		"UserAuthService/Login":    AccessNone,
		"AuthService/Ping":         AccessNone,
		"UserService/GetDevices":   AccessUser,
		"SystemService/Reboot":     AccessUser,
		"ClientService/SendAction": AccessClient,
	}
	for name, want := range tests {
		md, err := findMethod(name)
		if err != nil {
			t.Fatal(err)
		}
		if got := methodAccess(md); got != want {
			t.Errorf("methodAccess(%s) = %s, want %s", name, got, want)
		}
	}
}

func TestIsHeartbeat(t *testing.T) {
	tests := []struct {
		name string
		msg  proto.Message
		want bool
	}{
		{"empty legacy", &clientsapi.GroupsStreamResponse{}, true},
		{"legacy update", &clientsapi.GroupsStreamResponse{RemovedId: "g1"}, false},
		{"oneof heartbeat", &clientsapi.ZonesStreamResponse{Update: &clientsapi.ZonesStreamResponse_Heartbeat{Heartbeat: &clientsapi.Heartbeat{}}}, true},
		{"oneof update", &clientsapi.ZonesStreamResponse{Update: &clientsapi.ZonesStreamResponse_RemovedId{RemovedId: "z1"}}, false},
	}
	for _, tt := range tests {
		if got := isHeartbeat(tt.msg); got != tt.want {
			t.Errorf("%s: isHeartbeat = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestPrinterIsStable(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, false, false)
	msg := &clientsapi.AddZoneRequest{Name: "Kitchen", DeviceIds: []string{"a", "b"}}
	if err := p.Print(msg); err != nil {
		t.Fatal(err)
	}
	want := `{"name":"Kitchen","device_ids":["a","b"]}` + "\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestParseRequest(t *testing.T) {
	md, err := findMethod("UserService/AddZone")
	if err != nil {
		t.Fatal(err)
	}
	// Both proto and JSON (lowerCamel) names are accepted.
	req, err := parseRequest(md, []byte(`{"name":"Hall","deviceIds":["x"]}`))
	if err != nil {
		t.Fatal(err)
	}
	got := req.(*clientsapi.AddZoneRequest)
	if got.Name != "Hall" || !slices.Equal(got.DeviceIds, []string{"x"}) {
		t.Errorf("parsed %v", got)
	}
	if _, err := parseRequest(md, []byte(`{"nope":1}`)); err == nil {
		t.Error("unknown field accepted")
	}
	if _, err := parseRequest(md, []byte("  ")); err != nil {
		t.Errorf("empty request: %v", err)
	}
}

func TestPermuteArgs(t *testing.T) {
	app := newApp()
	tests := []struct {
		in, want []string
	}{
		{
			[]string{"whctl", "call", "GetDevices", "--count", "3", "--pretty"},
			[]string{"whctl", "call", "--count", "3", "--pretty", "GetDevices"},
		},
		{
			[]string{"whctl", "-p", "dev", "call", "AddZone", "{\"name\":\"x\"}", "-n=2"},
			[]string{"whctl", "-p", "dev", "call", "-n=2", "AddZone", "{\"name\":\"x\"}"},
		},
		{
			[]string{"whctl", "action", "dev1", "1", "light", "--set", "on=true", "--set", "level=50"},
			[]string{"whctl", "action", "--set", "on=true", "--set", "level=50", "dev1", "1", "light"},
		},
		{
			[]string{"whctl", "call", "AddZone", "-", "--", "--literal"},
			[]string{"whctl", "call", "AddZone", "-", "--", "--literal"},
		},
	}
	for _, tt := range tests {
		if got := permuteArgs(app, tt.in); !slices.Equal(got, tt.want) {
			t.Errorf("permuteArgs(%q)\n got %q\nwant %q", tt.in, got, tt.want)
		}
	}
}

func TestParseValue(t *testing.T) {
	boolSpec := &clientsapi.Spec{Kind: &clientsapi.Spec_Bool{Bool: &clientsapi.BoolSpec{}}}
	intSpec := &clientsapi.Spec{Kind: &clientsapi.Spec_Int{Int: &clientsapi.IntSpec{}}}
	durSpec := &clientsapi.Spec{Kind: &clientsapi.Spec_Duration{Duration: &clientsapi.DurationSpec{}}}
	colorSpec := &clientsapi.Spec{Kind: &clientsapi.Spec_Color{Color: &clientsapi.ColorSpec{}}}

	tests := []struct {
		spec *clientsapi.Spec
		in   string
		want *clientsapi.Value
	}{
		{boolSpec, "true", &clientsapi.Value{Kind: &clientsapi.Value_Bool{Bool: true}}},
		{intSpec, "-5", &clientsapi.Value{Kind: &clientsapi.Value_Int{Int: -5}}},
		{durSpec, "1m30s", &clientsapi.Value{Kind: &clientsapi.Value_Duration{Duration: 90000}}},
		{durSpec, "250", &clientsapi.Value{Kind: &clientsapi.Value_Duration{Duration: 250}}},
		{nil, `{"text":"hi"}`, &clientsapi.Value{Kind: &clientsapi.Value_Text{Text: "hi"}}},
		{colorSpec, `{"color":{"hue_sat":{"hue":120,"sat":100}}}`, &clientsapi.Value{Kind: &clientsapi.Value_Color{Color: &clientsapi.ColorValue{HueSat: &clientsapi.ColorHueSat{Hue: 120, Sat: 100}}}}},
	}
	for _, tt := range tests {
		got, err := parseValue(tt.spec, tt.in)
		if err != nil {
			t.Errorf("parseValue(%q): %v", tt.in, err)
			continue
		}
		if !proto.Equal(got, tt.want) {
			t.Errorf("parseValue(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}

	for _, tc := range []struct {
		spec *clientsapi.Spec
		in   string
	}{
		{boolSpec, "maybe"},
		{intSpec, "1.5"},
		{colorSpec, "red"},
		{nil, "true"},
	} {
		if _, err := parseValue(tc.spec, tc.in); err == nil {
			t.Errorf("parseValue(%q) succeeded, want error", tc.in)
		}
	}
}

func TestNormalise(t *testing.T) {
	for in, want := range map[string]string{
		"example-host":       "example-host:4000",
		"example-host:4100":  "example-host:4100",
		"192.0.2.1":          "192.0.2.1:4000",
		"[2001:db8::1]":      "[2001:db8::1]:4000",
		"[2001:db8::1]:4100": "[2001:db8::1]:4100",
	} {
		if got := normaliseServer(in); got != want {
			t.Errorf("normaliseServer(%q) = %q, want %q", in, got, want)
		}
	}
	if got := normaliseFingerprint(" AB:cd:EF "); got != "abcdef" {
		t.Errorf("normaliseFingerprint = %q", got)
	}
}

func TestConfigRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.json")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.ProfileName(""); got != defaultProfile {
		t.Errorf("ProfileName = %q", got)
	}
	cfg.Profiles["lab"] = &Profile{Server: "example-host:4000", Username: "alice"}
	cfg.Current = "lab"
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	again, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if again.ProfileName("") != "lab" || again.Profiles["lab"].Username != "alice" {
		t.Errorf("round trip lost data: %+v", again)
	}
	if _, err := again.Profile("missing"); err == nil {
		t.Error("missing profile found")
	}
}

func TestTokenExpiry(t *testing.T) {
	// {"alg":"none"}.{"exp":2000000000}. with no signature.
	token := "eyJhbGciOiJub25lIn0.eyJleHAiOjIwMDAwMDAwMDB9."
	exp, ok := tokenExpiry(token)
	if !ok || !exp.Equal(time.Unix(2000000000, 0)) {
		t.Errorf("tokenExpiry = %v, %v", exp, ok)
	}
	if _, ok := tokenExpiry("garbage"); ok {
		t.Error("garbage token parsed")
	}
}
