package main

import (
	"fmt"
	"log"
	"net"
	"os"

	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/discovery"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	app := &cli.App{
		Name:                 "woodhouse",
		Usage:                "Runs the woodhouse core",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "addr",
				Usage: "woodhouse api server address",
				Value: ":4000",
			},
			&cli.StringFlag{
				Name:      "config",
				Usage:     "Load configuration from `DIR`",
				EnvVars:   []string{"WOODHOUSE_CONFIG"},
				Value:     "woodhouse.yml",
				TakesFile: true,
			},
		},
		Before: func(c *cli.Context) error {
			return nil
		},
		After: func(c *cli.Context) error {
			return nil
		},
		Action: func(c *cli.Context) error {
			// Try to listen on the selected server addresses.
			lis, err := net.Listen("tcp", c.String("addr"))
			if err != nil {
				return fmt.Errorf("failed to listen on addr: %w", err)
			}

			// Create services.
			reactorService := NewReactorService()
			bridgeService := NewBridgeService(reactorService)

			// Broadcast our existence.
			broadcaster, err := discovery.NewBroadcaster("woodhouse-core", lis.Addr())
			if err != nil {
				return fmt.Errorf("failed to create broadcaster: %w", err)
			}
			defer broadcaster.Shutdown()

			// Create the gRPC server.
			// TODO: require valid certs
			// creds := credentials.NewTLS(&tls.Config{
			// 	InsecureSkipVerify: true,
			// })
			server := grpc.NewServer(
			// grpc.Creds(creds),
			)
			api.RegisterBridgeServiceServer(server, bridgeService)
			api.RegisterReactorServiceServer(server, reactorService)
			reflection.Register(server)

			// Run the server.
			log.Printf("bridge api server ready at grpc://%s", lis.Addr())
			if err := server.Serve(lis); err != nil {
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
