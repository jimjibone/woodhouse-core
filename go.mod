module github.com/jimjibone/woodhouse-4

go 1.26

require (
	github.com/a-h/templ v0.2.543
	github.com/eclipse/paho.mqtt.golang v1.4.1
	github.com/fatih/color v1.16.0
	github.com/gofrs/uuid/v5 v5.0.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/gorilla/websocket v1.5.0
	github.com/grandcat/zeroconf v1.0.0
	github.com/improbable-eng/grpc-web v0.15.0
	github.com/influxdata/influxdb-client-go/v2 v2.12.1
	github.com/jimjibone/queue v0.0.0-20251004200840-d3855e27766b
	github.com/nathan-osman/go-sunrise v1.1.0
	github.com/schollz/mnemonicode v1.0.1
	github.com/schollz/pake/v3 v3.0.5
	github.com/urfave/cli/v2 v2.16.3
	google.golang.org/grpc v1.49.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.2.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/deepmap/oapi-codegen v1.8.2 // indirect
	github.com/desertbit/timer v0.0.0-20180107155436-c41aec40b27f // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/influxdata/line-protocol v0.0.0-20200327222509-2487e7298839 // indirect
	github.com/klauspost/compress v1.11.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/miekg/dns v1.1.27 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rs/cors v1.8.3 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/tscholl2/siec v0.0.0-20210707234609-9bdfc483d499 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/image v0.41.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.14.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto v0.0.0-20210126160654-44e461bb6506 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	nhooyr.io/websocket v1.8.6 // indirect
)

replace (
	github.com/jimjibone/woodhouse-api => ./api
)
