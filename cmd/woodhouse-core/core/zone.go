package core

import (
	"fmt"
	"slices"
	"strings"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
)

// Zone is a room or area of the home. Unlike a Group, a Zone is pure metadata:
// it holds no device state and routes no actions, so it is never published to
// the DeviceManager as a synthetic device. Clients join zones against the
// devices stream themselves.
//
// A device belongs to at most one zone. ZoneManager owns that invariant.
type Zone struct {
	ZoneID    string   `json:"zone_id"`
	Name      string   `json:"name"`
	Icon      string   `json:"icon"`
	DeviceIDs []string `json:"device_ids"`
}

func NewZone(zoneID, name, icon string, deviceIDs []string) *Zone {
	return &Zone{
		ZoneID:    zoneID,
		Name:      name,
		Icon:      icon,
		DeviceIDs: slices.Clone(deviceIDs),
	}
}

func (zone *Zone) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "zone_id:%q, name:%q, icon:%q, devices:%d", zone.ZoneID, zone.Name, zone.Icon, len(zone.DeviceIDs))
	for i, deviceID := range zone.DeviceIDs {
		fmt.Fprintf(&b, "\n  %d: device_id:%q", i, deviceID)
	}
	return b.String()
}

// Clone returns a copy safe to hand to listeners while the manager keeps
// mutating the original.
func (zone *Zone) Clone() *Zone {
	return &Zone{
		ZoneID:    zone.ZoneID,
		Name:      zone.Name,
		Icon:      zone.Icon,
		DeviceIDs: slices.Clone(zone.DeviceIDs),
	}
}

func (zone *Zone) Pb() *clientsapi.Zone {
	return &clientsapi.Zone{
		Id:        zone.ZoneID,
		Name:      zone.Name,
		Icon:      zone.Icon,
		DeviceIds: slices.Clone(zone.DeviceIDs),
	}
}

func (zone *Zone) HasDevice(deviceID string) bool {
	return slices.Contains(zone.DeviceIDs, deviceID)
}

// removeDevice drops a device from the zone, reporting whether it was there.
func (zone *Zone) removeDevice(deviceID string) bool {
	next := slices.DeleteFunc(slices.Clone(zone.DeviceIDs), func(id string) bool {
		return id == deviceID
	})
	if len(next) == len(zone.DeviceIDs) {
		return false
	}
	zone.DeviceIDs = next
	return true
}
