<p align="center">
  <img src="logo/icon26.svg" alt="Woodhouse" width="120">
</p>

# Woodhouse

Woodhouse is a home automation system built for people who want their home to
respond instantly, run reliably for years, and do exactly what they tell it to —
no more, no less.

At its heart is `woodhouse-core`: a lightweight gRPC server that ties everything
together. Devices connect through **bridges** (for Zigbee, Shelly, Tasmota and
more), users control and observe their home through the built-in web interface,
and automations run as **reactors** - small programs that react to the state of
your home in real time.

This repository contains the core server and its admin web interface.

## Why Woodhouse

- **Fast.** Built with Go around a streaming gRPC API, Woodhouse reacts to device
  changes the moment they happen. State flows from your devices to the core and
  out to clients with minimal latency, so lights, heating and sensors feel
  immediate rather than laggy.

- **Reliable.** The core is a single, self-contained server binary with no
  sprawling dependency tree to break. Bridges and reactors connect as authenticated
  clients and reconnect automatically, so a flaky network or a restarted
  component never takes the whole system down. It is designed to be installed
  once and left to run.

- **Automation as code.** Real automation outgrows drag-and-drop editors.
  Woodhouse lets you write your automations as ordinary Go programs using the
  `wh` client library - complete with schedules, state, and the full power of a
  general-purpose language. A reactor can be anything from a simple "turn off the
  lights at midnight" rule to a complete, schedule-driven, multi-room heating
  controller. Your automation logic lives in version control, is testable, and is
  yours to extend without limits.

## How it fits together

| Component        | Role                                                                     |
| ---------------- | ------------------------------------------------------------------------ |
| `woodhouse-core` | The central server and admin web UI (this repo).                         |
| Bridges          | Connect physical devices (Zigbee, Shelly, Tasmota, ...) to the core.     |
| Reactors         | Code-driven automations that react to and control devices.               |
| `wh`             | Go client library used to build bridges and reactors.                    |
| `woodhouse-api`  | The gRPC/protobuf API shared by the core, bridges, reactors and clients. |

## Getting started

### Prerequisites

- [Go](https://go.dev/dl/) 1.26 or newer
- [Task](https://taskfile.dev/) (`go-task`) for the build and run commands
- [Node.js](https://nodejs.org/) and npm, to build the admin web interface
- [Protocol Buffers](https://protobuf.dev/) (`protoc`) to regenerate API code

### Clone

The API lives in a separate repository and is included as a git submodule, so
clone recursively:

```sh
git clone --recurse-submodules https://github.com/jimjibone/woodhouse-core.git
cd woodhouse-core
```

If you have already cloned without submodules, run:

```sh
git submodule update --init --recursive
```

### Build and run

The quickest way to get a server running is the `run-core` task, which builds the
core and starts it:

```sh
task run-core
```

For a full build that also generates the API code and bundles the latest admin
web interface, use:

```sh
task build-core-full
```

The resulting binary is written to `build-<os>-<arch>/woodhouse-core`. You can
run it directly and pass flags after `--`, for example to enable debug logging:

```sh
task run-core -- --debug
```

On first start the core writes a default config file (`woodhouse.yaml`) and a
data directory (`woodhouse.db`) into the working directory.

### Configuration

Configuration is a small YAML file. The defaults are:

```yaml
server:
    api-addr: localhost:4000 # gRPC API for bridges, reactors and clients
    web-addr: localhost:4080 # admin web interface
```

Point the server at a different config file with `--config`, or via the
`WOODHOUSE_CONFIG` environment variable.

### First run

1. Start the core with `task run-core`.
2. Open the admin web interface at <http://localhost:4080> and create your user
   account.
3. Connect a bridge (Zigbee, Shelly, Tasmota, …) to start discovering devices.
   Bridges pair with the core and must be approved before they can connect.
4. Write a reactor against the [`wh`](https://github.com/jimjibone/wh) client
   library to start automating your home.

## Writing a reactor

A reactor is just a Go program. It connects to the core as a client, grabs
handles to the devices it cares about, and reacts to their changes. Here is a
complete reactor that turns a light on whenever motion is detected:

```go
package main

import (
	"context"

	"github.com/jimjibone/log"
	"github.com/jimjibone/wh/v1"
	"github.com/jimjibone/wh/v1/reactors"
	"github.com/jimjibone/wh/v1/shared/stores"
)

func main() {
	// Connect to the core as a reactor client.
	store := stores.NewFSStore("reactor.db")
	client := reactors.NewClient(
		store,
		"localhost:4000",
		wh.WithClientID("motion-light"),
		wh.WithClientInfo("Motion Light", "Turns a light on when motion is detected", "1.0.0"),
	)

	go func() {
		// Grab typed handles to the devices we care about.
		motion := client.GetMotion("0x00124b002247ae1b")
		light := client.GetLightbulb("0x90395efffe3767b6")

		// Wait until the client is connected and devices are known.
		<-client.Ready()
		log.Infof("motion-light reactor started")

		// React to motion: light follows the sensor.
		motion.OnUpdate(func(changed bool) {
			if err := light.SetOn(context.Background(), motion.Motion()); err != nil {
				log.Errorf("failed to set light: %s", err)
			}
		})
	}()

	if err := client.Run(); err != nil {
		log.Fatalln(err)
	}
}
```

The reactor client gives you typed accessors for each kind of device service —
`GetMotion`, `GetLightbulb`, `GetButton`, `GetRelay`, `GetClimate` and more — each
with the methods and `OnUpdate` callbacks that make sense for it. Richer requests
go through helpers like `light.Request(ctx, reactors.LightbulbRequest{...})`, and
the `schedule` package adds time- and sun-based scheduling for automations that
follow the day.

The first time a reactor connects it must be approved in the admin web interface,
just like a bridge. From there it can read live device state and issue commands
with the full power of Go — timers, schedules, external APIs, persistent state
and more. Reactors can also publish their own virtual devices back to the core,
so your automations show up and can be controlled right alongside physical
hardware.

## History

This repository was the original home to woodhouse-api, wh, and numerous bridges.
Before making the project public the mono-repo was split into multiple repos to:

- Make it easier to apply different licensing (i.e. `woodhouse-core` is AGPLv3
  to protect the open source roots of the core project, `woodhouse-api` and `wh`
  are Apache v2 to allow easier integration into other projects).
- Make it easier to import the API and wh module into other projects.
- Allow client development to be independent from the core and other clients.
