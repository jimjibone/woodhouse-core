package main

import (
	"crypto/tls"
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
	"github.com/jimjibone/log"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/clients"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/config"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/limits"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/yamlfile"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/users"
	"github.com/jimjibone/woodhouse-core/discovery"
	"github.com/jimjibone/woodhouse-core/shared/cert"
	"github.com/jimjibone/woodhouse-core/shared/paths"
	"github.com/jimjibone/woodhouse-core/shared/stores"
	"github.com/jimjibone/woodhouse-core/webui"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
			config.SetPath(configPath)
			if _, err := os.Stat(configPath); !os.IsNotExist(err) {
				log.Infof("loading config from %s", configPath)
				err := yamlfile.LoadFile(&config.LoadedConfig, configPath)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}
			} else {
				log.Infof("using default config")
				// Written below, after Verify has filled in the defaults it
				// derives. Saving here would persist an empty instance-name.
				config.LoadedConfig.Changed = true
			}
			if err := config.LoadedConfig.Verify(); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			if config.LoadedConfig.Changed {
				log.Infof("saving config to %s", configPath)
				if err := config.Save(); err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}
			}
			log.Infof("instance name is %q", config.InstanceName())
			return nil
		},
		After: func(args *cli.Context) error {
			// Save the config if it has changed. Runtime changes save
			// themselves as they happen, so this is only a backstop.
			if config.LoadedConfig.Changed {
				log.Infof("saving config to %s", config.Path())
				if err := config.Save(); err != nil {
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
				return fmt.Errorf("failed to create client jwt manager: %s", err)
			}
			defer clientJwtManager.Close()

			clientManager, err := core.NewClientManager(store)
			if err != nil {
				return fmt.Errorf("failed to create client manager: %s", err)
			}
			defer clientManager.Close()

			// Pairing is human-paced: the 32-slot pool with 3-min expiry
			// needs ~11/min sustained to monopolize, so 3/min per source
			// makes that impossible from one address.
			pairLimits := limits.NewIPRate(limits.IPRateConfig{PerMin: 3, Burst: 5, Penalty: 3})
			// wh bridges keepalive-ping every 5s and several can share one
			// source IP; this supports ~20 bridges per IP with headroom.
			pingLimits := limits.NewIPRate(limits.IPRateConfig{PerMin: 240, Burst: 60})
			clientAuthService := clients.NewAuthService(certManager, clientJwtManager, clientManager, pairLimits, pingLimits)

			deviceManager, err := core.NewDeviceManager(store)
			if err != nil {
				return fmt.Errorf("failed to create device manager: %s", err)
			}
			defer deviceManager.Close()

			favoritesManager := core.NewFavoritesManager(store, deviceManager)
			defer favoritesManager.Close()

			groupManager, err := core.NewGroupManager(store, deviceManager)
			if err != nil {
				return fmt.Errorf("failed to create group manager: %s", err)
			}
			defer groupManager.Close()

			userManager, err := core.NewUserManager(store)
			if err != nil {
				return fmt.Errorf("failed to create user manager: %s", err)
			}
			defer userManager.Close()

			userJwtManager, err := users.NewJWTManager(store)
			if err != nil {
				return fmt.Errorf("failed to create user jwt manager: %s", err)
			}
			defer userJwtManager.Close()
			loginLimits := limits.NewLogin(limits.Config{})
			userAuthService := users.NewAuthService(userManager, userJwtManager, loginLimits)

			// Broadcast our existence. This is created before the services
			// because the settings manager renames the advertisement live when
			// a user changes the instance name.
			broadcaster, err := discovery.NewBroadcaster(config.InstanceName(), apiLis.Addr())
			if err != nil {
				return fmt.Errorf("failed to create broadcaster: %w", err)
			}
			defer broadcaster.Shutdown()

			settingsManager := core.NewSettingsManager(broadcaster)

			// Create services.
			clientService := clients.NewClientService(deviceManager, clientManager, clientJwtManager)
			userService := users.NewUserService(deviceManager, favoritesManager, groupManager, userManager, clientManager, settingsManager, clientJwtManager, userJwtManager)

			// Create the gRPC server.
			creds := credentials.NewServerTLSFromCert(certManager.Cert())
			authInterceptor := NewAuthInterceptor(clientJwtManager, userJwtManager, clientManager, userManager)
			server := grpc.NewServer(
				grpc.Creds(creds),
				grpc.UnaryInterceptor(authInterceptor.Unary()),
				grpc.StreamInterceptor(authInterceptor.Stream()),
			)

			// Register services.
			clientsapi.RegisterAuthServiceServer(server, clientAuthService)
			clientsapi.RegisterClientServiceServer(server, clientService)
			clientsapi.RegisterUserServiceServer(server, userService)
			clientsapi.RegisterUserAuthServiceServer(server, userAuthService)

			// Cross-check the authorization policy against what was actually
			// registered above. Stale entries for removed RPCs are dead data
			// that reads like a live capability, and a new RPC with no policy
			// entry fails closed silently — both are startup errors, not
			// something to discover in a later audit.
			var registeredMethods []string
			for name, info := range server.GetServiceInfo() {
				for _, method := range info.Methods {
					registeredMethods = append(registeredMethods, "/"+name+"/"+method.Name)
				}
			}
			if err := auth.VerifyPolicyCoverage(registeredMethods); err != nil {
				return fmt.Errorf("authorization policy: %w", err)
			}

			// Server reflection lets any peer that completes TLS enumerate the
			// full service schema, before authenticating. Only useful for
			// debugging tools.
			// log.Warnf("gRPC server reflection enabled: the API schema is readable by any peer that completes TLS")
			// reflection.Register(server)

			// Run the gRPC server.
			serverErr := make(chan error, 1)
			go func() {
				log.Infof("api server ready at grpc://%s", apiLis.Addr())
				if err := server.Serve(apiLis); err != nil {
					serverErr <- fmt.Errorf("grpc server: %w", err)
				}
				server.GracefulStop()
			}()

			// Run the web server with grpc-web support.
			webServerErr := make(chan error, 1)
			go func() {
				wrappedServer := grpcweb.WrapServer(server)
				mux := http.NewServeMux()
				buildfs, err := fs.Sub(webui.Content, "build")
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
				mux.HandleFunc("/api/login", userAuthService.LoginWeb)
				mux.HandleFunc("/api/refresh", userAuthService.RefreshWeb)
				mux.HandleFunc("/api/logout", userAuthService.LogoutWeb)
				// Note that we don't strip the last `/` from the api path as
				// this is required to remain a valid gRPC method call (all must
				// start with `/`).
				mux.Handle("/api/", http.StripPrefix("/api", http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
					if wrappedServer.IsGrpcWebRequest(req) || wrappedServer.IsAcceptableGrpcCorsRequest(req) {
						wrappedServer.ServeHTTP(res, req)
						return
					}
					http.NotFound(res, req)
				})))

				httpServer := &http.Server{
					Handler: mux,
					TLSConfig: &tls.Config{
						Certificates: []tls.Certificate{*certManager.Cert()},
						MinVersion:   tls.VersionTLS12,
					},
				}
				log.Infof("web server ready at https://%s", webLis.Addr())
				if err := httpServer.ServeTLS(webLis, "", ""); err != nil {
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
