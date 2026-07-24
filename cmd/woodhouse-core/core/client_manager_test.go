package core

import (
	"errors"
	"net/netip"
	"testing"

	"github.com/jimjibone/woodhouse-core/shared/stores"
)

func newTestClientManager(t *testing.T) *ClientManager {
	t.Helper()
	store := stores.NewMemStore()
	manager, err := NewClientManager(store)
	if err != nil {
		t.Fatalf("failed to create client manager: %s", err)
	}
	t.Cleanup(manager.Close)
	return manager
}

func mustAddr(t *testing.T, s string) netip.Addr {
	t.Helper()
	addr, err := netip.ParseAddr(s)
	if err != nil {
		t.Fatalf("ParseAddr(%q): %v", s, err)
	}
	return addr
}

func TestAddPairingRequest_PerSourceCap(t *testing.T) {
	manager := newTestClientManager(t)
	addr := mustAddr(t, "10.1.2.3")

	var requestIDs []string
	for i := 0; i < maxPendingPairingsPerSource; i++ {
		req := &PairingRequest{ClientID: "client-" + string(rune('a'+i))}
		requestID, _, err := manager.AddPairingRequest(req, addr)
		if err != nil {
			t.Fatalf("request %d: unexpected error: %s", i, err)
		}
		requestIDs = append(requestIDs, requestID)
	}

	// The next request from the same address should be rejected, even
	// though the global cap (maxPendingPairings) is nowhere near reached.
	_, _, err := manager.AddPairingRequest(&PairingRequest{ClientID: "one-too-many"}, addr)
	if !errors.Is(err, ErrTooManyPairingsFromSource) {
		t.Fatalf("expected ErrTooManyPairingsFromSource, got %v", err)
	}

	// A different source address is unaffected.
	otherAddr := mustAddr(t, "10.1.2.4")
	_, _, err = manager.AddPairingRequest(&PairingRequest{ClientID: "different-source"}, otherAddr)
	if err != nil {
		t.Fatalf("expected a different source address to still be allowed, got %s", err)
	}

	// Removing one pending request from the capped address frees a slot.
	if err := manager.RemovePairingRequest(requestIDs[0]); err != nil {
		t.Fatalf("failed to remove pairing request: %s", err)
	}
	_, _, err = manager.AddPairingRequest(&PairingRequest{ClientID: "freed-slot"}, addr)
	if err != nil {
		t.Fatalf("expected a freed slot to allow a new request, got %s", err)
	}
}

func TestAddPairingRequest_ZeroAddrSharesCap(t *testing.T) {
	manager := newTestClientManager(t)
	var zero netip.Addr

	for i := 0; i < maxPendingPairingsPerSource; i++ {
		req := &PairingRequest{ClientID: "zero-client-" + string(rune('a'+i))}
		if _, _, err := manager.AddPairingRequest(req, zero); err != nil {
			t.Fatalf("request %d: unexpected error: %s", i, err)
		}
	}

	// A second, distinct zero-value addr still shares the same cap.
	var zero2 netip.Addr
	_, _, err := manager.AddPairingRequest(&PairingRequest{ClientID: "one-too-many"}, zero2)
	if !errors.Is(err, ErrTooManyPairingsFromSource) {
		t.Fatalf("expected ErrTooManyPairingsFromSource for shared zero-addr cap, got %v", err)
	}
}
