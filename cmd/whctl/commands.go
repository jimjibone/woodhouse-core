package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
	"google.golang.org/protobuf/proto"
)

func loadConfig(c *cli.Context) (*Config, error) {
	path := c.String("config")
	if path == "" {
		var err error
		if path, err = defaultConfigPath(); err != nil {
			return nil, err
		}
	}
	return LoadConfig(path)
}

func openSession(c *cli.Context) (*Session, error) {
	cfg, err := loadConfig(c)
	if err != nil {
		return nil, err
	}
	name := cfg.ProfileName(c.String("profile"))
	prof, err := cfg.Profile(name)
	if err != nil {
		return nil, err
	}
	return Dial(cfg, name, prof, c.Bool("insecure"), c.Duration("timeout"))
}

func outputFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "pretty", Usage: "indent JSON output"},
		&cli.BoolFlag{Name: "defaults", Usage: "include fields that hold their default value"},
	}
}

func streamFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{Name: "count", Aliases: []string{"n"}, Usage: "stop a stream after `N` responses (heartbeats not counted)"},
		&cli.DurationFlag{Name: "duration", Aliases: []string{"for"}, Usage: "stop a stream after this long"},
		&cli.DurationFlag{Name: "idle", Usage: "stop a stream when nothing arrives for this long"},
		&cli.BoolFlag{Name: "snapshot", Usage: "stop a stream at its first heartbeat, i.e. after the initial state"},
		&cli.BoolFlag{Name: "heartbeats", Usage: "also print heartbeat responses"},
	}
}

func flags(groups ...[]cli.Flag) []cli.Flag {
	var out []cli.Flag
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

func streamOpts(c *cli.Context) StreamOpts {
	return StreamOpts{
		Count:      c.Int("count"),
		Duration:   c.Duration("duration"),
		Idle:       c.Duration("idle"),
		Snapshot:   c.Bool("snapshot"),
		Heartbeats: c.Bool("heartbeats"),
	}
}

func printer(c *cli.Context) *Printer {
	return NewPrinter(os.Stdout, c.Bool("pretty"), c.Bool("defaults"))
}

// readData resolves a request argument: inline JSON, "@path" for a file, or
// "-" / "@-" for stdin.
func readData(arg string) ([]byte, error) {
	switch {
	case arg == "-" || arg == "@-":
		return io.ReadAll(os.Stdin)
	case strings.HasPrefix(arg, "@"):
		return os.ReadFile(arg[1:])
	}
	return []byte(arg), nil
}

func callCommand() *cli.Command {
	return &cli.Command{
		Name:      "call",
		Usage:     "call any RPC with a JSON request",
		ArgsUsage: "METHOD [JSON|@FILE|-]",
		Description: "METHOD is Service/Method, or just Method when unique (see 'whctl list --all'). " +
			"The request defaults to {}. A server stream runs until the server ends it, a limit is reached or Ctrl-C.\n\n" +
			"Examples:\n" +
			"   whctl call UserService/GetSettings\n" +
			"   whctl call DevicesStream '{\"include_device_ids\":[\"zigbee00158d00045c089f\"]}' --duration 30s\n" +
			"   whctl call ZonesStream --snapshot\n" +
			"   whctl call AddZone @zone.json",
		Flags: flags([]cli.Flag{
			&cli.StringFlag{Name: "data", Aliases: []string{"d"}, Usage: "request as JSON, @FILE or - for stdin"},
		}, outputFlags(), streamFlags()),
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 || c.NArg() > 2 {
				return cli.ShowSubcommandHelp(c)
			}
			md, err := findMethod(c.Args().Get(0))
			if err != nil {
				return err
			}
			arg := c.String("data")
			if c.NArg() == 2 {
				if arg != "" {
					return errors.New("give the request either as an argument or with --data, not both")
				}
				arg = c.Args().Get(1)
			}
			data, err := readData(arg)
			if err != nil {
				return err
			}
			req, err := parseRequest(md, data)
			if err != nil {
				return err
			}

			sess, err := openSession(c)
			if err != nil {
				return err
			}
			defer sess.Close()
			p := printer(c)
			return sess.Call(c.Context, md, req, streamOpts(c), p.Print)
		},
	}
}

// snapshotCommand reads a collection. Without --watch it prints the current
// state and exits, using get (a stream the server ends) if there is one, else
// watch up to its first heartbeat, which every stream sends straight after
// its initial state. With --watch it follows watch for updates.
func snapshotCommand(name, get, watch, usage string) *cli.Command {
	isDevices := name == "devices"
	argsUsage := ""
	if isDevices {
		argsUsage = "[DEVICE_ID...]"
	}
	return &cli.Command{
		Name:      name,
		Usage:     "print " + usage + " (--watch to follow updates)",
		ArgsUsage: argsUsage,
		Flags: flags([]cli.Flag{
			&cli.BoolFlag{Name: "watch", Aliases: []string{"w"}, Usage: "keep streaming updates after the current state"},
		}, outputFlags(), streamFlags()),
		Action: func(c *cli.Context) error {
			if !isDevices && c.NArg() > 0 {
				return cli.ShowSubcommandHelp(c)
			}
			ids := map[string]bool{}
			for _, id := range c.Args().Slice() {
				ids[id] = true
			}

			opts := streamOpts(c)
			method := watch
			if !c.Bool("watch") {
				if get != "" {
					method = get
				} else {
					opts.Snapshot = true
				}
			}
			md, err := findMethod(method)
			if err != nil {
				return err
			}
			req, err := newMessage(md.Input())
			if err != nil {
				return err
			}
			if r, ok := req.(*clientsapi.DevicesStreamRequest); ok {
				r.IncludeDeviceIds = c.Args().Slice()
			}

			sess, err := openSession(c)
			if err != nil {
				return err
			}
			defer sess.Close()
			p := printer(c)
			return sess.Call(c.Context, md, req, opts, func(m proto.Message) error {
				if d, ok := m.(*clientsapi.Device); ok && len(ids) > 0 && !ids[d.Id] {
					return nil
				}
				return p.Print(m)
			})
		},
	}
}

func unaryCommand(name, method, usage string) *cli.Command {
	return &cli.Command{
		Name:  name,
		Usage: "print " + usage,
		Flags: outputFlags(),
		Action: func(c *cli.Context) error {
			md, err := findMethod(method)
			if err != nil {
				return err
			}
			req, err := newMessage(md.Input())
			if err != nil {
				return err
			}
			sess, err := openSession(c)
			if err != nil {
				return err
			}
			defer sess.Close()
			p := printer(c)
			return sess.Call(c.Context, md, req, StreamOpts{}, p.Print)
		},
	}
}

func actionCommand() *cli.Command {
	return &cli.Command{
		Name:      "action",
		Usage:     "write attributes of, or invoke a command on, a device service",
		ArgsUsage: "DEVICE_ID ENDPOINT_ID SERVICE_ID",
		Description: "Values are typed from the service's attribute and command specs, which are read from the device first. " +
			"A value written as Value JSON is sent as is, which is how structured values are given.\n" +
			"Prints each ActionResponse and fails unless the final status is COMPLETE.\n\n" +
			"Examples:\n" +
			"   whctl action zigbee00158d00045c089f 1 on_off --set on=true\n" +
			"   whctl action zigbee00158d00045c089f 1 light --set 'color={\"hue_sat\":{\"hue\":120,\"sat\":100}}'\n" +
			"   whctl action zigbee00158d00045c089f 0 identify --command identify --param duration=5s",
		Flags: flags([]cli.Flag{
			&cli.GenericFlag{Name: "set", Value: &multiFlag{}, Usage: "write attribute `ID=VALUE` (repeatable)"},
			&cli.StringFlag{Name: "command", Aliases: []string{"c"}, Usage: "invoke command `ID`"},
			&cli.GenericFlag{Name: "param", Value: &multiFlag{}, Usage: "command parameter `ID=VALUE` (repeatable)"},
		}, outputFlags()),
		Action: func(c *cli.Context) error {
			if c.NArg() != 3 {
				return cli.ShowSubcommandHelp(c)
			}
			deviceID, endpointID, serviceID := c.Args().Get(0), c.Args().Get(1), c.Args().Get(2)
			sets, err := parseAssignments(*c.Generic("set").(*multiFlag))
			if err != nil {
				return err
			}
			params, err := parseAssignments(*c.Generic("param").(*multiFlag))
			if err != nil {
				return err
			}
			command := c.String("command")
			switch {
			case command == "" && len(sets) == 0:
				return errors.New("give --set or --command")
			case command != "" && len(sets) > 0:
				return errors.New("give --set or --command, not both")
			case command == "" && len(params) > 0:
				return errors.New("--param needs --command")
			}

			sess, err := openSession(c)
			if err != nil {
				return err
			}
			defer sess.Close()

			ctx, cancel := context.WithTimeout(c.Context, c.Duration("timeout"))
			defer cancel()

			// Look the service up only when a value needs typing or the
			// command needs checking.
			var srv *clientsapi.Service
			if !allRaw(sets) || command != "" && !allRaw(params) {
				if srv, err = sess.findDeviceService(ctx, deviceID, endpointID, serviceID); err != nil {
					return err
				}
			}

			req := &clientsapi.ActionRequest{DeviceId: deviceID, EndpointId: endpointID, ServiceId: serviceID}
			if command != "" {
				cmd, err := buildCommand(srv, command, params)
				if err != nil {
					return err
				}
				req.Action = &clientsapi.ActionRequest_Command{Command: cmd}
			} else {
				w, err := buildWrite(srv, sets)
				if err != nil {
					return err
				}
				req.Action = &clientsapi.ActionRequest_Write{Write: w}
			}

			md, err := findMethod("UserService/SendAction")
			if err != nil {
				return err
			}
			p := printer(c)
			var last *clientsapi.ActionResponse
			err = sess.Call(ctx, md, req, StreamOpts{}, func(m proto.Message) error {
				last = m.(*clientsapi.ActionResponse)
				return p.Print(m)
			})
			if err != nil {
				return err
			}
			if ctx.Err() != nil {
				return fmt.Errorf("no final action status after %s", c.Duration("timeout"))
			}
			if last == nil {
				return errors.New("no action status received")
			}
			if last.Status != clientsapi.ActionResponse_COMPLETE {
				if last.Details != "" {
					return fmt.Errorf("action %s: %s", last.Status, last.Details)
				}
				return fmt.Errorf("action %s", last.Status)
			}
			return nil
		},
	}
}

func listCommand() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "list services, or the methods of one",
		ArgsUsage: "[SERVICE]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "all", Aliases: []string{"a"}, Usage: "list the methods of every service"},
		},
		Action: func(c *cli.Context) error {
			if c.Bool("all") {
				listAllMethods(os.Stdout)
				return nil
			}
			return listServices(os.Stdout, c.Args().First())
		},
	}
}

func describeCommand() *cli.Command {
	return &cli.Command{
		Name:      "describe",
		Usage:     "show a method's request and response messages, or a message or enum",
		ArgsUsage: "METHOD|MESSAGE|ENUM",
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return cli.ShowSubcommandHelp(c)
			}
			return describe(os.Stdout, c.Args().First())
		},
	}
}

func pingCommand() *cli.Command {
	return &cli.Command{
		Name:  "ping",
		Usage: "check the server is reachable, its certificate matches and the schema versions agree",
		Action: func(c *cli.Context) error {
			sess, err := openSession(c)
			if err != nil {
				return err
			}
			defer sess.Close()
			md, err := findMethod("AuthService/Ping")
			if err != nil {
				return err
			}
			start := time.Now()
			if err := sess.Call(c.Context, md, &clientsapi.PingRequest{}, StreamOpts{}, func(proto.Message) error { return nil }); err != nil {
				return err
			}
			return printValue(map[string]any{
				"server": sess.prof.Server,
				"rtt":    time.Since(start).Round(time.Microsecond).String(),
			})
		},
	}
}

func printValue(v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return NewPrinter(os.Stdout, false, false).PrintJSON(raw)
}

func loginCommand() *cli.Command {
	return &cli.Command{
		Name:  "login",
		Usage: "log in to a server and save the session in the profile",
		Description: "On the first login to a server its certificate fingerprint is pinned. It is shown for confirmation, " +
			"or can be given with --fingerprint (core advertises it as \"fp\" in its _woodhouse._tcp TXT record). " +
			"Later logins and calls fail if the certificate changes.\n\n" +
			"The password is read from the terminal, or from stdin with --password-stdin.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "server", Aliases: []string{"s"}, Usage: "API address `HOST[:PORT]` (port defaults to " + defaultPort + ")"},
			&cli.StringFlag{Name: "username", Aliases: []string{"u"}},
			&cli.BoolFlag{Name: "password-stdin", Usage: "read the password from stdin"},
			&cli.StringFlag{Name: "fingerprint", Usage: "expected SHA-256 `HEX` of the server certificate"},
			&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "trust the presented certificate without asking"},
		},
		Action: func(c *cli.Context) error {
			cfg, err := loadConfig(c)
			if err != nil {
				return err
			}
			name := cfg.ProfileName(c.String("profile"))
			prof := &Profile{}
			if existing := cfg.Profiles[name]; existing != nil {
				copied := *existing
				prof = &copied
			}

			server := c.String("server")
			if server == "" {
				server = prof.Server
			}
			if server == "" {
				return errors.New("--server is required for a new profile")
			}
			server = normaliseServer(server)
			if server != prof.Server {
				*prof = Profile{Server: server, Username: prof.Username}
			}
			prof.AccessToken = ""

			insecure := c.Bool("insecure")
			want := normaliseFingerprint(c.String("fingerprint"))
			if !insecure && (want != "" || prof.Fingerprint == "") {
				ctx, cancel := context.WithTimeout(c.Context, c.Duration("timeout"))
				got, err := probeFingerprint(ctx, server)
				cancel()
				if err != nil {
					return err
				}
				switch {
				case want != "":
					if got != want {
						return fmt.Errorf("server presented certificate %s, expected %s", got, want)
					}
				case c.Bool("yes"):
				default:
					if err := confirmFingerprint(server, got); err != nil {
						return err
					}
				}
				prof.Fingerprint = got
			}

			username := c.String("username")
			if username == "" {
				username = prof.Username
			}
			if username == "" {
				if c.Bool("password-stdin") || !term.IsTerminal(int(os.Stdin.Fd())) {
					return errors.New("--username is required")
				}
				if username, err = prompt("Username: "); err != nil {
					return err
				}
			}
			password, err := readPassword(c.Bool("password-stdin"))
			if err != nil {
				return err
			}

			sess, err := Dial(cfg, name, prof, insecure, c.Duration("timeout"))
			if err != nil {
				return err
			}
			defer sess.Close()
			ctx, cancel := context.WithTimeout(c.Context, c.Duration("timeout"))
			defer cancel()
			res, err := clientsapi.NewUserAuthServiceClient(sess.conn).Login(ctx, &clientsapi.UserLoginRequest{
				Username: username,
				Password: password,
			})
			if err != nil {
				return fmt.Errorf("login: %w", err)
			}

			prof.Username = username
			prof.RefreshToken = res.RefreshToken
			prof.AccessToken = res.AccessToken
			cfg.Profiles[name] = prof
			cfg.Current = name
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "logged in to %s as %s (profile %q)\n", server, username, name)
			return nil
		},
	}
}

func confirmFingerprint(server, fp string) error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("%s presented certificate %s; pass --fingerprint %s (or --yes) to trust it", server, fp, fp)
	}
	fmt.Fprintf(os.Stderr, "%s presented certificate SHA-256 %s\n", server, fp)
	answer, err := prompt("Trust it? [y/N] ")
	if err != nil {
		return err
	}
	if a := strings.ToLower(answer); a != "y" && a != "yes" {
		return errors.New("certificate not trusted")
	}
	return nil
}

func prompt(label string) (string, error) {
	fmt.Fprint(os.Stderr, label)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !(errors.Is(err, io.EOF) && line != "") {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func readPassword(fromStdin bool) (string, error) {
	if fromStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(data), "\r\n"), nil
	}
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", errors.New("no terminal to read the password from; use --password-stdin")
	}
	fmt.Fprint(os.Stderr, "Password: ")
	pw, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(pw), nil
}

func logoutCommand() *cli.Command {
	return &cli.Command{
		Name:  "logout",
		Usage: "end the profile's session on the server and drop its tokens",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "forget", Usage: "also delete the profile, including its pinned certificate"},
		},
		Action: func(c *cli.Context) error {
			sess, err := openSession(c)
			if err != nil {
				return err
			}
			defer sess.Close()
			if sess.prof.RefreshToken != "" {
				ctx, cancel := context.WithTimeout(c.Context, c.Duration("timeout"))
				_, err := clientsapi.NewUserAuthServiceClient(sess.conn).Logout(ctx, &clientsapi.UserLogoutRequest{RefreshToken: sess.prof.RefreshToken})
				cancel()
				if err != nil {
					fmt.Fprintf(os.Stderr, "whctl: server logout failed, dropping tokens anyway: %v\n", err)
				}
			}
			sess.prof.RefreshToken = ""
			sess.prof.AccessToken = ""
			if c.Bool("forget") {
				delete(sess.cfg.Profiles, sess.name)
				if sess.cfg.Current == sess.name {
					sess.cfg.Current = ""
				}
			}
			return sess.cfg.Save()
		},
	}
}

func statusCommand() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "show the profile's server, user and token expiry (no network)",
		Action: func(c *cli.Context) error {
			cfg, err := loadConfig(c)
			if err != nil {
				return err
			}
			name := cfg.ProfileName(c.String("profile"))
			prof, err := cfg.Profile(name)
			if err != nil {
				return err
			}
			out := struct {
				Profile        string `json:"profile"`
				Server         string `json:"server"`
				Fingerprint    string `json:"fingerprint,omitempty"`
				Username       string `json:"username,omitempty"`
				LoggedIn       bool   `json:"logged_in"`
				RefreshExpires string `json:"refresh_expires,omitempty"`
				AccessExpires  string `json:"access_expires,omitempty"`
			}{
				Profile:     name,
				Server:      prof.Server,
				Fingerprint: prof.Fingerprint,
				Username:    prof.Username,
			}
			if exp, ok := tokenExpiry(prof.RefreshToken); ok {
				out.RefreshExpires = exp.Format(time.RFC3339)
				out.LoggedIn = time.Now().Before(exp)
			}
			if exp, ok := tokenExpiry(prof.AccessToken); ok {
				out.AccessExpires = exp.Format(time.RFC3339)
			}
			return printValue(out)
		},
	}
}

func profilesCommand() *cli.Command {
	return &cli.Command{
		Name:  "profiles",
		Usage: "list profiles; the current one is marked with *",
		Action: func(c *cli.Context) error {
			cfg, err := loadConfig(c)
			if err != nil {
				return err
			}
			current := cfg.ProfileName("")
			for _, name := range cfg.ProfileNames() {
				mark := " "
				if name == current {
					mark = "*"
				}
				p := cfg.Profiles[name]
				fmt.Printf("%s %s\t%s\t%s\n", mark, name, p.Server, p.Username)
			}
			return nil
		},
	}
}

func useCommand() *cli.Command {
	return &cli.Command{
		Name:      "use",
		Usage:     "make a profile the current one",
		ArgsUsage: "PROFILE",
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return cli.ShowSubcommandHelp(c)
			}
			cfg, err := loadConfig(c)
			if err != nil {
				return err
			}
			name := c.Args().First()
			if _, err := cfg.Profile(name); err != nil {
				return err
			}
			cfg.Current = name
			return cfg.Save()
		},
	}
}
