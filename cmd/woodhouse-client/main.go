package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/stores"
	"github.com/jimjibone/woodhouse-4/wh/v1"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

func main() {
	app := &cli.App{
		Name:                 "woodhouse-client",
		Usage:                "Runs the woodhouse client",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:     "store",
				Usage:    "path to secrets storage location",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "addr",
				Usage:    "woodhouse server address (disables automatic discovery)",
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
			client := wh.NewClient(store, args.String("addr"), wh.WithConnectionHandler(func(ctx context.Context, conn *grpc.ClientConn) {
				log.Infof("client connected and waiting for more code!")
				<-ctx.Done()
				log.Infof("client finishing!")
			}))

			err := client.Run()
			if err != nil {
				return fmt.Errorf("failed to run client: %s", err)
			}

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
}
