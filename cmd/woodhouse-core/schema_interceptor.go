package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/jimjibone/woodhouse-api/schema"
)

// schemaExemptPrefixes are services that tooling talks to without being built
// against the Woodhouse schema.
var schemaExemptPrefixes = []string{
	"/grpc.health.v1.Health/",
	"/grpc.reflection.v1.ServerReflection/",
	"/grpc.reflection.v1alpha.ServerReflection/",
}

// SchemaVersionInterceptor rejects calls from clients built against another
// version of the service schema, so a stale bridge, reactor or UI fails loudly
// instead of half working. Every call must carry the schema.HeaderName
// metadata key set to the server's schema version.
type SchemaVersionInterceptor struct {
	version string
}

func NewSchemaVersionInterceptor(version int) *SchemaVersionInterceptor {
	return &SchemaVersionInterceptor{version: strconv.Itoa(version)}
}

// Unary returns a server interceptor function to check the schema version of
// unary RPCs.
func (interceptor *SchemaVersionInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if err := interceptor.check(ctx, info.FullMethod); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

// Stream returns a server interceptor function to check the schema version of
// stream RPCs.
func (interceptor *SchemaVersionInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if err := interceptor.check(stream.Context(), info.FullMethod); err != nil {
			return err
		}
		return handler(srv, stream)
	}
}

func (interceptor *SchemaVersionInterceptor) check(ctx context.Context, method string) error {
	// gRPC-Web requests don't have a leading `/`.
	if !strings.HasPrefix(method, "/") {
		method = "/" + method
	}
	for _, prefix := range schemaExemptPrefixes {
		if strings.HasPrefix(method, prefix) {
			return nil
		}
	}

	var versions []string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		versions = md.Get(schema.HeaderName)
	}
	if len(versions) == 0 {
		return interceptor.reject("missing")
	}
	for _, version := range versions {
		if version != interceptor.version {
			return interceptor.reject(version)
		}
	}
	return nil
}

func (interceptor *SchemaVersionInterceptor) reject(clientVersion string) error {
	// The header is peer-controlled, so keep what is echoed back short.
	if len(clientVersion) > 32 {
		clientVersion = clientVersion[:32] + "..."
	}
	return status.Error(codes.FailedPrecondition,
		fmt.Sprintf("client schema version %s, server %s: upgrade the client", clientVersion, interceptor.version))
}
