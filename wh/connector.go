package wh

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jimjibone/woodhouse-4/discovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	minBackoff = time.Second
	maxBackoff = 30 * time.Second
)

type Connector struct {
	disableDiscovery bool
	serverAddr       string
	lastBackoff      time.Time
	backoffDuration  time.Duration
	onConnected      ConnectionHandler
}

type ConnectionHandler func(ctx context.Context, conn *grpc.ClientConn) error

// Create a new connector.
func NewConnector(onConnected ConnectionHandler) *Connector {
	return &Connector{
		onConnected: onConnected,
	}
}

// Set the remote server address and disable automatic discovery.
func (c *Connector) SetServerAddr(addr string) {
	c.disableDiscovery = true
	c.serverAddr = addr
}

// Run the connector. Calls OnConnected after connecting to woodhouse.
func (c *Connector) Run(ctx context.Context) error {
	log.Printf("connector started")
	defer log.Printf("connector finished")

	for {
		found, err := c.discover(ctx)
		if err != nil {
			return err
		}
		if found {
			conn, err := c.connect(ctx)
			if err != nil {
				return err
			}
			if conn != nil {
				err = c.run(ctx, conn)
				conn.Close()
				if err != nil {
					return err
				}
			}
		}

		// Check if we're done.
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Backoff for a short while before the next connection attempt.
		c.backoff(ctx)
	}
}

func (c *Connector) discover(ctx context.Context) (found bool, err error) {
	if !c.disableDiscovery {
		found = false

		log.Printf("starting discovery")

		// Start listening for woodhouse cores.
		listener := discovery.NewListener("woodhouse-core")
		if err := listener.Start(); err != nil {
			return false, fmt.Errorf("failed to start discovery: %w", err)
		}
		defer listener.Stop()

		done := false
		for !done {
			select {
			case <-ctx.Done():
				done = true
			case result := <-listener.Results():
				log.Printf("discovered instance: %s, hostname: %s, addr: %s", result.Instance, result.Hostname, result.Addr)
				done = true
				found = true
				c.serverAddr = result.Addr
			}
		}
	} else {
		found = true
		log.Printf("using predefined server address: %s", c.serverAddr)
	}
	return found, nil
}

func (c *Connector) connect(ctx context.Context) (conn *grpc.ClientConn, err error) {
	// Connect and send our bridge info.
	log.Printf("connecting to: %s", c.serverAddr)
	// TODO: require valid certs
	// creds := credentials.NewTLS(&tls.Config{})
	creds := insecure.NewCredentials()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err = grpc.DialContext(
		ctx,
		c.serverAddr,
		grpc.WithTransportCredentials(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	log.Printf("connection complete!")
	return conn, nil
}

func (c *Connector) backoff(ctx context.Context) {
	// Reset the backoff duration if the backoff has not been used for a
	// suitable amount of time.
	dt := time.Since(c.lastBackoff)
	if dt > c.backoffDuration {
		log.Printf("backoff reset after %s", dt)
		c.backoffDuration = minBackoff
	}
	c.lastBackoff = time.Now()
	log.Printf("starting backoff for %s", c.backoffDuration)
	timer := time.NewTimer(c.backoffDuration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
	log.Printf("backoff finished")
	c.backoffDuration = c.backoffDuration * 2
}

func (c *Connector) run(ctx context.Context, conn *grpc.ClientConn) error {
	log.Printf("connected!")

	if c.onConnected != nil {
		return c.onConnected(ctx, conn)
	}
	return fmt.Errorf("no connection handler")
}
