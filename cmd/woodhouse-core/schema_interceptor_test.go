package main

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/jimjibone/woodhouse-api/schema"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const testSchemaVersion = 7

func schemaContext(versions ...string) context.Context {
	md := metadata.MD{}
	for _, version := range versions {
		md.Append(schema.HeaderName, version)
	}
	return metadata.NewIncomingContext(context.Background(), md)
}

type schemaTestStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s schemaTestStream) Context() context.Context { return s.ctx }

func TestSchemaVersionInterceptor(t *testing.T) {
	interceptor := NewSchemaVersionInterceptor(testSchemaVersion)

	tests := []struct {
		name     string
		ctx      context.Context
		method   string
		wantCode codes.Code
		wantMsg  string
	}{
		{name: "matching version", ctx: schemaContext("7"), method: testUserMethod},
		{name: "matching version for a client method", ctx: schemaContext("7"), method: testClientMethod},
		{name: "matching version without leading slash", ctx: schemaContext("7"), method: strings.TrimPrefix(testUserMethod, "/")},
		{
			name:     "older version",
			ctx:      schemaContext("6"),
			method:   testUserMethod,
			wantCode: codes.FailedPrecondition,
			wantMsg:  "client schema version 6, server 7: upgrade the client",
		},
		{
			name:     "newer version",
			ctx:      schemaContext("8"),
			method:   testClientMethod,
			wantCode: codes.FailedPrecondition,
			wantMsg:  "client schema version 8, server 7: upgrade the client",
		},
		{
			name:     "not a number",
			ctx:      schemaContext("seven"),
			method:   testUserMethod,
			wantCode: codes.FailedPrecondition,
			wantMsg:  "client schema version seven, server 7",
		},
		{
			name:     "missing header",
			ctx:      schemaContext(),
			method:   testUserMethod,
			wantCode: codes.FailedPrecondition,
			wantMsg:  "client schema version missing, server 7: upgrade the client",
		},
		{
			name:     "no metadata at all",
			ctx:      context.Background(),
			method:   testClientMethod,
			wantCode: codes.FailedPrecondition,
			wantMsg:  "client schema version missing, server 7: upgrade the client",
		},
		{
			name:     "a stale value among matching ones",
			ctx:      schemaContext("7", "6"),
			method:   testUserMethod,
			wantCode: codes.FailedPrecondition,
			wantMsg:  "client schema version 6",
		},
		{
			name:     "calls that need no auth are checked too",
			ctx:      schemaContext("6"),
			method:   "/woodhouse.api.v2.clients.AuthService/Pair",
			wantCode: codes.FailedPrecondition,
			wantMsg:  "client schema version 6",
		},
		{name: "health is exempt", ctx: schemaContext(), method: "/grpc.health.v1.Health/Check"},
		{name: "reflection is exempt", ctx: schemaContext("6"), method: "/grpc.reflection.v1.ServerReflection/ServerReflectionInfo"},
		{
			name:     "a lookalike service is not exempt",
			ctx:      schemaContext(),
			method:   "/grpc.health.v1.HealthExtra/Check",
			wantCode: codes.FailedPrecondition,
			wantMsg:  "missing",
		},
	}

	check := func(t *testing.T, err error, called bool, wantCode codes.Code, wantMsg string) {
		t.Helper()
		if got := status.Code(err); got != wantCode {
			t.Fatalf("code = %s, want %s (err: %v)", got, wantCode, err)
		}
		if wantCode == codes.OK {
			if !called {
				t.Errorf("handler not called")
			}
			return
		}
		if called {
			t.Errorf("handler called for a rejected call")
		}
		if !strings.Contains(status.Convert(err).Message(), wantMsg) {
			t.Errorf("message = %q, want it to contain %q", status.Convert(err).Message(), wantMsg)
		}
	}

	for _, test := range tests {
		t.Run("unary "+test.name, func(t *testing.T) {
			called := false
			_, err := interceptor.Unary()(test.ctx, nil, &grpc.UnaryServerInfo{FullMethod: test.method},
				func(ctx context.Context, req interface{}) (interface{}, error) {
					called = true
					return nil, nil
				})
			check(t, err, called, test.wantCode, test.wantMsg)
		})
		t.Run("stream "+test.name, func(t *testing.T) {
			called := false
			err := interceptor.Stream()(nil, schemaTestStream{ctx: test.ctx}, &grpc.StreamServerInfo{FullMethod: test.method},
				func(srv interface{}, stream grpc.ServerStream) error {
					called = true
					return nil
				})
			check(t, err, called, test.wantCode, test.wantMsg)
		})
	}
}

// The server's version is the schema package's, and a client sending that
// version is let through.
func TestSchemaVersionInterceptorUsesSchemaVersion(t *testing.T) {
	version := schema.MustLoad().Version
	interceptor := NewSchemaVersionInterceptor(version)

	ctx := schemaContext(strconv.Itoa(version))
	if err := interceptor.check(ctx, testUserMethod); err != nil {
		t.Errorf("current version rejected: %s", err)
	}
	ctx = schemaContext(strconv.Itoa(version + 1))
	if err := interceptor.check(ctx, testUserMethod); status.Code(err) != codes.FailedPrecondition {
		t.Errorf("next version: err = %v, want FailedPrecondition", err)
	}
}

// The version check runs before authentication, so a stale peer is told to
// upgrade rather than that it is unauthenticated.
func TestSchemaVersionCheckedBeforeAuth(t *testing.T) {
	authInterceptor, _, _, _, _ := newTestInterceptor(t)
	schemaInterceptor := NewSchemaVersionInterceptor(testSchemaVersion)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) { return nil, nil }
	info := &grpc.UnaryServerInfo{FullMethod: testUserMethod}

	// Stale version, no token.
	_, err := schemaInterceptor.Unary()(schemaContext("6"), nil, info,
		func(ctx context.Context, req interface{}) (interface{}, error) {
			return authInterceptor.Unary()(ctx, req, info, handler)
		})
	if status.Code(err) != codes.FailedPrecondition {
		t.Errorf("stale version: code = %s, want FailedPrecondition", status.Code(err))
	}

	// Current version, no token: past the version check, stopped by auth.
	_, err = schemaInterceptor.Unary()(schemaContext("7"), nil, info,
		func(ctx context.Context, req interface{}) (interface{}, error) {
			return authInterceptor.Unary()(ctx, req, info, handler)
		})
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("current version: code = %s, want Unauthenticated", status.Code(err))
	}
}
