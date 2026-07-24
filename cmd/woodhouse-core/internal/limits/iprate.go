package limits

import (
	"net/netip"
	"time"

	"golang.org/x/time/rate"
)

// IPRateConfig tunes a generic per-IP token bucket limiter. Zero values
// select the defaults for CacheSize and Now only: PerMin, Burst, and
// Penalty have no sensible one-size-fits-all default and are always
// chosen per surface by the caller (e.g. pairing vs. keepalive pings
// have very different legitimate traffic shapes).
type IPRateConfig struct {
	PerMin    float64          // sustained requests per IP per minute
	Burst     int              // burst per IP
	Penalty   int              // extra tokens burned when Penalise is called
	CacheSize int              // tracked IPs before LRU eviction (default 65536)
	Now       func() time.Time // clock, overridable in tests (default time.Now)
}

func (c IPRateConfig) withDefaults() IPRateConfig {
	if c.CacheSize == 0 {
		c.CacheSize = 65536
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	return c
}

// IPRate is a generic per-source-IP token bucket limiter, for gating
// unauthenticated surfaces other than login (e.g. device pairing,
// keepalive pings). All methods are safe for concurrent use.
//
// Known limitations (shared with the login limiter in login.go):
//   - State is in-memory only: a restart clears all buckets (accepted
//     trade-off).
//   - Evicting an IP from the LRU (once CacheSize is exceeded) hands
//     that IP a fresh bucket with a full burst on its next request.
type IPRate struct {
	cfg IPRateConfig
	ips *ipLimiters
}

// NewIPRate constructs an IPRate limiter from cfg.
func NewIPRate(cfg IPRateConfig) *IPRate {
	cfg = cfg.withDefaults()
	return &IPRate{
		cfg: cfg,
		ips: newIPLimiters(rate.Limit(cfg.PerMin/60), cfg.Burst, cfg.CacheSize),
	}
}

// Allow reports whether a request from ip is within its rate budget,
// consuming one token if so. A zero or invalid Addr (e.g. the address
// couldn't be determined) shares a single bucket rather than bypassing
// the limit. rejections is the number of consecutive denials for this
// IP (including this one, if denied), so callers can throttle their own
// logging.
func (r *IPRate) Allow(ip netip.Addr) (ok bool, rejections int) {
	entry := r.ips.get(ip.Unmap())
	ok = entry.lim.AllowN(r.cfg.Now(), 1)

	entry.mu.Lock()
	if ok {
		entry.rejections = 0
	} else {
		entry.rejections++
	}
	rejections = entry.rejections
	entry.mu.Unlock()

	return ok, rejections
}

// Penalise charges ip an extra Penalty tokens, e.g. for a malformed or
// otherwise abusive request, so it exhausts its budget faster than
// well-behaved use.
func (r *IPRate) Penalise(ip netip.Addr) {
	entry := r.ips.get(ip.Unmap())
	entry.lim.AllowN(r.cfg.Now(), r.cfg.Penalty) //nolint:errcheck
}
