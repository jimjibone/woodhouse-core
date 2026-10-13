package system

import (
	"context"
	"net"
	"testing"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	localapi "github.com/jimjibone/woodhouse-api/go/v1/local"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type fakeAgent struct {
	localapi.UnimplementedAgentServiceServer
	resetCalls int
}

func (f *fakeAgent) GetSystemInfo(ctx context.Context, req *clientsapi.GetSystemInfoRequest) (*clientsapi.GetSystemInfoResponse, error) {
	return &clientsapi.GetSystemInfoResponse{OsVersion: "1.2.3", Board: "rpi4"}, nil
}

func (f *fakeAgent) FactoryReset(ctx context.Context, req *clientsapi.FactoryResetRequest) (*clientsapi.FactoryResetResponse, error) {
	if !req.Confirm {
		return nil, status.Error(codes.InvalidArgument, "confirm must be true")
	}
	f.resetCalls++
	return &clientsapi.FactoryResetResponse{}, nil
}

func newAgentClient(t *testing.T, agent *fakeAgent) localapi.AgentServiceClient {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	localapi.RegisterAgentServiceServer(srv, agent)
	go srv.Serve(lis) //nolint:errcheck
	t.Cleanup(srv.Stop)
	conn, err := grpc.NewClient("passthrough:///agent",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return localapi.NewAgentServiceClient(conn)
}

func TestForwardsToAgent(t *testing.T) {
	agent := &fakeAgent{}
	s := New(newAgentClient(t, agent))
	ctx := context.Background()

	info, err := s.GetSystemInfo(ctx, &clientsapi.GetSystemInfoRequest{})
	if err != nil || info.OsVersion != "1.2.3" || info.Board != "rpi4" {
		t.Fatalf("info %+v err %v", info, err)
	}

	// Agent errors keep their code.
	if _, err := s.FactoryReset(ctx, &clientsapi.FactoryResetRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
	if _, err := s.FactoryReset(ctx, &clientsapi.FactoryResetRequest{Confirm: true}); err != nil || agent.resetCalls != 1 {
		t.Fatalf("reset err %v calls %d", err, agent.resetCalls)
	}

	// Methods the agent does not implement surface as UNIMPLEMENTED.
	if _, err := s.Reboot(ctx, &clientsapi.RebootRequest{}); status.Code(err) != codes.Unimplemented {
		t.Fatalf("expected Unimplemented, got %v", err)
	}
}

func TestWithoutAgent(t *testing.T) {
	s := New(nil)
	if _, err := s.GetSystemInfo(context.Background(), &clientsapi.GetSystemInfoRequest{}); status.Code(err) != codes.Unavailable {
		t.Fatalf("expected Unavailable, got %v", err)
	}
}
