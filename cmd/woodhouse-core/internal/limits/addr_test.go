package limits

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc/peer"
)

func TestPeerAddr_FromTCPAddr(t *testing.T) {
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 12345},
	})

	addr := PeerAddr(ctx)
	if !addr.IsValid() {
		t.Fatalf("expected a valid addr")
	}
	if addr.String() != "192.0.2.1" {
		t.Fatalf("expected 192.0.2.1, got %s", addr.String())
	}
}

func TestPeerAddr_NoPeerReturnsZero(t *testing.T) {
	addr := PeerAddr(context.Background())
	if addr.IsValid() {
		t.Fatalf("expected a zero addr for a context with no peer, got %s", addr.String())
	}
}
