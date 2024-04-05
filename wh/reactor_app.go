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
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/stores"
	whv1 "github.com/jimjibone/woodhouse-4/wh/v1"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
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
		&cli.PathFlag{
			Name:     "store",
			Usage:    "path to config storage location",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "addr",
			Usage:    "woodhouse-core server address",
			Required: true,
		},
		&cli.BoolFlag{
			Name:    "debug",
			Aliases: []string{"v"},
			Usage:   "enable debug logging",
		},
	}, ra.Flags...)
	app := &cli.App{
		Name:                 ra.Name,
		Usage:                ra.Description,
		EnableBashCompletion: true,
		Flags:                flags,
		Action: func(args *cli.Context) error {
			if args.Bool("debug") {
				log.SetOptions(log.WithMinLevel(log.DebugLevel))
			}

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

			// Create the store.
			store := stores.NewFSStore(args.String("store"))

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
					errs <- reactFunc(args, ctx, ra.reactor)
					wg.Done()
				}(reactFunc)
			}

			// Run the connection stuff.
			wg.Add(1)
			go func() {
				client := whv1.NewClient(store, args.String("addr"), whv1.WithClientID(ra.BridgeID),
					whv1.WithConnectionHandler(func(ctx context.Context, conn *grpc.ClientConn) {
						err := ra.bridge.Run(ctx, conn)
						if err != nil {
							log.Errorf("bridge run failed: %s", err)
						}
					}),
					whv1.WithConnectionHandler(func(ctx context.Context, conn *grpc.ClientConn) {
						err := ra.reactor.Run(ctx, conn)
						if err != nil {
							log.Errorf("reactor run failed: %s", err)
						}
					}),
				)
				err := client.Run()
				if err != nil {
					errs <- err
				}
				wg.Done()
			}()

			// Wait for program exit...
			var finalErr error
			select {
			case <-ctx.Done():
			case err := <-errs:
				// Eventually return this error if things are not ok.
				if err != nil {
					finalErr = err
				}
			}
			// Tell the other goroutines to cancel.
			cancel()
			wg.Wait()
			return finalErr
		},
	}

	return app.Run(arguments)
}
