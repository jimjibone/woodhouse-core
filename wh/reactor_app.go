package wh

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/urfave/cli/v2"
)

type ReactorApp struct {
	Name     string        // The name of the program. Defaults to path.Base(os.Args[0])
	Usage    string        // Description of the program.
	Flags    []cli.Flag    // Additional list of flags to parse.
	Reactors []ReactorFunc // Reactor functions to be called once the reactor starts.
}

type ReactorFunc func(ctx context.Context, reactor *Reactor) error

func (ra *ReactorApp) Run(arguments []string) error {
	flags := append([]cli.Flag{
		&cli.StringFlag{
			Name:  "addr",
			Usage: "woodhouse-core server address (disables automatic discovery)",
		},
	}, ra.Flags...)
	app := &cli.App{
		Name:                 ra.Name,
		Usage:                ra.Usage,
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

			// Create the reactor.
			reactor := NewReactor()

			// Collect errors from goroutines.
			var wg sync.WaitGroup
			errs := make(chan error, 1)

			// Run the reactor stuff.
			for _, reactFunc := range ra.Reactors {
				wg.Add(1)
				go func(reactFunc ReactorFunc) {
					err := reactFunc(ctx, reactor)
					if err != nil {
						errs <- err
					}
					wg.Done()
				}(reactFunc)
			}

			// Run the connection stuff.
			wg.Add(1)
			go func() {
				connector := NewConnector(reactor.Run)
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
