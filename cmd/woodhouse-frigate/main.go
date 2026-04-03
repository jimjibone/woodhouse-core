package main

import (
	"context"
	"os"
	"sync"

	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/stores"
	"github.com/jimjibone/woodhouse-4/wh/v1"
	"github.com/urfave/cli/v2"
)

const debugSaveJson bool = true

func main() {
	app := &cli.App{
		Name:  "woodhouse-frigate",
		Usage: "Woodhouse client for a Frigate installation.",
		Flags: []cli.Flag{
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
			&cli.StringFlag{
				Name:  "id",
				Usage: "ID used by this bridge",
				Value: "frigate",
			},
			&cli.StringFlag{
				Name:     "frigate",
				Usage:    "frigate server address (e.g. localhost:8088)",
				Required: true,
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"v"},
				Usage:   "enable debug logging",
			},
		},
		Before: func(args *cli.Context) error {
			// Setup logging.
			if args.Bool("debug") {
				log.SetOptions(log.WithMinLevel(log.DebugLevel))
			} else {
				log.SetOptions(log.WithMinLevel(log.InfoLevel))
			}
			return nil
		},
		Action: func(args *cli.Context) error {
			// Create the store.
			store := stores.NewFSStore(args.String("store"))

			// Create the client.
			client := wh.NewClient(
				store,
				args.String("addr"),
				wh.WithClientID(args.String("id")),
				wh.WithClientInfo("Frigate Bridge", "Bridge for integrating Frigate with Woodhouse", "0.1.0"),
				wh.WithImages(),
			)

			// Start the Frigate goroutine.
			wg := &sync.WaitGroup{}
			defer wg.Wait()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			wg.Add(1)
			go func() {
				frigate := Frigate{
					ServerAddr: args.String("frigate"),
				}
				err := frigate.Run(ctx, client)
				if err != nil {
					log.Errorf("failed to run: %s", err)
				}
				wg.Done()
			}()

			// Run the client.
			err := client.Run()
			if err != nil {
				return err
			}

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
}
