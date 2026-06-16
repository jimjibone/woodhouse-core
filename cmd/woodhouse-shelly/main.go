package main

import (
	"context"
	"os"
	"sync"

	"github.com/jimjibone/log"
	"github.com/jimjibone/woodhouse-core/shared/stores"
	"github.com/jimjibone/woodhouse-core/wh/v1"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "woodhouse-shelly",
		Usage: "Woodhouse client for Shelly devices.",
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
				Value: "shelly",
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
				wh.WithClientInfo("Shelly Bridge", "Bridge for Shelly devices", "0.1.0"),
			)

			// Start the Shelly goroutine.
			wg := &sync.WaitGroup{}
			defer wg.Wait()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			wg.Add(1)
			go func() {
				err := shellyStuff(wg, ctx, client)
				if err != nil {
					log.Errorf("failed to run: %s", err)
				}
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
