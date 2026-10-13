package discovery

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/jimjibone/log"
)

const (
	defaultPollInterval = 2 * time.Second
	defaultMinBackoff   = 1 * time.Second
	defaultMaxBackoff   = 30 * time.Second
)

// server is the part of a zeroconf registration the broadcaster needs.
type server interface {
	Shutdown()
}

// netSnapshot is the set of interfaces that could be advertised on, plus a
// key that changes whenever any of their addresses change. An empty key means
// there is nothing to advertise.
type netSnapshot struct {
	ifaces []net.Interface
	key    string
}

type addrProvider func() netSnapshot
type registerFunc func(instance string, port int, ifaces []net.Interface) (server, error)

// Broadcaster advertises the service over mDNS. It does not require the host
// to have an address when created: a background loop registers once an
// interface has one, re-registers whenever the addresses change (DHCP renew,
// WiFi reconnect) and withdraws the advertisement while there are none.
type Broadcaster struct {
	mu       sync.Mutex
	server   server
	instance string
	port     int
	ifaces   []net.Interface // interfaces of the live registration
	key      string          // snapshot key of the live registration

	addrs        addrProvider
	register     registerFunc
	pollInterval time.Duration
	minBackoff   time.Duration
	maxBackoff   time.Duration

	wake     chan struct{}
	done     chan struct{}
	wg       sync.WaitGroup
	shutdown sync.Once
}

func NewBroadcaster(instance string, serveraddr net.Addr) (*Broadcaster, error) {
	_, port, err := net.SplitHostPort(serveraddr.String())
	if err != nil {
		return nil, err
	}
	portno, err := strconv.Atoi(port)
	if err != nil {
		return nil, err
	}
	b := newBroadcaster(instance, portno, systemAddrs, register)
	b.start()
	return b, nil
}

func newBroadcaster(instance string, port int, addrs addrProvider, reg registerFunc) *Broadcaster {
	return &Broadcaster{
		instance:     instance,
		port:         port,
		addrs:        addrs,
		register:     reg,
		pollInterval: defaultPollInterval,
		minBackoff:   defaultMinBackoff,
		maxBackoff:   defaultMaxBackoff,
		wake:         make(chan struct{}, 1),
		done:         make(chan struct{}),
	}
}

func (b *Broadcaster) start() {
	b.wg.Add(1)
	go b.run()
}

func (b *Broadcaster) run() {
	defer b.wg.Done()
	failures := 0
	for {
		if b.sync() {
			failures = 0
		} else {
			failures++
		}

		delay := b.pollInterval
		if failures > 0 {
			delay = b.backoff(failures)
		}
		timer := time.NewTimer(delay)
		select {
		case <-b.done:
			timer.Stop()
			return
		case <-b.wake:
			timer.Stop()
		case <-timer.C:
		}
	}
}

// sync brings the registration in line with the current addresses. It
// returns false if a registration was attempted and failed.
func (b *Broadcaster) sync() bool {
	snap := b.addrs()

	b.mu.Lock()
	defer b.mu.Unlock()

	select {
	case <-b.done:
		return true
	default:
	}

	if snap.key == "" {
		if b.server != nil {
			log.Infof("no host addresses, withdrawing mDNS advertisement %q", b.instance)
		}
		b.clear()
		return true
	}

	if b.server != nil && snap.key == b.key {
		return true
	}

	b.clear()
	srv, err := b.register(b.instance, b.port, snap.ifaces)
	if err != nil {
		log.Warnf("failed to advertise %q over mDNS, will retry: %s", b.instance, err)
		return false
	}
	b.server = srv
	b.ifaces = snap.ifaces
	b.key = snap.key
	log.Infof("advertising %q over mDNS on %s", b.instance, snap.key)
	return true
}

// clear withdraws any live registration. The caller must hold b.mu.
func (b *Broadcaster) clear() {
	if b.server != nil {
		b.server.Shutdown()
		b.server = nil
	}
	b.ifaces = nil
	b.key = ""
}

func (b *Broadcaster) backoff(failures int) time.Duration {
	d := b.minBackoff
	for i := 1; i < failures && d < b.maxBackoff; i++ {
		d *= 2
	}
	if d > b.maxBackoff {
		d = b.maxBackoff
	}
	return d
}

// Instance returns the name currently being advertised.
func (b *Broadcaster) Instance() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.instance
}

// SetInstance changes the advertised instance name. zeroconf cannot rename a
// live registration, so this withdraws the current advertisement and publishes
// a fresh one. There is a brief window where nothing is advertised; browsers
// re-resolve, so this is expected for a DNS-SD rename.
//
// If the new registration fails the previous name is restored, leaving the
// broadcaster as it was. If that restore also fails the broadcaster is left
// advertising nothing and the error says so; the background loop keeps
// retrying the previous name.
//
// If nothing is being advertised (no host addresses yet, or registration is
// failing) the name is stored and used by the next registration attempt.
func (b *Broadcaster) SetInstance(instance string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if instance == b.instance {
		return nil
	}

	if b.server == nil {
		b.instance = instance
		b.poke()
		return nil
	}

	b.server.Shutdown()
	b.server = nil

	srv, err := b.register(instance, b.port, b.ifaces)
	if err != nil {
		restored, restoreErr := b.register(b.instance, b.port, b.ifaces)
		if restoreErr != nil {
			b.clear()
			b.poke()
			return fmt.Errorf("failed to advertise %q (%w) and failed to restore %q: %w", instance, err, b.instance, restoreErr)
		}
		b.server = restored
		return fmt.Errorf("failed to advertise %q: %w", instance, err)
	}

	b.server = srv
	b.instance = instance
	return nil
}

// poke wakes the background loop without blocking.
func (b *Broadcaster) poke() {
	select {
	case b.wake <- struct{}{}:
	default:
	}
}

func (b *Broadcaster) Shutdown() {
	b.shutdown.Do(func() {
		close(b.done)
		b.wg.Wait()

		b.mu.Lock()
		defer b.mu.Unlock()
		b.clear()
	})
}

// systemAddrs lists the interfaces zeroconf would use (up and multicast) that
// have an address zeroconf would advertise (any non-loopback address).
func systemAddrs() netSnapshot {
	all, err := net.Interfaces()
	if err != nil {
		return netSnapshot{}
	}
	var ifaces []net.Interface
	var parts []string
	for _, iface := range all {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		var ips []string
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok || ipnet.IP.IsLoopback() {
				continue
			}
			ips = append(ips, ipnet.IP.String())
		}
		if len(ips) == 0 {
			continue
		}
		sort.Strings(ips)
		ifaces = append(ifaces, iface)
		parts = append(parts, iface.Name+"="+strings.Join(ips, ","))
	}
	sort.Strings(parts)
	return netSnapshot{ifaces: ifaces, key: strings.Join(parts, " ")}
}

func register(instance string, port int, ifaces []net.Interface) (server, error) {
	srv, err := zeroconf.Register(instance, "_woodhouse._tcp", "local.", port, []string{}, ifaces)
	if err != nil {
		return nil, err
	}
	return srv, nil
}
