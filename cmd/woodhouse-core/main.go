package main

import (
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/clients"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/config"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/internal/yamlfile"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/users"
	"github.com/jimjibone/woodhouse-4/discovery"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/cert"
	"github.com/jimjibone/woodhouse-4/shared/paths"
	"github.com/jimjibone/woodhouse-4/shared/stores"
	"github.com/jimjibone/woodhouse-4/webapp"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

func main() {
	app := &cli.App{
		Name:                 "woodhouse-core",
		Usage:                "Runs the woodhouse core",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "config",
				Usage:     "Load configuration from `FILE`",
				EnvVars:   []string{"WOODHOUSE_CONFIG"},
				Value:     "woodhouse.yaml",
				TakesFile: true,
			},
			&cli.StringFlag{
				Name:      "config-dir",
				Usage:     "Configuration directory",
				EnvVars:   []string{"WOODHOUSE_CONFIG_DIR"},
				Value:     "woodhouse.db",
				TakesFile: true,
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"v"},
				Usage:   "Enable debug logging",
			},
		},
		Before: func(args *cli.Context) error {
			// Setup logging.
			if args.Bool("debug") {
				log.SetOptions(log.WithMinLevel(log.DebugLevel))
			} else {
				log.SetOptions(log.WithMinLevel(log.InfoLevel))
			}

			log.Infof("woodhouse is starting")

			// Load the config.
			configPath := paths.AbsPathify(args.String("config"))
			if _, err := os.Stat(configPath); !os.IsNotExist(err) {
				log.Infof("loading config from %s", configPath)
				err := yamlfile.LoadFile(&config.LoadedConfig, configPath)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}
			} else {
				log.Infof("using default config")
				err := yamlfile.SaveFile(config.LoadedConfig, configPath)
				if err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}
			}
			if err := config.LoadedConfig.Verify(); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			return nil
		},
		After: func(args *cli.Context) error {
			// Save the config if it has changed.
			configPath := paths.AbsPathify(args.String("config"))
			if config.LoadedConfig.Changed {
				log.Infof("saving config to %s", configPath)
				err := yamlfile.SaveFile(config.LoadedConfig, configPath)
				if err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}
			}

			log.Infof("woodhouse finished")

			return nil
		},
		Action: func(args *cli.Context) error {
			// Try to listen on the selected server addresses.
			apiLis, err := net.Listen("tcp", config.LoadedConfig.Server.ApiAddr)
			if err != nil {
				return fmt.Errorf("failed to listen on api addr: %w", err)
			}
			insecureApiLis, err := net.Listen("tcp", "127.0.0.1:4001")
			if err != nil {
				return fmt.Errorf("failed to listen on insecure api addr: %w", err)
			}
			webLis, err := net.Listen("tcp", config.LoadedConfig.Server.WebAddr)
			if err != nil {
				return fmt.Errorf("failed to listen on http addr: %w", err)
			}

			// Create the config store.
			store := stores.NewFSStore(args.Path("config-dir"))

			certManager, err := cert.NewCertManager(store)
			if err != nil {
				return fmt.Errorf("failed to create cert manager: %s", err)
			}
			clientJwtManager, err := clients.NewJWTManager(store)
			if err != nil {
				return fmt.Errorf("failed to create jwt manager: %s", err)
			}
			defer clientJwtManager.Close()
			clientAuthService := clients.NewAuthService(certManager, clientJwtManager)

			authInterceptor := clients.NewAuthInterceptor(clientJwtManager)

			deviceManager, err := core.NewDeviceManager(store)
			if err != nil {
				return fmt.Errorf("failed to create device manager: %s", err)
			}
			defer deviceManager.Close()

			// Create services.
			clientService := clients.NewClientService(deviceManager)
			userService := users.NewUserService(deviceManager)

			// Broadcast our existence.
			broadcaster, err := discovery.NewBroadcaster("woodhouse-core", apiLis.Addr())
			if err != nil {
				return fmt.Errorf("failed to create broadcaster: %w", err)
			}
			defer broadcaster.Shutdown()

			// Create the gRPC server.
			creds := credentials.NewServerTLSFromCert(certManager.Cert())
			server := grpc.NewServer(
				grpc.Creds(creds),
				grpc.UnaryInterceptor(authInterceptor.Unary()),
				grpc.StreamInterceptor(authInterceptor.Stream()),
			)
			insecureServer := grpc.NewServer()

			// Register services.
			clientsapi.RegisterAuthServiceServer(server, clientAuthService)
			clientsapi.RegisterClientServiceServer(server, clientService)
			clientsapi.RegisterUserServiceServer(insecureServer, userService)
			reflection.Register(server)
			clientsapi.RegisterAuthServiceServer(insecureServer, clientAuthService)
			reflection.Register(insecureServer)

			// Run the gRPC server.
			serverErr := make(chan error, 1)
			go func() {
				log.Infof("api server ready at grpc://%s", apiLis.Addr())
				if err := server.Serve(apiLis); err != nil {
					serverErr <- fmt.Errorf("grpc server: %w", err)
				}
				server.GracefulStop()
			}()
			go func() {
				log.Infof("insecure api server ready at grpc://%s", insecureApiLis.Addr())
				if err := insecureServer.Serve(insecureApiLis); err != nil {
					serverErr <- fmt.Errorf("grpc insecure server: %w", err)
				}
				insecureServer.GracefulStop()
			}()

			// Run the web server with grpc-web support.
			webServerErr := make(chan error, 1)
			go func() {
				wrappedServer := grpcweb.WrapServer(insecureServer)
				mux := http.NewServeMux()
				buildfs, err := fs.Sub(webapp.Content, "build")
				if err != nil {
					panic(err)
				}
				mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
					// Send all other requests to the index page as we're serving a
					// single page application.
					filename := "index.html"
					f, err := buildfs.Open("index.html")
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
				mux.Handle("/manifest.json", http.FileServer(http.FS(buildfs)))
				mux.Handle("/service-worker.js", http.FileServer(http.FS(buildfs)))
				mux.Handle("/favicon.png", http.FileServer(http.FS(buildfs)))
				mux.Handle("/favicon-128.png", http.FileServer(http.FS(buildfs)))
				mux.Handle("/favicon-256.png", http.FileServer(http.FS(buildfs)))
				mux.Handle("/favicon-512.png", http.FileServer(http.FS(buildfs)))
				mux.Handle("/apple-touch-icon.png", http.FileServer(http.FS(buildfs)))
				mux.Handle("/_app/", http.FileServer(http.FS(buildfs)))
				mux.Handle("/api/", http.StripPrefix("/api/", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
					wrappedServer.ServeHTTP(res, req)
				})))

				httpServer := &http.Server{
					Handler: mux,
				}
				log.Infof("web server ready at http://%s", webLis.Addr())
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
				log.Infof("exiting...")
			}

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
}
