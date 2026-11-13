package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jimjibone/woodhouse-core/shared/atomicfile"
)

// Config is the whctl state file: one profile per server/user pair. It holds
// refresh tokens, so it is written with mode 0600.
type Config struct {
	// Current is the profile used when none is given on the command line.
	Current  string              `json:"current,omitempty"`
	Profiles map[string]*Profile `json:"profiles"`

	path string
}

// Profile is a server, the TLS certificate pinned for it and the session
// logged in to it.
type Profile struct {
	Server string `json:"server"`
	// Fingerprint is the lowercase hex SHA-256 of the server's TLS leaf
	// certificate (DER), the same value core advertises as "fp" in its
	// discovery TXT record. Empty means not pinned yet.
	Fingerprint  string `json:"fingerprint,omitempty"`
	Username     string `json:"username,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
}

const defaultProfile = "default"

// defaultConfigPath is $XDG_CONFIG_HOME/whctl/config.json, falling back to
// ~/.config/whctl/config.json.
func defaultConfigPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "whctl", "config.json"), nil
}

// LoadConfig reads the config at path. A missing file is an empty config.
func LoadConfig(path string) (*Config, error) {
	cfg := &Config{Profiles: map[string]*Profile{}, path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]*Profile{}
	}
	cfg.path = path
	return cfg, nil
}

// Save writes the config atomically with mode 0600.
func (c *Config) Save() error {
	if err := os.MkdirAll(filepath.Dir(c.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicfile.WriteFile(c.path, 0o600, bytes.NewReader(data))
}

// ProfileName resolves the profile to use: the explicit name if given, else
// the current profile, else "default".
func (c *Config) ProfileName(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if c.Current != "" {
		return c.Current
	}
	return defaultProfile
}

// Profile returns the named profile, or an error saying how to create it.
func (c *Config) Profile(name string) (*Profile, error) {
	p := c.Profiles[name]
	if p == nil {
		return nil, fmt.Errorf("profile %q does not exist; run 'whctl login --server HOST:PORT' first", name)
	}
	return p, nil
}

// ProfileNames returns the profile names, sorted.
func (c *Config) ProfileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
