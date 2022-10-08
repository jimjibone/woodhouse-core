package discovery

import (
	"net"
	"strconv"

	"github.com/grandcat/zeroconf"
)

type Broadcaster struct {
	server *zeroconf.Server
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
	server, err := zeroconf.Register(instance, "_woodhouse._tcp", "local.", portno, []string{}, nil)
	if err != nil {
		return nil, err
	}
	return &Broadcaster{server: server}, nil
}

func (b *Broadcaster) Shutdown() {
	b.server.Shutdown()
}
