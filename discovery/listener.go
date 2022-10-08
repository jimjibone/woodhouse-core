package discovery

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/grandcat/zeroconf"
)

type ServerInfo struct {
	Instance string
	Hostname string
	Addr     string
}

type Listener struct {
	instance string
	entries  chan *zeroconf.ServiceEntry
	results  chan ServerInfo
	wg       sync.WaitGroup
	cancel   func()
}

func NewListener(instance string) *Listener {
	l := &Listener{
		instance: instance,
		entries:  make(chan *zeroconf.ServiceEntry),
		results:  make(chan ServerInfo),
	}
	return l
}

func (l *Listener) Results() <-chan ServerInfo {
	return l.results
}

func (l *Listener) Start() error {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return fmt.Errorf("failed to start: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel

	err = resolver.Browse(ctx, "_woodhouse._tcp", ".local", l.entries)
	if err != nil {
		return fmt.Errorf("failed to browse: %w", err)
	}

	l.wg.Add(1)
	go l.run(resolver)
	return nil
}

func (l *Listener) Stop() {
	if l.cancel != nil {
		l.cancel()
		l.wg.Wait()
	}
}

func (l *Listener) run(resolver *zeroconf.Resolver) {
	defer l.wg.Done()
	for entry := range l.entries {
		ipstring := ""
		if len(entry.AddrIPv4) > 0 {
			ipstring = entry.AddrIPv4[0].String()
		}
		hostname := strings.TrimSuffix(entry.HostName, ".local.")

		if strings.Contains(entry.Instance, l.instance) {
			l.results <- ServerInfo{
				Instance: entry.Instance,
				Hostname: hostname,
				Addr:     net.JoinHostPort(ipstring, strconv.Itoa(entry.Port)),
			}
		}
	}
}
