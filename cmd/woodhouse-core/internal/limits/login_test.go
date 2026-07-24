package limits

import (
	"net/netip"
	"testing"
	"time"
)

// fakeClock lets tests advance time deterministically, without sleeping.
type fakeClock struct {
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
}

func mustAddr(t *testing.T, s string) netip.Addr {
	t.Helper()
	addr, err := netip.ParseAddr(s)
	if err != nil {
		t.Fatalf("ParseAddr(%q): %v", s, err)
	}
	return addr
}

func TestAllowIP_BurstThenRefill(t *testing.T) {
	clock := newFakeClock()
	l := NewLogin(Config{LoginBurst: 10, LoginPerMin: 60, Now: clock.Now})
	ip := mustAddr(t, "10.0.0.1")

	for i := 0; i < 10; i++ {
		ok, rejections := l.AllowIP(ip)
		if !ok {
			t.Fatalf("attempt %d: expected allow within burst", i)
		}
		if rejections != 0 {
			t.Fatalf("attempt %d: expected rejections reset to 0, got %d", i, rejections)
		}
	}

	// Burst exhausted; next attempt should be denied.
	ok, rejections := l.AllowIP(ip)
	if ok {
		t.Fatalf("expected deny after burst exhausted")
	}
	if rejections != 1 {
		t.Fatalf("expected rejections=1, got %d", rejections)
	}

	// Still denied immediately after.
	ok, rejections = l.AllowIP(ip)
	if ok {
		t.Fatalf("expected deny before refill")
	}
	if rejections != 2 {
		t.Fatalf("expected rejections=2, got %d", rejections)
	}

	// Advance the clock by 1s (1/s sustained rate) and expect exactly one
	// token refilled.
	clock.Advance(time.Second)
	ok, rejections = l.AllowIP(ip)
	if !ok {
		t.Fatalf("expected allow after 1s refill")
	}
	if rejections != 0 {
		t.Fatalf("expected rejections reset to 0 after allow, got %d", rejections)
	}
	ok, _ = l.AllowIP(ip)
	if ok {
		t.Fatalf("expected deny again, only one token should have refilled")
	}
}

func TestRecordFailure_DrainsPenaltyTokens(t *testing.T) {
	clock := newFakeClock()
	l := NewLogin(Config{LoginBurst: 10, LoginPerMin: 60, FailPenalty: 3, Now: clock.Now})
	ip := mustAddr(t, "10.0.0.2")

	// Each failure burns 3 tokens. Burst is 10, so 3 failures leave 1
	// token (10 - 3*3); one more AllowIP call spends it, and the next
	// is denied.
	l.RecordFailure(ip, "alice")
	l.RecordFailure(ip, "alice")
	l.RecordFailure(ip, "alice")

	ok, _ := l.AllowIP(ip)
	if !ok {
		t.Fatalf("expected the last remaining token to still allow one more attempt")
	}
	ok, _ = l.AllowIP(ip)
	if ok {
		t.Fatalf("expected burst drained after 3 failures (9 tokens) + 1 allowed attempt")
	}
}

func TestAllowIP_ZeroAddrSharesBucket(t *testing.T) {
	clock := newFakeClock()
	l := NewLogin(Config{LoginBurst: 3, LoginPerMin: 60, Now: clock.Now})

	var zero netip.Addr
	for i := 0; i < 3; i++ {
		ok, _ := l.AllowIP(zero)
		if !ok {
			t.Fatalf("attempt %d: expected allow within burst for zero addr", i)
		}
	}
	ok, _ := l.AllowIP(zero)
	if ok {
		t.Fatalf("expected deny after burst exhausted for zero addr")
	}

	// A second, distinct invalid addr (also zero-value) shares the bucket.
	var zero2 netip.Addr
	ok, _ = l.AllowIP(zero2)
	if ok {
		t.Fatalf("expected zero-value addrs to share one bucket")
	}
}

func TestIPLimiters_LRUEviction(t *testing.T) {
	clock := newFakeClock()
	l := NewLogin(Config{LoginBurst: 2, LoginPerMin: 60, IPCacheSize: 2, Now: clock.Now})

	ip1 := mustAddr(t, "10.0.0.1")
	ip2 := mustAddr(t, "10.0.0.2")
	ip3 := mustAddr(t, "10.0.0.3")

	// Drain ip1's burst.
	l.AllowIP(ip1)
	l.AllowIP(ip1)
	if ok, _ := l.AllowIP(ip1); ok {
		t.Fatalf("expected ip1 burst drained")
	}

	// Touch ip2, then add ip3: with cache size 2, ip1 (least recently
	// used) should be evicted.
	l.AllowIP(ip2)
	l.AllowIP(ip3)

	// ip1 should now come back with a fresh, full burst (documented
	// eviction behavior).
	ok, _ := l.AllowIP(ip1)
	if !ok {
		t.Fatalf("expected ip1 to have a fresh bucket after eviction")
	}
	ok, _ = l.AllowIP(ip1)
	if !ok {
		t.Fatalf("expected ip1's fresh bucket to have full burst (2 tokens)")
	}
}

func TestAccountRetryIn_BackoffSchedule(t *testing.T) {
	clock := newFakeClock()
	l := NewLogin(Config{
		BackoffFree: 2,
		BackoffBase: time.Second,
		BackoffMax:  60 * time.Second,
		Now:         clock.Now,
	})

	// Unknown username: never backed off.
	if wait := l.AccountRetryIn("nobody"); wait != 0 {
		t.Fatalf("expected 0 wait for unknown username, got %s", wait)
	}

	cases := []struct {
		failures int
		want     time.Duration
	}{
		{1, 0},
		{2, 0},
		{3, 1 * time.Second},
		{4, 2 * time.Second},
		{5, 4 * time.Second},
		{6, 8 * time.Second},
		{7, 16 * time.Second},
		{8, 32 * time.Second},
		{9, 60 * time.Second}, // capped
		{10, 60 * time.Second},
	}

	username := "bob"
	for _, tc := range cases {
		l.RecordFailure(netip.Addr{}, username)
		wait := l.AccountRetryIn(username)
		if wait != tc.want {
			t.Fatalf("after %d failures: wait = %s, want %s", tc.failures, wait, tc.want)
		}
	}

	// Advance clock past the current wait and confirm it clears.
	clock.Advance(60 * time.Second)
	if wait := l.AccountRetryIn(username); wait != 0 {
		t.Fatalf("expected wait to clear after advancing past backoff, got %s", wait)
	}

	// RecordSuccess resets the counter entirely.
	l.RecordFailure(netip.Addr{}, username) // back into backoff territory eventually
	l.RecordSuccess(username)
	if wait := l.AccountRetryIn(username); wait != 0 {
		t.Fatalf("expected wait 0 after RecordSuccess, got %s", wait)
	}
}

func TestAcctLimiters_LRUEviction(t *testing.T) {
	clock := newFakeClock()
	// Note: zero-value Config fields select defaults (BackoffFree=2), so
	// there's no way to configure BackoffFree=0 explicitly; use the
	// default and push past it instead.
	l := NewLogin(Config{
		AcctCacheSize: 2,
		BackoffBase:   time.Second,
		BackoffMax:    60 * time.Second,
		Now:           clock.Now,
	})

	// Push "alice" past the free threshold (2 failures) so she'd be
	// backed off if still tracked.
	l.RecordFailure(netip.Addr{}, "alice")
	l.RecordFailure(netip.Addr{}, "alice")
	l.RecordFailure(netip.Addr{}, "alice")
	if wait := l.AccountRetryIn("alice"); wait == 0 {
		t.Fatalf("expected alice to be backed off before eviction")
	}

	// Touch "bob" and "carol": with cache size 2, "alice" (least
	// recently used) should be evicted.
	l.RecordFailure(netip.Addr{}, "bob")
	l.RecordFailure(netip.Addr{}, "carol")

	if wait := l.AccountRetryIn("alice"); wait != 0 {
		t.Fatalf("expected alice's counter to be evicted (fresh state), got wait %s", wait)
	}
}
