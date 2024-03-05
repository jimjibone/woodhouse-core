package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/config"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/internal"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/internal/yamlfile"
	"github.com/jimjibone/woodhouse-4/discovery"
	"github.com/jimjibone/woodhouse-4/webapp"
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
				Name:      "config",
				Usage:     "Load configuration from `DIR`",
				EnvVars:   []string{"WOODHOUSE_CONFIG"},
				Value:     "woodhouse.yaml",
				TakesFile: true,
			},
		},
		Before: func(args *cli.Context) error {
			// Load the config.
			configPath := internal.AbsPathify(args.String("config"))
			if _, err := os.Stat(configPath); !os.IsNotExist(err) {
				log.Printf("loading config from %s", configPath)
				err := yamlfile.LoadFile(&config.LoadedConfig, configPath)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}
			} else {
				log.Printf("using default config")
				err := yamlfile.SaveFile(config.LoadedConfig, configPath)
				if err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}
			}
			return nil
		},
		After: func(args *cli.Context) error {
			// Save the config if it has changed.
			configPath := internal.AbsPathify(args.String("config"))
			log.Printf("saving config to %s", configPath)
			if config.LoadedConfig.Changed {
				err := yamlfile.SaveFile(config.LoadedConfig, configPath)
				if err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}
			}
			return nil
		},
		Action: func(args *cli.Context) error {
			// Try to listen on the selected server addresses.
			apiLis, err := net.Listen("tcp", config.LoadedConfig.Server.ApiAddr)
			if err != nil {
				return fmt.Errorf("failed to listen on api addr: %w", err)
			}
			webLis, err := net.Listen("tcp", config.LoadedConfig.Server.WebAddr)
			if err != nil {
				return fmt.Errorf("failed to listen on http addr: %w", err)
			}

			// Create device store.
			deviceStore, err := NewDeviceStore(config.LoadedConfig.Stores.DeviceStoreEnabled, config.LoadedConfig.Stores.DeviceStorePath)
			if err != nil {
				return fmt.Errorf("failed to create device store: %s", err)
			}
			defer deviceStore.Close()

			// Create history store.
			historyStore := NewHistoryStore(deviceStore)
			defer historyStore.Close()

			// Create services.
			reactorService := NewReactorService(deviceStore)
			bridgeService := NewBridgeService(deviceStore, reactorService)

			// Broadcast our existence.
			broadcaster, err := discovery.NewBroadcaster("woodhouse-core", apiLis.Addr())
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

			// Run the gRPC server.
			serverErr := make(chan error, 1)
			go func() {
				log.Printf("api server ready at grpc://%s", apiLis.Addr())
				if err := server.Serve(apiLis); err != nil {
					serverErr <- fmt.Errorf("grpc server: %w", err)
				}
			}()

			// Run the web server with grpc-web support.
			webServerErr := make(chan error, 1)
			go func() {
				wrappedServer := grpcweb.WrapServer(server)
				mux := http.NewServeMux()
				publicfs, err := fs.Sub(webapp.Content, "public")
				if err != nil {
					panic(err)
				}
				mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
					// Send all other requests to the index page as we're serving a
					// single page application.
					filename := "index.html"
					f, err := publicfs.Open("index.html")
					if err != nil {
						panic(err)
					}
					ctype := mime.TypeByExtension(filepath.Ext(filename))
					rw.Header().Set("Content-Type", ctype)
					_, err = io.Copy(rw, f)
					if err != nil {
						panic(err)
					}
				})
				mux.Handle("/favicon.png", http.FileServer(http.FS(publicfs)))
				mux.Handle("/build/", http.FileServer(http.FS(publicfs)))
				mux.Handle("/api/", http.StripPrefix("/api/", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
					wrappedServer.ServeHTTP(res, req)
				})))

				httpServer := &http.Server{
					Handler: mux,
				}
				log.Printf("web server ready at http://%s", webLis.Addr())
				if err := httpServer.Serve(webLis); err != nil {
					webServerErr <- fmt.Errorf("web server: %w", err)
				}
			}()

			// Wait for exit.
			sig := make(chan os.Signal, 3)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
			select {
			case err := <-serverErr:
				return err
			case err := <-webServerErr:
				return err
			case <-sig:
				log.Printf("Exiting...")
			}

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
}
