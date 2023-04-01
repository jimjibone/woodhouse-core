package wh

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/apitools"
	"github.com/urfave/cli/v2"
)

type ReactorApp struct {
	BridgeID    string        // A unique bridge ID for this reactor.
	Name        string        // The name of the program. Defaults to path.Base(os.Args[0])
	Description string        // A description for this reactor.
	Flags       []cli.Flag    // Additional list of flags to parse.
	Reactors    []ReactorFunc // Reactor functions to be called once the reactor starts.
	reactor     *Reactor
	bridge      *Bridge
}

type ReactorFunc func(args *cli.Context, ctx context.Context, reactor *Reactor) error

// Return the reactor.
func (ra *ReactorApp) Reactor() *Reactor {
	if ra.reactor == nil {
		ra.reactor = NewReactor()
	}
	return ra.reactor
}

func (ra *ReactorApp) createBridge() {
	ra.bridge = NewBridge(&api.BridgeInfo{
		BridgeId:    ra.BridgeID,
		Name:        ra.Name,
		Description: ra.Description,
		BootTime:    apitools.TimeToTimestamp(time.Now()),
	})
}

func (ra *ReactorApp) AddDevice(deviceID string, device Device) {
	if ra.bridge == nil {
		ra.createBridge()
	}
	ra.bridge.AddDevice(deviceID, device)
}

func (ra *ReactorApp) Run(arguments []string) error {
	flags := append([]cli.Flag{
		&cli.StringFlag{
			Name:  "addr",
			Usage: "woodhouse-core server address (disables automatic discovery)",
		},
	}, ra.Flags...)
	app := &cli.App{
		Name:                 ra.Name,
		Usage:                ra.Description,
		EnableBashCompletion: true,
		Flags:                flags,
		Action: func(args *cli.Context) error {
			// Create the main app context.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Cancel the main app context if an interrupt is received.
			var sig = make(chan os.Signal, 2)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
			go func() {
				select {
				case <-ctx.Done():
				case <-sig:
					cancel()
				}
			}()

			// Create the bridge if not done already.
			if ra.bridge == nil {
				ra.createBridge()
			}

			// Create the reactor.
			ra.Reactor()

			// Collect errors from goroutines.
			var wg sync.WaitGroup
			errs := make(chan error, len(ra.Reactors)+1)

			// Run the reactor stuff.
			for _, reactFunc := range ra.Reactors {
				wg.Add(1)
				go func(reactFunc ReactorFunc) {
					err := reactFunc(args, ctx, ra.reactor)
					if err != nil {
						errs <- err
					}
					wg.Done()
				}(reactFunc)
			}

			// Run the connection stuff.
			wg.Add(1)
			go func() {
				// connector := NewConnector(reactor.Run)
				connector := NewConnector(ra.bridge.Run, ra.reactor.Run)
				err := connector.Run(ctx)
				if err != nil {
					errs <- err
				}
				wg.Done()
			}()

			// Wait for program exit...
			select {
			case <-ctx.Done():
			case err := <-errs:
				return err
			}
			cancel()
			wg.Wait()
			return nil
		},
	}

	return app.Run(arguments)
}
