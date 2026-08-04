package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/yamlfile"
)

func TestValidateInstanceName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "woodhouse", "woodhouse"},
		{"trims surrounding whitespace", "  banana  ", "banana"},
		// RFC 6763 4.1.1 allows rich instance names, so these must survive.
		{"keeps inner spaces", "Front Room Woodhouse", "Front Room Woodhouse"},
		{"keeps punctuation", "My' house (main)", "My' house (main)"},
		{"keeps non-ascii", "Küche", "Küche"},
		{"at the length limit", strings.Repeat("a", 63), strings.Repeat("a", 63)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ValidateInstanceName(test.input)
			if err != nil {
				t.Fatalf("ValidateInstanceName(%q) returned an error: %v", test.input, err)
			}
			if got != test.want {
				t.Errorf("ValidateInstanceName(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestValidateInstanceNameRejects(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"whitespace only", "   "},
		// The advertised name is a single DNS label, so 63 bytes is the cap.
		{"too long", strings.Repeat("a", 64)},
		// Multi-byte characters count against the byte limit, not a rune limit.
		{"too long in bytes", strings.Repeat("ü", 32)},
		{"newline", "wood\nhouse"},
		{"tab", "wood\thouse"},
		{"null", "wood\x00house"},
		{"delete", "wood\x7fhouse"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got, err := ValidateInstanceName(test.input); err == nil {
				t.Errorf("ValidateInstanceName(%q) = %q, want an error", test.input, got)
			}
		})
	}
}

func TestVerifyNormalisesInstanceName(t *testing.T) {
	cfg := CoreConfig{
		InstanceName: "  banana  ",
		Server:       ServerConfig{ApiAddr: "localhost:4000", WebAddr: "localhost:4080"},
	}
	if err := cfg.Verify(); err != nil {
		t.Fatalf("Verify returned an error: %v", err)
	}
	if cfg.InstanceName != "banana" {
		t.Errorf("InstanceName = %q, want %q", cfg.InstanceName, "banana")
	}
	// Normalising rewrote the value, so it has to reach disk.
	if !cfg.Changed {
		t.Error("Changed = false, want true after the name was normalised")
	}
}

func TestVerifyDefaultsInstanceNameToHostname(t *testing.T) {
	cfg := CoreConfig{
		Server: ServerConfig{ApiAddr: "localhost:4000", WebAddr: "localhost:4080"},
	}
	if err := cfg.Verify(); err != nil {
		t.Fatalf("Verify returned an error: %v", err)
	}
	if cfg.InstanceName == "" {
		t.Error("InstanceName is empty, want the system hostname")
	}
	if !cfg.Changed {
		t.Error("Changed = false, want true after the hostname default was applied")
	}
}

func TestVerifyRejectsUnusableInstanceName(t *testing.T) {
	cfg := CoreConfig{
		InstanceName: strings.Repeat("a", 64),
		Server:       ServerConfig{ApiAddr: "localhost:4000", WebAddr: "localhost:4080"},
	}
	if err := cfg.Verify(); err == nil {
		t.Fatal("Verify accepted an instance name that cannot be advertised")
	}
}

func TestSetInstanceNamePersistsImmediately(t *testing.T) {
	path := filepath.Join(t.TempDir(), "woodhouse.yaml")

	previousPath := Path()
	previousConfig := LoadedConfig
	t.Cleanup(func() {
		SetPath(previousPath)
		LoadedConfig = previousConfig
	})

	SetPath(path)
	LoadedConfig = CoreConfig{
		InstanceName: "before",
		Server:       ServerConfig{ApiAddr: "localhost:4000", WebAddr: "localhost:4080"},
	}

	stored, err := SetInstanceName("  after  ")
	if err != nil {
		t.Fatalf("SetInstanceName returned an error: %v", err)
	}
	if stored != "after" {
		t.Errorf("SetInstanceName returned %q, want %q", stored, "after")
	}
	if got := InstanceName(); got != "after" {
		t.Errorf("InstanceName() = %q, want %q", got, "after")
	}

	// The point of the feature: the change is on disk now, not at shutdown.
	var reloaded CoreConfig
	if err := yamlfile.LoadFile(&reloaded, path); err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if reloaded.InstanceName != "after" {
		t.Errorf("persisted instance-name = %q, want %q", reloaded.InstanceName, "after")
	}
	// The rest of the config must survive a settings write.
	if reloaded.Server.ApiAddr != "localhost:4000" || reloaded.Server.WebAddr != "localhost:4080" {
		t.Errorf("persisted server config = %+v, want the addresses unchanged", reloaded.Server)
	}
}

func TestSetInstanceNameRejectsInvalidWithoutWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "woodhouse.yaml")

	previousPath := Path()
	previousConfig := LoadedConfig
	t.Cleanup(func() {
		SetPath(previousPath)
		LoadedConfig = previousConfig
	})

	SetPath(path)
	LoadedConfig = CoreConfig{
		InstanceName: "before",
		Server:       ServerConfig{ApiAddr: "localhost:4000", WebAddr: "localhost:4080"},
	}

	if _, err := SetInstanceName("  "); err == nil {
		t.Fatal("SetInstanceName accepted an empty name")
	}
	if got := InstanceName(); got != "before" {
		t.Errorf("InstanceName() = %q, want the previous name %q", got, "before")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("a rejected name still wrote the config file")
	}
}

func TestShowInstanceNameDefaultsOff(t *testing.T) {
	// A fresh install should call itself "Woodhouse", not the hostname.
	if defaultConfig.ShowInstanceName {
		t.Error("defaultConfig.ShowInstanceName = true, want false")
	}

	cfg := CoreConfig{
		Server: ServerConfig{ApiAddr: "localhost:4000", WebAddr: "localhost:4080"},
	}
	if err := cfg.Verify(); err != nil {
		t.Fatalf("Verify returned an error: %v", err)
	}
	if cfg.ShowInstanceName {
		t.Error("Verify turned ShowInstanceName on, want it left off")
	}
}

func TestSetShowInstanceNamePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "woodhouse.yaml")

	previousPath := Path()
	previousConfig := LoadedConfig
	t.Cleanup(func() {
		SetPath(previousPath)
		LoadedConfig = previousConfig
	})

	SetPath(path)
	LoadedConfig = CoreConfig{
		InstanceName: "banana",
		Server:       ServerConfig{ApiAddr: "localhost:4000", WebAddr: "localhost:4080"},
	}

	if ShowInstanceName() {
		t.Fatal("ShowInstanceName() = true before it was set")
	}
	if err := SetShowInstanceName(true); err != nil {
		t.Fatalf("SetShowInstanceName returned an error: %v", err)
	}
	if !ShowInstanceName() {
		t.Error("ShowInstanceName() = false after enabling it")
	}

	var reloaded CoreConfig
	if err := yamlfile.LoadFile(&reloaded, path); err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if !reloaded.ShowInstanceName {
		t.Error("persisted show-instance-name = false, want true")
	}
	// Turning the preference on must not disturb the name it displays.
	if reloaded.InstanceName != "banana" {
		t.Errorf("persisted instance-name = %q, want %q", reloaded.InstanceName, "banana")
	}

	if err := SetShowInstanceName(false); err != nil {
		t.Fatalf("SetShowInstanceName(false) returned an error: %v", err)
	}
	if ShowInstanceName() {
		t.Error("ShowInstanceName() = true after disabling it")
	}
}
