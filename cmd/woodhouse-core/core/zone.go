package core

import (
	"fmt"
	"slices"
	"strings"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
)

// Zone is a room or area of the home. Unlike a Group, a Zone is pure metadata:
// it holds no device state and routes no actions, so it is never published to
// the DeviceManager as a synthetic device. Clients join zones against the
// devices stream themselves.
//
// A device belongs to at most one zone, and its endpoints with it. An endpoint
// may override that by being listed in another zone's Endpoints. An endpoint
// belongs to at most one zone. ZoneManager owns these invariants.
type Zone struct {
	ZoneID    string        `json:"zone_id"`
	Name      string        `json:"name"`
	Icon      string        `json:"icon"`
	DeviceIDs []string      `json:"device_ids"`
	Endpoints []EndpointRef `json:"endpoints"`
}

// EndpointRef identifies one endpoint of a device.
type EndpointRef struct {
	DeviceID   string `json:"device_id"`
	EndpointID string `json:"endpoint_id"`
}

func NewZone(zoneID, name, icon string, deviceIDs []string, endpoints ...EndpointRef) *Zone {
	return &Zone{
		ZoneID:    zoneID,
		Name:      name,
		Icon:      icon,
		DeviceIDs: slices.Clone(deviceIDs),
		Endpoints: slices.Clone(endpoints),
	}
}

// EndpointRefsFromPb converts the wire form of endpoint references.
func EndpointRefsFromPb(refs []*clientsapi.EndpointRef) []EndpointRef {
	out := make([]EndpointRef, 0, len(refs))
	for _, ref := range refs {
		out = append(out, EndpointRef{DeviceID: ref.GetDeviceId(), EndpointID: ref.GetEndpointId()})
	}
	return out
}

func (zone *Zone) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "zone_id:%q, name:%q, icon:%q, devices:%d, endpoints:%d", zone.ZoneID, zone.Name, zone.Icon, len(zone.DeviceIDs), len(zone.Endpoints))
	for i, deviceID := range zone.DeviceIDs {
		fmt.Fprintf(&b, "\n  %d: device_id:%q", i, deviceID)
	}
	for i, ref := range zone.Endpoints {
		fmt.Fprintf(&b, "\n  %d: device_id:%q endpoint_id:%q", i, ref.DeviceID, ref.EndpointID)
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
		Endpoints: slices.Clone(zone.Endpoints),
	}
}

func (zone *Zone) Pb() *clientsapi.Zone {
	pb := &clientsapi.Zone{
		Id:        zone.ZoneID,
		Name:      zone.Name,
		Icon:      zone.Icon,
		DeviceIds: slices.Clone(zone.DeviceIDs),
	}
	for _, ref := range zone.Endpoints {
		pb.Endpoints = append(pb.Endpoints, &clientsapi.EndpointRef{DeviceId: ref.DeviceID, EndpointId: ref.EndpointID})
	}
	return pb
}

func (zone *Zone) HasDevice(deviceID string) bool {
	return slices.Contains(zone.DeviceIDs, deviceID)
}

// HasEndpoint reports whether the endpoint is listed as an override.
func (zone *Zone) HasEndpoint(deviceID, endpointID string) bool {
	return slices.Contains(zone.Endpoints, EndpointRef{DeviceID: deviceID, EndpointID: endpointID})
}

// removeEndpoints drops every endpoint matching drop, reporting whether any
// were there.
func (zone *Zone) removeEndpoints(drop func(EndpointRef) bool) bool {
	next := slices.DeleteFunc(slices.Clone(zone.Endpoints), drop)
	if len(next) == len(zone.Endpoints) {
		return false
	}
	zone.Endpoints = next
	return true
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
