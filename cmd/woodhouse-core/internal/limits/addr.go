package limits

import (
	"context"
	"net"
	"net/http"
	"net/netip"

	"google.golang.org/grpc/peer"
)

// PeerAddr extracts the source IP of a gRPC call from its context, for
// use as a rate-limits bucket key. It covers both native gRPC and
// grpc-web, since grpcweb.WrapServer forwards to grpc.Server.ServeHTTP,
// which populates the peer from the underlying http.Request.RemoteAddr.
//
// A zero netip.Addr is returned when the peer address can't be
// determined; callers should treat that as sharing a single bucket
// rather than bypassing the limit, never as a free pass.
//
// No proxy headers (X-Forwarded-For etc.) are consulted: they're
// trivially spoofable by any client that can reach this endpoint
// directly, which is the only way to reach it today.
func PeerAddr(ctx context.Context) netip.Addr {
	p, ok := peer.FromContext(ctx)
	if !ok || p.Addr == nil {
		return netip.Addr{}
	}

	// Fast path: the common case for a real network peer.
	if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
		addr, ok := netip.AddrFromSlice(tcpAddr.IP)
		if !ok {
			return netip.Addr{}
		}
		return addr.Unmap()
	}

	// Fallback: parse whatever string representation we got.
	host, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return netip.Addr{}
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}
	return addr.Unmap()
}

// RequestAddr extracts the source IP of an HTTP request, for use as a
// rate-limits bucket key. See PeerAddr's doc comment for the zero-Addr
// and proxy-header caveats, which apply here too.
func RequestAddr(r *http.Request) netip.Addr {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return netip.Addr{}
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}
	return addr.Unmap()
}
