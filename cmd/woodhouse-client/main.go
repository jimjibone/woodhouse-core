package main

import (
	"context"
	"fmt"
	"os"
	"time"

	api "github.com/jimjibone/woodhouse-4/api/go"
	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/apitools"
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
				Usage:    "path to config storage location",
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

				clientService := clientsapi.NewClientServiceClient(conn)
				clientCtx, cancel := context.WithCancel(ctx)
				defer cancel()

				// Start sending our status to the server.
				statusStream, err := clientService.StatusStream(clientCtx)
				if err != nil {
					log.Errorf("failed to start status stream: %s", err)
					return
				}
				id, err := store.Get("wh.id")
				if err != nil {
					log.Errorf("failed to get client id: %s", err)
					return
				}
				statusStream.Send(&clientsapi.StatusUpdate{
					ClientInfo: &clientsapi.ClientInfo{
						Id:          string(id),
						Name:        "Client",
						Description: "A client bridge.",
						BootTime:    uint64(time.Now().Unix()),
					},
					DeviceInfo: []*clientsapi.Device{
						{
							Id:  "6F6B1AA7-D4D9-4C7B-998F-7B6C5E5DAF76",
							Typ: clientsapi.Device_SWITCH,
							Services: []*clientsapi.Service{
								{
									Id:  "lightbulb",
									Typ: clientsapi.Service_GENERIC,
									Attrs: []*clientsapi.Attribute{
										{
											Id: "on",
											Attr: &clientsapi.Attribute_Bool{
												Bool: &clientsapi.BoolAttribute{
													Value: false,
													Perms: clientsapi.Permissions_PERM_READWRITE,
												},
											},
										},
										{
											Id: "bri",
											Attr: &clientsapi.Attribute_Int{
												Int: &clientsapi.IntAttribute{
													Value: 0,
													Min:   0,
													Max:   100,
													Step:  1,
													Unit:  clientsapi.Unit_UNIT_PERCENTAGE,
													Perms: clientsapi.Permissions_PERM_READWRITE,
												},
											},
										},
									},
								},
							},
						},
					},
				})

				bridgeService := api.NewBridgeServiceClient(conn)

				response, err := bridgeService.SetBridgeInfo(context.Background(), &api.BridgeInfo{
					BridgeId:    "thing",
					Name:        "Thing",
					Description: "The thing.",
					BootTime:    apitools.TimeToTimestamp(time.Now()),
				})
				if err != nil {
					log.Errorf("failed to set bridge info: %s", err)
				} else {
					log.Infof("set bridge info: %s", response)
				}

				<-ctx.Done()
				log.Infof("client finishing!")
			}))

			// switchDevice := wh.NewDevice("1234", wh.DeviceTypeSwitch)
			// switchService := wh.NewService(wh.ServiceTypeSwitch)
			// switchAttr := wh.NewAttribute(wh.AttributeTypeSwitch)
			// switchService.AddAttribute(switchAttr)
			// switchDevice.AddService(switchService)
			// client.AddDevice(switchDevice)

			dev := NewSwitchDevice("1234")
			client.AddDevice(dev.dev)

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
