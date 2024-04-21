package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	app := &cli.App{
		Name:                 "whctl",
		Usage:                "Control woodhouse using your terminal.",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:  "store",
				Usage: "path to config storage location",
				Value: "~/.whctl",
			},
			&cli.StringFlag{
				Name:  "addr",
				Usage: "woodhouse server address (disables automatic discovery)",
				Value: "localhost:4001",
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
			// store := stores.NewFSStore(paths.AbsPathify(args.String("store")))

			// Require TLS but we don't care about trusting it, we'll sort that out in a
			// moment.
			// creds := credentials.NewTLS(&tls.Config{
			// 	InsecureSkipVerify: true,
			// })
			creds := insecure.NewCredentials()

			// Connect to the server.
			connCtx, connCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer connCancel()
			conn, err := grpc.DialContext(
				connCtx,
				args.String("addr"),
				grpc.WithTransportCredentials(creds),
			)
			if err != nil {
				return fmt.Errorf("connection failed: %s", err)
			}
			defer conn.Close()

			service := clientsapi.NewUserServiceClient(conn)

			deviceStream, err := service.GetDevices(context.Background(), &clientsapi.GetDevicesRequest{})
			if err != nil {
				return fmt.Errorf("failed to start devices stream: %s", err)
			}

			defer log.Infof("done!")

			for {
				dev, err := deviceStream.Recv()
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					return fmt.Errorf("failed receive device: %s", err)
				}
				log.Infof("device: %s", dev)
			}

			log.Infof("got devices!")

			actionRequest := &clientsapi.ActionRequest{
				DeviceId:  "fake123",
				ServiceId: "lightbulb",
				Values: []*clientsapi.Value{
					{
						Id: "on",
						Bool: &clientsapi.BoolValue{
							Value: true,
						},
					},
				},
			}
			log.Infof("sending action: %s", actionRequest)

			actionStream, err := service.SendAction(context.Background(), actionRequest)
			if err != nil {
				return fmt.Errorf("failed to start action stream: %s", err)
			}

			for {
				resp, err := actionStream.Recv()
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					return fmt.Errorf("failed receive action response: %s", err)
				}
				log.Infof("action response: %s", resp)
			}

			log.Infof("action finished!")

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
}
