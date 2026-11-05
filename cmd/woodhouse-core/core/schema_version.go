package core

import (
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/jimjibone/log"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

// SchemaVersion is the version of the stored state this core writes. Version 2
// is the endpoint device model.
const SchemaVersion = 2

// schemaVersionKey is the store key recording the version of the stored state.
const schemaVersionKey = "schema-version"

// v1StateKeys are the stored devices, groups and zones, which v1 wrote and v2
// cannot read. Favourites are not stored apart: they are flags in the devices.
// The endpoint labels are v2 state, but are listed so a wipe cannot leave labels
// that point at devices that are gone.
var v1StateKeys = []string{
	"devices", // Legacy name of devices.json.
	"devices.json",
	"groups.json",
	"zones.json",
	endpointLabelsKey,
}

// WipeV1State deletes the stored devices, favourites, groups, zones and
// endpoint labels the first time a core of schema version 2 starts, then
// records the version so it never runs again. Devices are rebuilt by the
// bridges; favourites, groups and zones are set up again by hand, as the
// device IDs change. Everything else (users, clients, invites, notifications,
// auth state) is left alone.
//
// It must run before the managers load their state.
func WipeV1State(store stores.Store) error {
	l := log.NewContext(log.DefaultLogger, "schema-version", log.DebugLevel)

	if store.Has(schemaVersionKey) {
		data, err := store.Get(schemaVersionKey)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", schemaVersionKey, err)
		}
		version, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err == nil && version >= SchemaVersion {
			return nil
		}
		// An unreadable version is treated as missing.
	}

	var wiped []string
	for _, key := range v1StateKeys {
		if !store.Has(key) {
			continue
		}
		err := store.Del(key)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to wipe %s: %w", key, err)
		}
		wiped = append(wiped, key)
	}
	if len(wiped) > 0 {
		l.Infof("first start of schema version %d, wiped stored state: %s", SchemaVersion, strings.Join(wiped, ", "))
	} else {
		l.Infof("first start of schema version %d, no stored state to wipe", SchemaVersion)
	}

	err := store.Set(schemaVersionKey, []byte(strconv.Itoa(SchemaVersion)))
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", schemaVersionKey, err)
	}
	return nil
}
