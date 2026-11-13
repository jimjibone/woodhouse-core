// whctl is a command line client for woodhouse-core. It logs in as a user and
// calls any RPC of the user-facing API with JSON requests and responses, and
// has shortcuts for the common reads and for sending actions.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app := newApp()
	if err := app.RunContext(ctx, permuteArgs(app, os.Args)); err != nil {
		fmt.Fprintf(os.Stderr, "whctl: %v\n", err)
		os.Exit(1)
	}
}

func newApp() *cli.App {
	return &cli.App{
		Name:  "whctl",
		Usage: "command line client for woodhouse-core",
		Description: "Requests and responses are JSON (protobuf JSON mapping, snake_case field names). " +
			"Unary responses print as one JSON document; server streams print one JSON document per line. " +
			"Flags may be given before or after arguments.",
		HideHelpCommand: true,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Usage: "config file (default $XDG_CONFIG_HOME/whctl/config.json or ~/.config/whctl/config.json)", EnvVars: []string{"WHCTL_CONFIG"}},
			&cli.StringFlag{Name: "profile", Aliases: []string{"p"}, Usage: "profile to use (default: the current profile)", EnvVars: []string{"WHCTL_PROFILE"}},
			&cli.BoolFlag{Name: "insecure", Usage: "do not check the server certificate against the pinned fingerprint"},
			&cli.DurationFlag{Name: "timeout", Value: 30 * time.Second, Usage: "timeout for unary calls and login"},
		},
		Commands: []*cli.Command{
			loginCommand(),
			logoutCommand(),
			statusCommand(),
			profilesCommand(),
			useCommand(),
			pingCommand(),
			listCommand(),
			describeCommand(),
			callCommand(),
			actionCommand(),
			snapshotCommand("devices", "UserService/GetDevices", "UserService/DevicesStream",
				"devices and their state; optional device IDs filter the output"),
			snapshotCommand("clients", "UserService/GetClients", "UserService/ClientsStream",
				"clients (bridges and reactors)"),
			snapshotCommand("groups", "", "UserService/GroupsStream", "groups"),
			snapshotCommand("zones", "", "UserService/ZonesStream", "zones"),
			snapshotCommand("favorites", "", "UserService/FavoritesStream", "favorite device services"),
			snapshotCommand("users", "", "UserService/UsersStream", "users"),
			snapshotCommand("notifications", "", "UserService/NotificationsStream", "notifications for the logged in user"),
			snapshotCommand("pairings", "", "UserService/PairingRequestsStream", "pending client pairing requests"),
			unaryCommand("invites", "UserService/ListInvites", "pending user invites"),
			unaryCommand("settings", "UserService/GetSettings", "server settings"),
		},
	}
}

// permuteArgs moves a command's flags ahead of its positional arguments, so
// "whctl call UserService/GetDevices --count 3" works: urfave/cli v2 stops
// parsing flags at the first positional argument. Everything after "--" is
// left alone.
func permuteArgs(app *cli.App, args []string) []string {
	if len(args) < 2 {
		return args
	}
	out := []string{args[0]}
	i := 1
	// Global flags up to the command name.
	for i < len(args) && strings.HasPrefix(args[i], "-") && args[i] != "--" {
		out = append(out, args[i])
		if takesValue(app.Flags, args[i]) && i+1 < len(args) {
			i++
			out = append(out, args[i])
		}
		i++
	}
	if i >= len(args) {
		return out
	}
	cmd := app.Command(args[i])
	out = append(out, args[i])
	i++
	if cmd == nil {
		return append(out, args[i:]...)
	}

	var flags, positional []string
	for ; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i:]...)
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			flags = append(flags, arg)
			if takesValue(cmd.Flags, arg) && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		positional = append(positional, arg)
	}
	out = append(out, flags...)
	return append(out, positional...)
}

// takesValue reports whether arg names a non-bool flag given without "=".
func takesValue(flags []cli.Flag, arg string) bool {
	name := strings.TrimLeft(arg, "-")
	if strings.Contains(name, "=") {
		return false
	}
	for _, f := range flags {
		for _, n := range f.Names() {
			if n != name {
				continue
			}
			_, isBool := f.(*cli.BoolFlag)
			return !isBool
		}
	}
	return false
}

// multiFlag is a repeatable string flag that, unlike cli.StringSliceFlag, does
// not split values on commas, which JSON values contain.
type multiFlag []string

func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }
func (m *multiFlag) String() string     { return strings.Join(*m, " ") }
