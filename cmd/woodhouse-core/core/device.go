package core

import (
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/jimjibone/log"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-api/values"
	"google.golang.org/protobuf/proto"
)

// RootEndpointID is the ID of the root endpoint, which holds the device-wide
// services (info, online, ...).
const RootEndpointID = "0"

type Device struct {
	ClientID  string
	ID        string
	Typ       clientsapi.Device_DeviceType
	Endpoints map[string]*DeviceEndpoint // key=endpoint id
	LastSeen  time.Time                  // See clientsapi.Device.last_seen.
}

// DeviceEndpoint is an endpoint of a device and the services it holds. Events
// are never stored.
type DeviceEndpoint struct {
	ID       string
	Label    string
	Services map[string]*clientsapi.Service // key=service id
}

func newDevice(log *log.Context, clientID string, update *clientsapi.Device) (*Device, error) {
	dev := &Device{
		ClientID:  clientID,
		ID:        update.GetId(),
		Endpoints: make(map[string]*DeviceEndpoint),
	}
	if log != nil {
		log.Debugf("device %q created", dev.ID)
	}
	_, err := dev.update(log, clientID, update)
	if err != nil {
		return nil, err
	}
	return dev, nil
}

// service returns the service or nil if not found.
func (dev *Device) service(endpointID, serviceID string) *clientsapi.Service {
	if ep, found := dev.Endpoints[endpointID]; found {
		return ep.Services[serviceID]
	}
	return nil
}

// pb returns a copy of the full state of the device. Endpoints and services are
// sorted by ID, attributes are in the order the bridge first reported them.
func (dev *Device) pb() *clientsapi.Device {
	pb := &clientsapi.Device{
		ClientId:  dev.ClientID,
		Id:        dev.ID,
		FullState: true,
		Typ:       dev.Typ,
		Endpoints: []*clientsapi.Endpoint{},
		LastSeen:  dev.lastSeenPb(),
	}
	for _, ep := range dev.Endpoints {
		pb.Endpoints = append(pb.Endpoints, ep.pb())
	}
	// Sorted to maintain consistent structure between saves.
	sort.Slice(pb.Endpoints, func(i, j int) bool {
		return pb.Endpoints[i].GetId() < pb.Endpoints[j].GetId()
	})
	return pb
}

func (ep *DeviceEndpoint) pb() *clientsapi.Endpoint {
	pb := &clientsapi.Endpoint{
		Id:       ep.ID,
		Label:    ep.Label,
		Services: []*clientsapi.Service{},
	}
	for _, srv := range ep.Services {
		pb.Services = append(pb.Services, proto.Clone(srv).(*clientsapi.Service))
	}
	// Sorted to maintain consistent structure between saves.
	sort.Slice(pb.Services, func(i, j int) bool {
		return pb.Services[i].GetId() < pb.Services[j].GetId()
	})
	return pb
}

// lastSeenPb returns LastSeen for the wire, nil if never seen.
func (dev *Device) lastSeenPb() *clientsapi.TimeValue {
	if dev.LastSeen.IsZero() {
		return nil
	}
	return values.TimeToPb(dev.LastSeen)
}

// seen moves LastSeen forward to t and reports whether it moved.
func (dev *Device) seen(t time.Time) bool {
	if t.After(dev.LastSeen) {
		dev.LastSeen = t
		return true
	}
	return false
}

// seenService moves LastSeen forward for an update of the service: to the time
// of any of its attributes, or for the online service to its last_seen value.
// The online service's own attribute times are skipped, as the server stamps
// them when it marks the device offline.
func (dev *Device) seenService(srv *clientsapi.Service) bool {
	moved := false
	for _, attr := range srv.GetAttrs() {
		if srv.GetTyp() != clientsapi.Service_ONLINE {
			if attr.GetTime() != nil && dev.seen(values.TimeFromPb(attr.GetTime())) {
				moved = true
			}
		} else if attr.GetId() == "last_seen" {
			if t, ok := values.AsTime(attr.GetValue()); ok && dev.seen(t) {
				moved = true
			}
		}
	}
	return moved
}

// seenEvents moves LastSeen forward to the time of any of the events.
func (dev *Device) seenEvents(events []*clientsapi.Event) bool {
	moved := false
	for _, event := range events {
		if event.GetTime() != nil && dev.seen(values.TimeFromPb(event.GetTime())) {
			moved = true
		}
	}
	return moved
}

// onlineService returns the online service of the root endpoint or nil.
func (dev *Device) onlineService() *clientsapi.Service {
	if ep, found := dev.Endpoints[RootEndpointID]; found {
		for _, srv := range ep.Services {
			if srv.GetTyp() == clientsapi.Service_ONLINE {
				return srv
			}
		}
	}
	return nil
}

func onlineAttribute(srv *clientsapi.Service) *clientsapi.Attribute {
	for _, attr := range srv.GetAttrs() {
		if attr.GetId() == "online" && values.Kind(attr.GetValue()) == "bool" {
			return attr
		}
	}
	return nil
}

func (dev *Device) isOnline() bool {
	if srv := dev.onlineService(); srv != nil {
		if attr := onlineAttribute(srv); attr != nil {
			return attr.GetValue().GetBool()
		}
	}
	return false
}

func (dev *Device) setOffline(log *log.Context) *clientsapi.Device {
	srv := dev.onlineService()
	if srv == nil {
		return nil
	}
	if attr := onlineAttribute(srv); attr != nil {
		attr.Value = values.Bool(false)
		attr.Time = values.TimeToPb(time.Now())
	}
	if log != nil {
		log.Debugf("device %q went offline\n%s", dev.ID, prettyService("  ", srv))
	}
	return &clientsapi.Device{
		Id:        dev.ID,
		ClientId:  dev.ClientID,
		FullState: false,
		Typ:       dev.Typ,
		Endpoints: []*clientsapi.Endpoint{
			{
				Id:       RootEndpointID,
				Label:    dev.Endpoints[RootEndpointID].Label,
				Services: []*clientsapi.Service{proto.Clone(srv).(*clientsapi.Service)},
			},
		},
	}
}

// Update the device state and return the changes or an error. The returned
// device holds the merged state of every service the update touched, so
// consumers do not need to merge partial updates themselves. Events in the
// update are ignored: they are not part of the device state.
//
// In a full state update everything the update does not report is removed. In
// a partial update an empty endpoint label, missing service features, commands
// or events and a nil attribute spec mean "unchanged".
func (dev *Device) update(log *log.Context, clientID string, update *clientsapi.Device) (*clientsapi.Device, error) {
	update.ClientId = clientID
	update.LastSeen = nil // Set by the server only.
	if dev.ClientID != clientID {
		dev.ClientID = clientID
		if log != nil {
			log.Debugf("device %q client set to %q", dev.ID, dev.ClientID)
		}
	}
	if dev.Typ != update.GetTyp() {
		dev.Typ = update.GetTyp()
		if log != nil {
			log.Debugf("device %q type set to %q", dev.ID, dev.Typ)
		}
	}
	if update.GetFullState() {
		dev.gcEndpoints(log, update)
	}

	// Attributes without a time were observed when they arrived.
	ingest := values.TimeToPb(time.Now())
	for _, upd := range update.GetEndpoints() {
		for _, srv := range upd.GetServices() {
			for _, attr := range srv.GetAttrs() {
				if attr.Time == nil {
					attr.Time = proto.Clone(ingest).(*clientsapi.TimeValue)
				}
			}
		}
	}

	changes := &clientsapi.Device{
		Id:        update.GetId(),
		ClientId:  clientID,
		FullState: update.GetFullState(),
		Typ:       update.GetTyp(),
	}
	seenMoved := false
	for _, upd := range update.GetEndpoints() {
		ep, found := dev.Endpoints[upd.GetId()]
		if !found {
			ep = &DeviceEndpoint{ID: upd.GetId(), Services: make(map[string]*clientsapi.Service)}
			dev.Endpoints[ep.ID] = ep
			if log != nil {
				log.Debugf("device %q added endpoint %q", dev.ID, ep.ID)
			}
		}
		if update.GetFullState() || upd.GetLabel() != "" {
			ep.Label = upd.GetLabel()
		}
		if update.GetFullState() {
			ep.gcServices(log, dev.ID, upd)
		}

		epChanges := &clientsapi.Endpoint{Id: ep.ID, Label: ep.Label}
		for _, srv := range upd.GetServices() {
			seenMoved = dev.seenService(srv) || seenMoved
			merged := dev.updateService(log, ep, update.GetFullState(), srv)
			epChanges.Services = append(epChanges.Services, proto.Clone(merged).(*clientsapi.Service))
		}
		changes.Endpoints = append(changes.Endpoints, epChanges)
	}
	if seenMoved || update.GetFullState() {
		changes.LastSeen = dev.lastSeenPb()
	}
	return changes, nil
}

// Garbage collect endpoints no longer reported by this device.
func (dev *Device) gcEndpoints(log *log.Context, update *clientsapi.Device) {
	for epID := range dev.Endpoints {
		found := slices.ContainsFunc(update.GetEndpoints(), func(upd *clientsapi.Endpoint) bool {
			return upd.GetId() == epID
		})
		if !found {
			if log != nil {
				log.Debugf("device %q removed endpoint %q", dev.ID, epID)
			}
			delete(dev.Endpoints, epID)
		}
	}
}

// Garbage collect services no longer reported by this endpoint.
func (ep *DeviceEndpoint) gcServices(log *log.Context, deviceID string, update *clientsapi.Endpoint) {
	for srvID, srv := range ep.Services {
		found := slices.ContainsFunc(update.GetServices(), func(upd *clientsapi.Service) bool {
			return upd.GetId() == srvID
		})
		if !found {
			if log != nil {
				log.Debugf("device %q endpoint %q removed service\n%s", deviceID, ep.ID, prettyService("  ", srv))
			}
			delete(ep.Services, srvID)
		}
	}
}

// Add or update a service and return the new state.
func (dev *Device) updateService(log *log.Context, ep *DeviceEndpoint, fullState bool, update *clientsapi.Service) *clientsapi.Service {
	srv, found := ep.Services[update.GetId()]
	if !found {
		if log != nil {
			log.Debugf("device %q endpoint %q added service\n%s", dev.ID, ep.ID, prettyService("  ", update))
		}
		ep.Services[update.GetId()] = update
		return update
	}

	mergeService(srv, update, fullState)
	if log != nil {
		log.Debugf("device %q endpoint %q updated service\n%s", dev.ID, ep.ID, prettyService("  ", srv))
	}
	return srv
}

// mergeService merges a service update into dst. The favorite flag is owned by
// the server and is left alone. The update is not modified or retained, so it
// may be shared.
func mergeService(dst, update *clientsapi.Service, fullState bool) {
	// Update general service info.
	dst.Typ = update.GetTyp()
	dst.Alias = update.GetAlias()
	if fullState || len(update.GetFeatures()) > 0 {
		dst.Features = slices.Clone(update.GetFeatures())
	}
	if fullState || len(update.GetCommands()) > 0 {
		dst.Commands = slices.Clone(update.GetCommands())
	}
	if fullState || len(update.GetEvents()) > 0 {
		dst.Events = slices.Clone(update.GetEvents())
	}

	if fullState {
		// Remove attributes that are no longer found in the service.
		dst.Attrs = slices.DeleteFunc(dst.Attrs, func(attr *clientsapi.Attribute) bool {
			return !slices.ContainsFunc(update.GetAttrs(), func(upd *clientsapi.Attribute) bool {
				return upd.GetId() == attr.GetId()
			})
		})
	}

	// Update attributes, keeping the order they were first reported in.
	for _, upd := range update.GetAttrs() {
		next := proto.Clone(upd).(*clientsapi.Attribute)
		i := slices.IndexFunc(dst.Attrs, func(attr *clientsapi.Attribute) bool {
			return attr.GetId() == next.GetId()
		})
		if i < 0 {
			dst.Attrs = append(dst.Attrs, next)
			continue
		}
		prev := dst.Attrs[i]
		if next.Spec == nil {
			next.Spec = prev.Spec
		}
		if next.Perms == clientsapi.Permissions_PERM_UNDEFINED {
			next.Perms = prev.Perms
		}
		dst.Attrs[i] = next
	}
}

func (dev *Device) updateFavoriteService(log *log.Context, endpointID, serviceID string, favorite bool) {
	if srv := dev.service(endpointID, serviceID); srv != nil {
		if log != nil {
			if favorite {
				log.Debugf("device %q added service favorite: %s/%s", dev.ID, endpointID, serviceID)
			} else {
				log.Debugf("device %q removed service favorite: %s/%s", dev.ID, endpointID, serviceID)
			}
		}

		// Update service favorite state.
		srv.Favorite = favorite
	}
}

func prettyService(pad string, srv *clientsapi.Service) string {
	str := fmt.Sprintf("%s- srv id:%q, typ:%q, alias:%q, attrs:%d, fave:%t", pad, srv.GetId(), srv.GetTyp(), srv.GetAlias(), len(srv.Attrs), srv.GetFavorite())
	for _, attr := range srv.Attrs {
		str += fmt.Sprintf("\n%s  - attr %s", pad, attr)
	}
	return str
}
