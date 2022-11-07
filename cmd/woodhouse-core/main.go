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
				Name:  "addr",
				Usage: "woodhouse api server address",
				Value: ":4000",
			},
			&cli.StringFlag{
				Name:  "http",
				Usage: "woodhouse web server address",
				Value: ":4080",
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
			apiLis, err := net.Listen("tcp", c.String("addr"))
			if err != nil {
				return fmt.Errorf("failed to listen on api addr: %w", err)
			}
			webLis, err := net.Listen("tcp", c.String("http"))
			if err != nil {
				return fmt.Errorf("failed to listen on http addr: %w", err)
			}

			// Create services.
			deviceStore := NewDeviceStore()
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
