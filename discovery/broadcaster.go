package discovery

import (
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/grandcat/zeroconf"
)

type Broadcaster struct {
	mu       sync.Mutex
	server   *zeroconf.Server
	instance string
	port     int
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
	server, err := register(instance, portno)
	if err != nil {
		return nil, err
	}
	return &Broadcaster{server: server, instance: instance, port: portno}, nil
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
// advertising nothing and the error says so.
func (b *Broadcaster) SetInstance(instance string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if instance == b.instance && b.server != nil {
		return nil
	}

	if b.server != nil {
		b.server.Shutdown()
		b.server = nil
	}

	server, err := register(instance, b.port)
	if err != nil {
		restored, restoreErr := register(b.instance, b.port)
		if restoreErr != nil {
			return fmt.Errorf("failed to advertise %q (%w) and failed to restore %q: %w", instance, err, b.instance, restoreErr)
		}
		b.server = restored
		return fmt.Errorf("failed to advertise %q: %w", instance, err)
	}

	b.server = server
	b.instance = instance
	return nil
}

func (b *Broadcaster) Shutdown() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.server == nil {
		return
	}
	b.server.Shutdown()
	b.server = nil
}

func register(instance string, port int) (*zeroconf.Server, error) {
	return zeroconf.Register(instance, "_woodhouse._tcp", "local.", port, []string{}, nil)
}
