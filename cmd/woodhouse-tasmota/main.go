package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/stores"
	"github.com/jimjibone/woodhouse-4/wh/v1"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
	"github.com/urfave/cli/v2"
)

type StatusResponse struct {
	Status struct {
		Module       int
		DeviceName   string
		FriendlyName []string
		Topic        string
		ButtonTopic  string
		Power        int
		PowerOnState int
		LedState     int
		LedMask      string
		SaveData     int
		SaveState    int
		SwitchTopic  string
		SwitchMode   []int
		ButtonRetain int
		SwitchRetain int
		SensorRetain int
		PowerRetain  int
		InfoRetain   int
		StateRetain  int
	}
	StatusNET struct {
		Hostname   string
		IPAddress  string
		Gateway    string
		Subnetmask string
		DNSServer  string
		Mac        string
		Webserver  int
		WifiConfig int
		WifiPower  float64
	}
}

func (sr *StatusResponse) Name() string {
	name := sr.Status.Topic
	if len(sr.Status.FriendlyName) > 0 {
		name = sr.Status.FriendlyName[0]
	}
	return name
}

func (sr *StatusResponse) ID() string {
	id := sr.StatusNET.Mac
	id = strings.ToLower(strings.Join(strings.Split(id, ":"), ""))
	return id
}

func (sr *StatusResponse) Power() bool {
	return sr.Status.Power == 1
}

func (sr *StatusResponse) String() string {
	power := "off"
	if sr.Power() {
		power = "on"
	}
	return fmt.Sprintf("id: %q, name: %q, mac: %s, power: %s", sr.ID(), sr.Name(), sr.StatusNET.Mac, power)
}

func GetCommand(ctx context.Context, endpoint, command string) ([]byte, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/cm?cmnd=%s", endpoint, command), nil)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func GetStatus(ctx context.Context, endpoint string) (*StatusResponse, error) {
	resp, err := GetCommand(ctx, endpoint, "Status%%200")
	if err != nil {
		return nil, err
	}

	status := &StatusResponse{}
	err = json.Unmarshal(resp, status)
	if err != nil {
		return nil, err
	}

	return status, nil
}

type PowerResponse struct {
	PowerStr string `json:"POWER"`
}

func (pr *PowerResponse) Power() bool {
	return pr.PowerStr == "ON"
}

func SetPower(ctx context.Context, endpoint string, on bool) (*PowerResponse, error) {
	onString := "off"
	if on {
		onString = "on"
	}
	body, err := GetCommand(ctx, endpoint, fmt.Sprintf("Power%%20%s", onString))
	if err != nil {
		return nil, err
	}

	resp := &PowerResponse{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type TasmotaDevice struct {
	log    *log.Context
	client *wh.Client
	ip     string
	dev    *devices.Device
	info   *services.Info
	online *services.Online
	relay  *services.Relay
}

func NewTasmotaDevice(client *wh.Client, id, ip string) *TasmotaDevice {
	dev := &TasmotaDevice{
		log:    log.NewContext(log.DefaultLogger, id, log.DebugLevel),
		client: client,
		ip:     ip,
		dev:    devices.NewDevice(id, clientsapi.Device_RELAY),
		info:   services.NewInfo(),
		online: services.NewOnline(),
		relay:  services.NewRelay("relay"),
	}

	dev.log.Infof("created device at %s", ip)

	dev.relay.On.OnAction(func(val bool) {
		resp, err := SetPower(context.Background(), dev.ip, val)
		if err != nil {
			dev.log.Errorf("failed to set power: %s", err)
		}
		if dev.relay.On.Set(resp.Power()) {
			powerString := "off"
			if resp.Power() {
				powerString = "on"
			}
			dev.log.Infof("via request: power changed to %s", powerString)
		}
	})

	dev.dev.AddService(dev.info, dev.online, dev.relay)
	if err := client.AddDevice(dev.dev); err != nil {
		panic(err)
	}

	return dev
}

func (dev *TasmotaDevice) PollStatus(ctx context.Context) {
	status, err := GetStatus(ctx, dev.ip)
	if err != nil {
		// Assume offline.
		if dev.online.Online.Set(false) {
			dev.log.Infof("went offline")
		}
		return
	}

	if dev.info.Name.Set(status.Name()) {
		dev.log.Infof("name changed to %q", status.Name())
	}
	dev.online.LastSeen.Set(time.Now())
	if dev.online.Online.Set(true) {
		dev.log.Infof("came online")
	}
	if dev.relay.On.Set(status.Power()) {
		powerString := "off"
		if status.Power() {
			powerString = "on"
		}
		dev.log.Infof("power changed to %s", powerString)
	}
}

func main() {
	app := &cli.App{
		Name:  "woodhouse-tasmota",
		Usage: "Woodhouse client for Tasmota devices.",
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
		Commands: []*cli.Command{
			{
				Name:      "add-device",
				Usage:     "Add a Tasmota device by its IP",
				ArgsUsage: "<device-ip-1> <device-ip-2...>",
				Action: func(args *cli.Context) error {
					// Create the store.
					store := stores.NewFSStore(args.String("store"))

					// Load previous endpoints.
					endpoints := make(map[string]string) // key=id (mac), value=ip
					if store.Has("endpoints") {
						err := stores.GetJson(store, "endpoints", &endpoints)
						if err != nil {
							return fmt.Errorf("failed to get previous endpoints: %w", err)
						}
					}

					// Parse remaining args for the device IP address.
					ips := args.Args().Slice()
					if len(ips) == 0 {
						return fmt.Errorf("no IP address provided")
					}
					for _, rawIP := range ips {
						// Check that this is a valid IP.
						ip := net.ParseIP(rawIP)
						if ip == nil {
							return fmt.Errorf("invalid IP address %q", rawIP)
						}
						ipString := ip.String()
						fmt.Println(ipString)

						// Check that we can connect to this IP.
						status, err := GetStatus(context.Background(), ipString)
						if err != nil {
							return err
						}
						fmt.Println(status)

						// Add the IP to the list if it's new.
						endpoints[status.ID()] = ipString
					}

					// Save the new endpoints.
					err := stores.SetJson(store, "endpoints", endpoints)
					if err != nil {
						return err
					}

					return nil
				},
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

			// Load previous endpoints.
			endpoints := make(map[string]string) // key=id (mac), value=ip
			if store.Has("endpoints") {
				err := stores.GetJson(store, "endpoints", &endpoints)
				if err != nil {
					return fmt.Errorf("failed to get previous endpoints: %w", err)
				}
			}

			// Create the client.
			client := wh.NewClient(store, args.String("addr"), wh.WithClientID(args.String("id")))

			// Set up devices.
			var devices []*TasmotaDevice
			for id, ip := range endpoints {
				dev := NewTasmotaDevice(client, id, ip)
				dev.PollStatus(context.Background())
				devices = append(devices, dev)
			}

			// Start the Tasmota goroutine.
			wg := &sync.WaitGroup{}
			defer wg.Wait()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			wg.Add(1)
			go func() {
				defer wg.Done()
				ticker := time.NewTicker(2 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return

					case <-ticker.C:
						// Poll devices.
						for _, dev := range devices {
							dev.PollStatus(ctx)
						}
					}
				}
			}()

			// // Run the client.
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
