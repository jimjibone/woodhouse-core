package limits

import (
	"net/netip"
	"testing"
	"time"
)

func TestIPRate_BurstThenRefill(t *testing.T) {
	clock := newFakeClock()
	r := NewIPRate(IPRateConfig{PerMin: 60, Burst: 5, Now: clock.Now})
	ip := mustAddr(t, "10.0.1.1")

	for i := 0; i < 5; i++ {
		ok, rejections := r.Allow(ip)
		if !ok {
			t.Fatalf("attempt %d: expected allow within burst", i)
		}
		if rejections != 0 {
			t.Fatalf("attempt %d: expected rejections reset to 0, got %d", i, rejections)
		}
	}

	// Burst exhausted; next attempt should be denied.
	ok, rejections := r.Allow(ip)
	if ok {
		t.Fatalf("expected deny after burst exhausted")
	}
	if rejections != 1 {
		t.Fatalf("expected rejections=1, got %d", rejections)
	}

	// Advance the clock by 1s (1/s sustained rate) and expect exactly one
	// token refilled.
	clock.Advance(time.Second)
	ok, rejections = r.Allow(ip)
	if !ok {
		t.Fatalf("expected allow after 1s refill")
	}
	if rejections != 0 {
		t.Fatalf("expected rejections reset to 0 after allow, got %d", rejections)
	}
	ok, _ = r.Allow(ip)
	if ok {
		t.Fatalf("expected deny again, only one token should have refilled")
	}
}

func TestIPRate_PenaliseDrainsTokens(t *testing.T) {
	clock := newFakeClock()
	r := NewIPRate(IPRateConfig{PerMin: 60, Burst: 5, Penalty: 3, Now: clock.Now})
	ip := mustAddr(t, "10.0.1.2")

	// Penalise once, burning 3 of the 5 tokens; 2 attempts should still
	// succeed, the 3rd should be denied.
	r.Penalise(ip)

	ok, _ := r.Allow(ip)
	if !ok {
		t.Fatalf("expected allow, 2 tokens should remain after penalty")
	}
	ok, _ = r.Allow(ip)
	if !ok {
		t.Fatalf("expected allow, 1 token should remain after penalty")
	}
	ok, _ = r.Allow(ip)
	if ok {
		t.Fatalf("expected deny, tokens should be exhausted after penalty + 2 allows")
	}
}

func TestIPRate_ZeroAddrSharesBucket(t *testing.T) {
	clock := newFakeClock()
	r := NewIPRate(IPRateConfig{PerMin: 60, Burst: 3, Now: clock.Now})

	var zero netip.Addr
	for i := 0; i < 3; i++ {
		ok, _ := r.Allow(zero)
		if !ok {
			t.Fatalf("attempt %d: expected allow within burst for zero addr", i)
		}
	}
	ok, _ := r.Allow(zero)
	if ok {
		t.Fatalf("expected deny after burst exhausted for zero addr")
	}

	// A second, distinct invalid addr (also zero-value) shares the bucket.
	var zero2 netip.Addr
	ok, _ = r.Allow(zero2)
	if ok {
		t.Fatalf("expected zero-value addrs to share one bucket")
	}
}

func TestIPRate_RejectionsResetOnAllow(t *testing.T) {
	clock := newFakeClock()
	r := NewIPRate(IPRateConfig{PerMin: 60, Burst: 1, Now: clock.Now})
	ip := mustAddr(t, "10.0.1.3")

	ok, rejections := r.Allow(ip)
	if !ok || rejections != 0 {
		t.Fatalf("first attempt: expected allow with rejections=0, got ok=%v rejections=%d", ok, rejections)
	}

	// Consecutive denials increment the counter.
	ok, rejections = r.Allow(ip)
	if ok || rejections != 1 {
		t.Fatalf("expected deny with rejections=1, got ok=%v rejections=%d", ok, rejections)
	}
	ok, rejections = r.Allow(ip)
	if ok || rejections != 2 {
		t.Fatalf("expected deny with rejections=2, got ok=%v rejections=%d", ok, rejections)
	}

	// Advance past refill and allow again: rejections resets to 0.
	clock.Advance(time.Second)
	ok, rejections = r.Allow(ip)
	if !ok || rejections != 0 {
		t.Fatalf("expected allow with rejections reset to 0, got ok=%v rejections=%d", ok, rejections)
	}
}

func TestIPRate_LRUEviction(t *testing.T) {
	clock := newFakeClock()
	r := NewIPRate(IPRateConfig{PerMin: 60, Burst: 2, CacheSize: 2, Now: clock.Now})

	ip1 := mustAddr(t, "10.0.1.10")
	ip2 := mustAddr(t, "10.0.1.11")
	ip3 := mustAddr(t, "10.0.1.12")

	// Drain ip1's burst.
	r.Allow(ip1)
	r.Allow(ip1)
	if ok, _ := r.Allow(ip1); ok {
		t.Fatalf("expected ip1 burst drained")
	}

	// Touch ip2, then add ip3: with cache size 2, ip1 (least recently
	// used) should be evicted.
	r.Allow(ip2)
	r.Allow(ip3)

	// ip1 should now come back with a fresh, full burst (documented
	// eviction behavior).
	ok, _ := r.Allow(ip1)
	if !ok {
		t.Fatalf("expected ip1 to have a fresh bucket after eviction")
	}
	ok, _ = r.Allow(ip1)
	if !ok {
		t.Fatalf("expected ip1's fresh bucket to have full burst (2 tokens)")
	}
}
