package clients

import (
	"context"
	"net"
	"testing"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/limits"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// ctxWithIP returns a context carrying a gRPC peer with the given IP, as
// limits.PeerAddr expects to find via peer.FromContext.
func ctxWithIP(ip string) context.Context {
	return peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP(ip), Port: 12345},
	})
}

// Note: this only exercises Ping's rate gating directly. A full bidi Pair
// stream test is not included here; Pair's gating logic (pairLimits.Allow /
// Penalise) is covered indirectly by the internal/limits tests, and the
// per-source pending-pairing cap it calls into is covered by
// core/client_manager_test.go.
func TestPing_RateLimited(t *testing.T) {
	pingLimits := limits.NewIPRate(limits.IPRateConfig{PerMin: 60, Burst: 2})
	srv := NewAuthService(nil, nil, nil, nil, pingLimits)
	ctx := ctxWithIP("10.2.0.1")

	for i := 0; i < 2; i++ {
		if _, err := srv.Ping(ctx, &clientsapi.PingRequest{}); err != nil {
			t.Fatalf("ping %d: unexpected error: %s", i, err)
		}
	}

	_, err := srv.Ping(ctx, &clientsapi.PingRequest{})
	if err == nil {
		t.Fatalf("expected the 3rd ping to be rate limited")
	}
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("expected codes.ResourceExhausted, got %s", status.Code(err))
	}

	// A different source address is unaffected.
	otherCtx := ctxWithIP("10.2.0.2")
	if _, err := srv.Ping(otherCtx, &clientsapi.PingRequest{}); err != nil {
		t.Fatalf("expected a different source address to still be allowed, got %s", err)
	}
}
