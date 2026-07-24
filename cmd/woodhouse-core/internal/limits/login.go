// Package limits implements the core server's login abuse controls
// (SECURITY-REVIEW H-2): a per-source-IP token bucket ahead of password
// verification, and a per-account progressive backoff. Both gates run
// before the argon2id password check so an unthrottled attacker cannot
// use the login endpoint as either a credential-guessing oracle or a
// CPU-exhaustion lever (argon2id is tuned to 64 MiB/attempt).
//
// Known limitations:
//   - IPv6 attackers rotating a /64 get a fresh bucket per address; this
//     is bounded only by the IP LRU (IPCacheSize), so the per-account
//     backoff is the real backstop regardless of source address.
//   - State is in-memory only: a restart clears all counters and
//     backoffs (accepted trade-off).
//   - Evicting an IP from the LRU (once IPCacheSize is exceeded) hands
//     that IP a fresh bucket with a full burst on its next request; an
//     attacker would need to churn IPCacheSize distinct addresses to
//     exploit this, which the account backoff still bounds.
package limits

import (
	"container/list"
	"net/netip"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Config tunes the login limiter. Zero values select the defaults.
type Config struct {
	LoginPerMin   float64          // sustained login attempts per IP per minute (default 60, i.e. 1/s)
	LoginBurst    int              // burst per IP (default 10)
	FailPenalty   int              // extra tokens burned per failed credential (default 3)
	IPCacheSize   int              // tracked IPs before LRU eviction (default 65536)
	AcctCacheSize int              // tracked usernames before LRU eviction (default 8192)
	BackoffFree   int              // failures allowed before backoff kicks in (default 2)
	BackoffBase   time.Duration    // initial backoff after BackoffFree is exceeded (default 1s)
	BackoffMax    time.Duration    // backoff cap (default 60s)
	Now           func() time.Time // clock, overridable in tests (default time.Now)
}

func (c Config) withDefaults() Config {
	if c.LoginPerMin == 0 {
		c.LoginPerMin = 60
	}
	if c.LoginBurst == 0 {
		c.LoginBurst = 10
	}
	if c.FailPenalty == 0 {
		c.FailPenalty = 3
	}
	if c.IPCacheSize == 0 {
		c.IPCacheSize = 65536
	}
	if c.AcctCacheSize == 0 {
		c.AcctCacheSize = 8192
	}
	if c.BackoffFree == 0 {
		c.BackoffFree = 2
	}
	if c.BackoffBase == 0 {
		c.BackoffBase = time.Second
	}
	if c.BackoffMax == 0 {
		c.BackoffMax = 60 * time.Second
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	return c
}

// Login enforces the login abuse controls. All methods are safe for
// concurrent use.
type Login struct {
	cfg Config

	ips      *ipLimiters
	accounts *acctLimiters
}

func NewLogin(cfg Config) *Login {
	cfg = cfg.withDefaults()
	return &Login{
		cfg:      cfg,
		ips:      newIPLimiters(rate.Limit(cfg.LoginPerMin/60), cfg.LoginBurst, cfg.IPCacheSize),
		accounts: newAcctLimiters(cfg.AcctCacheSize),
	}
}

// AllowIP reports whether a login attempt from ip is within its rate
// budget, consuming one token if so. A zero or invalid Addr (e.g. the
// address couldn't be parsed) shares a single bucket rather than
// bypassing the limit. rejections is the number of consecutive denials
// for this IP (including this one, if denied), so callers can throttle
// their own logging.
func (l *Login) AllowIP(ip netip.Addr) (ok bool, rejections int) {
	entry := l.ips.get(ip.Unmap())
	ok = entry.lim.AllowN(l.cfg.Now(), 1)

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

// RecordFailure charges ip extra tokens for a failed login attempt
// (so guessing exhausts the IP's budget faster than legitimate use)
// and bumps username's consecutive-failure counter, which drives
// AccountRetryIn's backoff. username is recorded whether or not it
// corresponds to a real account, otherwise the backoff itself would
// leak account existence.
func (l *Login) RecordFailure(ip netip.Addr, username string) {
	entry := l.ips.get(ip.Unmap())
	entry.lim.AllowN(l.cfg.Now(), l.cfg.FailPenalty) //nolint:errcheck

	now := l.cfg.Now()
	acct := l.accounts.get(username)
	acct.mu.Lock()
	acct.failures++
	acct.last = now
	acct.mu.Unlock()
}

// RecordSuccess clears username's consecutive-failure counter.
func (l *Login) RecordSuccess(username string) {
	l.accounts.remove(username)
}

// AccountRetryIn reports how much longer username must wait before its
// next login attempt, or 0 if it's allowed now. It does not consume
// anything, so it's safe to call as a read-only gate ahead of the
// password check.
func (l *Login) AccountRetryIn(username string) time.Duration {
	acct := l.accounts.peek(username)
	if acct == nil {
		return 0
	}

	acct.mu.Lock()
	failures, last := acct.failures, acct.last
	acct.mu.Unlock()

	if failures <= l.cfg.BackoffFree {
		return 0
	}

	backoff := l.cfg.BackoffBase << (failures - l.cfg.BackoffFree - 1)
	if backoff > l.cfg.BackoffMax || backoff <= 0 {
		backoff = l.cfg.BackoffMax
	}

	remaining := last.Add(backoff).Sub(l.cfg.Now())
	remaining = max(remaining, 0)
	return remaining
}

// ipLimiters is a size-capped LRU of per-IP token buckets.
type ipLimiters struct {
	limit rate.Limit
	burst int
	max   int

	mu  sync.Mutex
	lru *list.List // of *ipEntry, front = most recent
	m   map[netip.Addr]*list.Element
}

type ipEntry struct {
	ip  netip.Addr
	lim *rate.Limiter

	mu         sync.Mutex
	rejections int
}

func newIPLimiters(limit rate.Limit, burst, max int) *ipLimiters {
	return &ipLimiters{
		limit: limit,
		burst: burst,
		max:   max,
		lru:   list.New(),
		m:     make(map[netip.Addr]*list.Element),
	}
}

func (il *ipLimiters) get(ip netip.Addr) *ipEntry {
	il.mu.Lock()
	defer il.mu.Unlock()
	if el, ok := il.m[ip]; ok {
		il.lru.MoveToFront(el)
		return el.Value.(*ipEntry)
	}
	entry := &ipEntry{ip: ip, lim: rate.NewLimiter(il.limit, il.burst)}
	il.m[ip] = il.lru.PushFront(entry)
	if il.lru.Len() > il.max {
		oldest := il.lru.Back()
		il.lru.Remove(oldest)
		delete(il.m, oldest.Value.(*ipEntry).ip)
	}
	return entry
}

// acctLimiters is a size-capped LRU of per-username failure counters.
type acctLimiters struct {
	max int

	mu  sync.Mutex
	lru *list.List // of *acctEntryHolder, front = most recent
	m   map[string]*list.Element
}

type acctEntryHolder struct {
	username string
	entry    *acctEntry
}

type acctEntry struct {
	mu       sync.Mutex
	failures int
	last     time.Time
}

func newAcctLimiters(max int) *acctLimiters {
	return &acctLimiters{
		max: max,
		lru: list.New(),
		m:   make(map[string]*list.Element),
	}
}

// get returns username's entry, creating it (and moving it to the
// front of the LRU) if necessary.
func (al *acctLimiters) get(username string) *acctEntry {
	al.mu.Lock()
	defer al.mu.Unlock()
	if el, ok := al.m[username]; ok {
		al.lru.MoveToFront(el)
		return el.Value.(*acctEntryHolder).entry
	}
	entry := &acctEntry{}
	al.m[username] = al.lru.PushFront(&acctEntryHolder{username: username, entry: entry})
	if al.lru.Len() > al.max {
		oldest := al.lru.Back()
		al.lru.Remove(oldest)
		delete(al.m, oldest.Value.(*acctEntryHolder).username)
	}
	return entry
}

// peek returns username's entry without creating one, or nil if
// there isn't one. It still promotes an existing entry to the front
// of the LRU.
func (al *acctLimiters) peek(username string) *acctEntry {
	al.mu.Lock()
	defer al.mu.Unlock()
	el, ok := al.m[username]
	if !ok {
		return nil
	}
	al.lru.MoveToFront(el)
	return el.Value.(*acctEntryHolder).entry
}

func (al *acctLimiters) remove(username string) {
	al.mu.Lock()
	defer al.mu.Unlock()
	if el, ok := al.m[username]; ok {
		al.lru.Remove(el)
		delete(al.m, username)
	}
}
