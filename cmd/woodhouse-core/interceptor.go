package main

import (
	"context"
	"strings"

	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/clients"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/users"
	"github.com/jimjibone/woodhouse-4/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthInterceptor struct {
	log     *log.Context
	clients *clients.JWTManager
	users   *users.JWTManager
}

func NewAuthInterceptor(clients *clients.JWTManager, users *users.JWTManager) *AuthInterceptor {
	interceptor := &AuthInterceptor{
		log:     log.NewContext(log.DefaultLogger, "auth-interceptor", log.DebugLevel),
		clients: clients,
		users:   users,
	}

	return interceptor
}

// Unary returns a server interceptor function to authenticate and authorize unary RPC
func (interceptor *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if auth.RequiresAuth(info.FullMethod) {
			clientID, ctx2, err := interceptor.authorize(ctx, info.FullMethod)
			if err != nil {
				interceptor.log.Warnf("--> unary %s: %q not authorized: %s", info.FullMethod, clientID, err)
				return nil, err
			}
			interceptor.log.Infof("--> unary %s: %q authorized", info.FullMethod, clientID)
			return handler(ctx2, req)
		}

		if info.FullMethod != "/woodhouse.api.v1.clients.AuthService/Ping" {
			interceptor.log.Infof("--> unary %s: no auth required", info.FullMethod)
		}
		return handler(ctx, req)
	}
}

// Stream returns a server interceptor function to authenticate and authorize stream RPC
func (interceptor *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if auth.RequiresAuth(info.FullMethod) {
			clientID, ctx2, err := interceptor.authorize(stream.Context(), info.FullMethod)
			if err != nil {
				interceptor.log.Warnf("--> stream %s: %q not authorized: %s", info.FullMethod, clientID, err)
				return err
			}
			interceptor.log.Infof("--> stream %s: %q authorized", info.FullMethod, clientID)
			return handler(srv, newWrappedStream(stream, ctx2))
		}

		if info.FullMethod != "/woodhouse.api.v1.clients.AuthService/Ping" {
			interceptor.log.Infof("--> stream %s: no auth required", info.FullMethod)
		}
		return handler(srv, stream)
	}
}

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func newWrappedStream(s grpc.ServerStream, ctx context.Context) grpc.ServerStream {
	return &wrappedStream{s, ctx}
}

func (stream *wrappedStream) Context() context.Context {
	return stream.ctx
}

// authorize extracts the authorization token from the request context and
// checks that the access token is still authorized. Returns the client id,
// context containing the access token claims, or possibly an error.
func (interceptor *AuthInterceptor) authorize(ctx context.Context, method string) (string, context.Context, error) {
	// gRPC-Web requests don't have a leading `/`.
	if !strings.HasPrefix(method, "/") {
		method = "/" + method
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", nil, status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	values := md["authorization"]
	if len(values) == 0 || len(values[0]) == 0 {
		return "", nil, status.Errorf(codes.Unauthenticated, "authorization token is not provided")
	}

	if auth.IsUserMethod(method) {
		accessToken := values[0]
		claims, err := interceptor.users.VerifyAccessToken(accessToken)
		if err != nil {
			return "", nil, status.Errorf(codes.Unauthenticated, "client access token is invalid: %v", err)
		}

		if auth.IsUserAuthorised(method, claims.Role) {
			return claims.Username, context.WithValue(ctx, "claims", claims), nil
		}

		return claims.Username, nil, status.Error(codes.PermissionDenied, "no permission to access this RPC")
	}

	// Must be a client.
	accessToken := values[0]
	claims, err := interceptor.clients.VerifyAccessToken(accessToken)
	if err != nil {
		return "", nil, status.Errorf(codes.Unauthenticated, "client access token is invalid: %v", err)
	}

	return claims.ClientID, context.WithValue(ctx, "claims", claims), nil
}
