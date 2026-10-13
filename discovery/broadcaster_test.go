package discovery

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

type fakeServer struct {
	instance string
	key      string
	mu       sync.Mutex
	down     bool
}

func (s *fakeServer) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.down = true
}

func (s *fakeServer) isDown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.down
}

type fakeNet struct {
	mu      sync.Mutex
	snap    netSnapshot
	fails   int // number of upcoming register calls to fail
	failFor map[string]bool
	servers []*fakeServer
}

func (f *fakeNet) set(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if key == "" {
		f.snap = netSnapshot{}
		return
	}
	f.snap = netSnapshot{ifaces: []net.Interface{{Name: "wlan0"}}, key: key}
}

func (f *fakeNet) addrs() netSnapshot {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.snap
}

func (f *fakeNet) register(instance string, port int, ifaces []net.Interface) (server, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fails > 0 {
		f.fails--
		return nil, errors.New("register failed")
	}
	if f.failFor[instance] {
		return nil, errors.New("register failed")
	}
	s := &fakeServer{instance: instance, key: f.snap.key}
	f.servers = append(f.servers, s)
	return s, nil
}

func (f *fakeNet) all() []*fakeServer {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*fakeServer(nil), f.servers...)
}

// live returns the registrations that have not been shut down.
func (f *fakeNet) live() []*fakeServer {
	var out []*fakeServer
	for _, s := range f.all() {
		if !s.isDown() {
			out = append(out, s)
		}
	}
	return out
}

func newTestBroadcaster(t *testing.T, f *fakeNet, instance string) *Broadcaster {
	t.Helper()
	b := newBroadcaster(instance, 1234, f.addrs, f.register)
	b.pollInterval = 5 * time.Millisecond
	b.minBackoff = 5 * time.Millisecond
	b.maxBackoff = 20 * time.Millisecond
	b.start()
	t.Cleanup(b.Shutdown)
	return b
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func liveIs(f *fakeNet, instance, key string) func() bool {
	return func() bool {
		l := f.live()
		return len(l) == 1 && l[0].instance == instance && l[0].key == key
	}
}

func TestBroadcasterWaitsForAddress(t *testing.T) {
	f := &fakeNet{}
	newTestBroadcaster(t, f, "alice")

	time.Sleep(30 * time.Millisecond)
	if n := len(f.all()); n != 0 {
		t.Fatalf("registered %d times with no address", n)
	}

	f.set("wlan0=192.0.2.10")
	waitFor(t, "registration", liveIs(f, "alice", "wlan0=192.0.2.10"))
}

func TestBroadcasterReregistersOnAddressChange(t *testing.T) {
	f := &fakeNet{}
	f.set("wlan0=192.0.2.10")
	newTestBroadcaster(t, f, "alice")
	waitFor(t, "registration", liveIs(f, "alice", "wlan0=192.0.2.10"))

	f.set("wlan0=192.0.2.20")
	waitFor(t, "re-registration", liveIs(f, "alice", "wlan0=192.0.2.20"))
	if n := len(f.all()); n != 2 {
		t.Fatalf("got %d registrations, want 2", n)
	}
}

func TestBroadcasterWithdrawsWhenAddressesGo(t *testing.T) {
	f := &fakeNet{}
	f.set("wlan0=192.0.2.10")
	newTestBroadcaster(t, f, "alice")
	waitFor(t, "registration", liveIs(f, "alice", "wlan0=192.0.2.10"))

	f.set("")
	waitFor(t, "withdrawal", func() bool { return len(f.live()) == 0 })

	f.set("wlan0=192.0.2.10")
	waitFor(t, "re-registration", liveIs(f, "alice", "wlan0=192.0.2.10"))
}

func TestBroadcasterRetriesFailedRegistration(t *testing.T) {
	f := &fakeNet{fails: 3}
	f.set("wlan0=192.0.2.10")
	newTestBroadcaster(t, f, "alice")
	waitFor(t, "registration after failures", liveIs(f, "alice", "wlan0=192.0.2.10"))
}

func TestBroadcasterBackoff(t *testing.T) {
	b := newBroadcaster("alice", 1234, nil, nil)
	b.minBackoff = time.Second
	b.maxBackoff = 10 * time.Second
	for failures, want := range map[int]time.Duration{
		1: time.Second, 2: 2 * time.Second, 3: 4 * time.Second, 4: 8 * time.Second, 5: 10 * time.Second, 50: 10 * time.Second,
	} {
		if got := b.backoff(failures); got != want {
			t.Errorf("backoff(%d) = %s, want %s", failures, got, want)
		}
	}
}

func TestBroadcasterSetInstanceWithoutAddress(t *testing.T) {
	f := &fakeNet{}
	b := newTestBroadcaster(t, f, "alice")

	if err := b.SetInstance("bob"); err != nil {
		t.Fatalf("SetInstance: %v", err)
	}
	if got := b.Instance(); got != "bob" {
		t.Fatalf("Instance() = %q, want bob", got)
	}

	f.set("wlan0=192.0.2.10")
	waitFor(t, "registration under new name", liveIs(f, "bob", "wlan0=192.0.2.10"))
	for _, s := range f.all() {
		if s.instance != "bob" {
			t.Fatalf("registered stale name %q", s.instance)
		}
	}
}

func TestBroadcasterSetInstanceLive(t *testing.T) {
	f := &fakeNet{}
	f.set("wlan0=192.0.2.10")
	b := newTestBroadcaster(t, f, "alice")
	waitFor(t, "registration", liveIs(f, "alice", "wlan0=192.0.2.10"))

	if err := b.SetInstance("bob"); err != nil {
		t.Fatalf("SetInstance: %v", err)
	}
	if !liveIs(f, "bob", "wlan0=192.0.2.10")() {
		t.Fatalf("live registrations after rename: %+v", f.live())
	}
}

func TestBroadcasterSetInstanceRestoresOnFailure(t *testing.T) {
	f := &fakeNet{failFor: map[string]bool{"bob": true}}
	f.set("wlan0=192.0.2.10")
	b := newTestBroadcaster(t, f, "alice")
	waitFor(t, "registration", liveIs(f, "alice", "wlan0=192.0.2.10"))

	if err := b.SetInstance("bob"); err == nil {
		t.Fatal("SetInstance succeeded, want error")
	}
	if got := b.Instance(); got != "alice" {
		t.Fatalf("Instance() = %q, want alice", got)
	}
	if !liveIs(f, "alice", "wlan0=192.0.2.10")() {
		t.Fatalf("live registrations after failed rename: %+v", f.live())
	}
}

func TestBroadcasterShutdown(t *testing.T) {
	f := &fakeNet{}
	f.set("wlan0=192.0.2.10")
	b := newTestBroadcaster(t, f, "alice")
	waitFor(t, "registration", liveIs(f, "alice", "wlan0=192.0.2.10"))

	b.Shutdown()
	if n := len(f.live()); n != 0 {
		t.Fatalf("%d registrations live after Shutdown", n)
	}

	f.set("wlan0=192.0.2.20")
	time.Sleep(30 * time.Millisecond)
	if n := len(f.all()); n != 1 {
		t.Fatalf("registered after Shutdown: %d registrations", n)
	}
	b.Shutdown() // idempotent
}
