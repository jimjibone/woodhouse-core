package config

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/yamlfile"
)

type CoreConfig struct {
	Changed      bool   `yaml:"-"`
	InstanceName string `yaml:"instance-name"`
	// ShowInstanceName opts the admin interface into showing InstanceName in
	// place of the Woodhouse product name. Off by default, so a single-server
	// install just says "Woodhouse".
	ShowInstanceName bool         `yaml:"show-instance-name"`
	Server           ServerConfig `yaml:"server"`
}

type ServerConfig struct {
	ApiAddr string `yaml:"api-addr"`
	WebAddr string `yaml:"web-addr"`
}

var LoadedConfig CoreConfig = defaultConfig

var defaultConfig = CoreConfig{
	Server: ServerConfig{
		ApiAddr: "localhost:4000",
		WebAddr: "localhost:4080",
	},
}

// mu guards LoadedConfig and path. Startup reads are single-threaded, but the
// settings RPC lets a user change the instance name while the server is
// running, so every runtime access goes through the accessors below.
var mu sync.RWMutex

// path is where LoadedConfig was loaded from, and where Save writes it back.
// It is set once from main so that persistence does not have to live there.
var path string

// SetPath records the config file location for later saves.
func SetPath(configPath string) {
	mu.Lock()
	defer mu.Unlock()
	path = configPath
}

// Path returns the config file location.
func Path() string {
	mu.RLock()
	defer mu.RUnlock()
	return path
}

// InstanceName returns the name this server advertises on the local network.
func InstanceName() string {
	mu.RLock()
	defer mu.RUnlock()
	return LoadedConfig.InstanceName
}

// SetInstanceName validates and stores a new instance name, persisting the
// config to disk. It returns the name as stored, which may differ from the
// input because it is normalised. Setting the name it already has is a no-op.
//
// Note that this only changes the configured name - re-announcing the new name
// on the network is the caller's job. Use core.SettingsManager to do both.
func SetInstanceName(name string) (string, error) {
	normalised, err := ValidateInstanceName(name)
	if err != nil {
		return "", err
	}

	mu.Lock()
	defer mu.Unlock()

	if LoadedConfig.InstanceName == normalised {
		return normalised, nil
	}

	previous := LoadedConfig.InstanceName
	LoadedConfig.InstanceName = normalised
	if err := save(); err != nil {
		// Leave the in-memory config matching what is on disk.
		LoadedConfig.InstanceName = previous
		return "", err
	}
	return normalised, nil
}

// ShowInstanceName reports whether the admin interface should show the
// instance name instead of the Woodhouse product name.
func ShowInstanceName() bool {
	mu.RLock()
	defer mu.RUnlock()
	return LoadedConfig.ShowInstanceName
}

// SetShowInstanceName stores the preference and persists the config to disk.
// Unlike the instance name this is purely cosmetic, so nothing is re-announced.
func SetShowInstanceName(show bool) error {
	mu.Lock()
	defer mu.Unlock()

	if LoadedConfig.ShowInstanceName == show {
		return nil
	}

	previous := LoadedConfig.ShowInstanceName
	LoadedConfig.ShowInstanceName = show
	if err := save(); err != nil {
		// Leave the in-memory config matching what is on disk.
		LoadedConfig.ShowInstanceName = previous
		return err
	}
	return nil
}

// maxInstanceNameBytes is the DNS label limit. A DNS-SD service instance name
// is a single label, so a longer name cannot be advertised.
const maxInstanceNameBytes = 63

// ValidateInstanceName checks that a name is usable as a DNS-SD service
// instance name and returns it normalised. RFC 6763 section 4.1.1 allows any
// UTF-8, including spaces and punctuation, so the only limits are the label
// length and characters that cannot survive the wire format.
func ValidateInstanceName(name string) (string, error) {
	normalised := strings.TrimSpace(name)
	if normalised == "" {
		return "", fmt.Errorf("instance name must not be empty")
	}
	if !utf8.ValidString(normalised) {
		return "", fmt.Errorf("instance name must be valid UTF-8")
	}
	if len(normalised) > maxInstanceNameBytes {
		return "", fmt.Errorf("instance name must be %d bytes or fewer, got %d", maxInstanceNameBytes, len(normalised))
	}
	for _, r := range normalised {
		// Covers the C0 range plus DEL and the C1 range.
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return "", fmt.Errorf("instance name must not contain control characters")
		}
	}
	return normalised, nil
}

// Save writes the config to disk and clears the Changed flag.
func Save() error {
	mu.Lock()
	defer mu.Unlock()
	return save()
}

// save writes the config to disk. The caller must hold mu.
func save() error {
	if path == "" {
		return fmt.Errorf("config path not set")
	}
	if err := yamlfile.SaveFile(LoadedConfig, path); err != nil {
		return fmt.Errorf("failed to save config to %s: %w", path, err)
	}
	LoadedConfig.Changed = false
	return nil
}

// Returns an error if the config is not valid.
func (c *CoreConfig) Verify() error {
	// Default the instance name to the system hostname if not set.
	if c.InstanceName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("instance-name not set and failed to get hostname: %w", err)
		}
		name, err := ValidateInstanceName(hostname)
		if err != nil {
			return fmt.Errorf("instance-name not set and hostname %q is not usable: %w", hostname, err)
		}
		c.InstanceName = name
		c.Changed = true
	} else {
		name, err := ValidateInstanceName(c.InstanceName)
		if err != nil {
			return fmt.Errorf("invalid instance-name: %w", err)
		}
		if name != c.InstanceName {
			c.InstanceName = name
			c.Changed = true
		}
	}
	if err := c.Server.Verify(); err != nil {
		return err
	}
	return nil
}

// Returns an error if the config is not valid.
func (c ServerConfig) Verify() error {
	if c.ApiAddr == "" {
		return fmt.Errorf("server.api-addr must be defined")
	}
	if c.WebAddr == "" {
		return fmt.Errorf("server.web-addr must be defined")
	}
	return nil
}
