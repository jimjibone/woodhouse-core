package core

import (
	"sync"

	"github.com/jimjibone/log"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/config"
	"github.com/jimjibone/woodhouse-core/discovery"
)

// SettingsManager owns the server settings that can be changed while the
// server is running. It keeps the config file and the network advertisement in
// step, so that a change made through the API takes effect without a restart.
type SettingsManager struct {
	log         *log.Context
	mu          sync.Mutex
	broadcaster *discovery.Broadcaster
}

func NewSettingsManager(broadcaster *discovery.Broadcaster) *SettingsManager {
	return &SettingsManager{
		log:         log.NewContext(log.DefaultLogger, "settings-manager", log.DebugLevel),
		broadcaster: broadcaster,
	}
}

// InstanceName returns the name this server advertises on the local network.
func (m *SettingsManager) InstanceName() string {
	return config.InstanceName()
}

// ShowInstanceName reports whether the admin interface should show the
// instance name in place of the Woodhouse product name.
func (m *SettingsManager) ShowInstanceName() bool {
	return config.ShowInstanceName()
}

// SetShowInstanceName stores the preference. It only affects what the admin
// interface renders, so there is nothing to re-announce.
func (m *SettingsManager) SetShowInstanceName(show bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config.ShowInstanceName() == show {
		return nil
	}
	if err := config.SetShowInstanceName(show); err != nil {
		return err
	}

	m.log.Infof("show instance name in the interface: %t", show)
	return nil
}

// SetInstanceName validates and stores a new instance name, then re-announces
// it on the network. It returns the name as stored, which may differ from the
// input because it is normalised.
//
// If the re-announcement fails the stored name is rolled back, so the config
// file never claims a name that is not actually being advertised.
func (m *SettingsManager) SetInstanceName(name string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	previous := config.InstanceName()

	stored, err := config.SetInstanceName(name)
	if err != nil {
		return "", err
	}
	if stored == previous {
		return stored, nil
	}

	if m.broadcaster != nil {
		if err := m.broadcaster.SetInstance(stored); err != nil {
			if _, restoreErr := config.SetInstanceName(previous); restoreErr != nil {
				m.log.Errorf("failed to restore instance name to %q: %s", previous, restoreErr)
			}
			return "", err
		}
	}

	m.log.Infof("instance name changed from %q to %q", previous, stored)
	return stored, nil
}
