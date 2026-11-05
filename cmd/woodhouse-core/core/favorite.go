package core

import (
	"time"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-api/values"
	"google.golang.org/protobuf/proto"
)

type FavoriteID struct {
	DeviceID   string `json:"device_id"`
	EndpointID string `json:"endpoint_id"`
	ServiceID  string `json:"service_id"`
}

func (f FavoriteID) Key() string {
	return f.DeviceID + "." + f.EndpointID + "." + f.ServiceID
}

type Optional[T any] struct {
	set bool
	val T
}

func (o *Optional[T]) Has() bool { return o.set }
func (o *Optional[T]) Get() T    { return o.val }
func (o *Optional[T]) Unset()    { o.set = false }
func (o *Optional[T]) Set(val T) {
	o.set = true
	o.val = val
}

type Favorite struct {
	DeviceID      string
	EndpointID    string
	ServiceID     string
	Name          Optional[string]
	EndpointLabel Optional[string] // The label from the bridge.
	UserLabel     string           // The user's label for the endpoint, if any. Overrides EndpointLabel.
	DeviceType    Optional[clientsapi.Device_DeviceType]
	Online        Optional[bool]
	LastSeen      Optional[time.Time]
	BatteryLevel  Optional[int64]
	Service       *clientsapi.Service
}

func (favorite *Favorite) ID() FavoriteID {
	return FavoriteID{DeviceID: favorite.DeviceID, EndpointID: favorite.EndpointID, ServiceID: favorite.ServiceID}
}

func (favorite *Favorite) Update(update *clientsapi.Device) bool {
	changed := false
	if update != nil {
		// The device type is only sent by some updates, so only take it when set.
		if typ := update.GetTyp(); typ != clientsapi.Device_UNDEFINED {
			if favorite.DeviceType.Get() != typ {
				changed = true
			}
			favorite.DeviceType.Set(typ)
		}

		fullState := update.GetFullState()

		for _, ep := range update.GetEndpoints() {
			if ep.GetId() == favorite.EndpointID {
				if favorite.EndpointLabel.Get() != ep.GetLabel() {
					changed = true
				}
				favorite.EndpointLabel.Set(ep.GetLabel())
			}

			for _, update := range ep.GetServices() {
				// The device-wide services live on the root endpoint.
				if ep.GetId() == RootEndpointID && update.GetTyp() == clientsapi.Service_INFO {
					for _, attr := range update.GetAttrs() {
						switch attr.GetId() {
						case "name":
							if favorite.Name.Get() != attr.GetValue().GetText() {
								changed = true
							}
							favorite.Name.Set(attr.GetValue().GetText())
						}
					}
				}
				if ep.GetId() == RootEndpointID && update.GetTyp() == clientsapi.Service_ONLINE {
					for _, attr := range update.GetAttrs() {
						switch attr.GetId() {
						case "online":
							if favorite.Online.Get() != attr.GetValue().GetBool() {
								changed = true
							}
							favorite.Online.Set(attr.GetValue().GetBool())

						case "last_seen":
							attrTime, _ := values.AsTime(attr.GetValue())
							if !favorite.LastSeen.Get().Equal(attrTime) {
								changed = true
							}
							favorite.LastSeen.Set(attrTime)
						}
					}
				}
				if update.GetTyp() == clientsapi.Service_BATTERY {
					for _, attr := range update.GetAttrs() {
						switch attr.GetId() {
						case "level":
							attrLevel := attr.GetValue().GetInt()
							if favorite.BatteryLevel.Get() != attrLevel {
								changed = true
							}
							favorite.BatteryLevel.Set(attrLevel)
						}
					}
				}
				if ep.GetId() == favorite.EndpointID && update.GetId() == favorite.ServiceID {
					changed = true
					if favorite.Service == nil {
						favorite.Service = proto.Clone(update).(*clientsapi.Service)
					} else {
						mergeService(favorite.Service, update, fullState)
					}
				}
			}
		}
	}
	return changed
}

// SetUserLabel sets the user's label for the favorite's endpoint, reporting
// whether it changed.
func (favorite *Favorite) SetUserLabel(label string) bool {
	if favorite.UserLabel == label {
		return false
	}
	favorite.UserLabel = label
	return true
}

func (favorite *Favorite) Clone() *Favorite {
	clone := *favorite
	clone.Service = proto.Clone(favorite.Service).(*clientsapi.Service)
	return &clone
}

func (favorite *Favorite) Pb() *clientsapi.DeviceService {
	srv := &clientsapi.DeviceService{
		Key:        favorite.ID().Key(),
		DeviceId:   favorite.DeviceID,
		EndpointId: favorite.EndpointID,
		FullState:  true,
		Service:    proto.Clone(favorite.Service).(*clientsapi.Service),
	}

	if favorite.Name.Has() {
		srv.DeviceName = proto.String(favorite.Name.Get())
	}
	if favorite.UserLabel != "" {
		srv.EndpointLabel = proto.String(favorite.UserLabel)
	} else if favorite.EndpointLabel.Has() {
		srv.EndpointLabel = proto.String(favorite.EndpointLabel.Get())
	}
	if favorite.DeviceType.Has() {
		srv.DeviceType = favorite.DeviceType.Get().Enum()
	}
	if favorite.Online.Has() {
		srv.Online = proto.Bool(favorite.Online.Get())
	}
	if favorite.LastSeen.Has() {
		srv.LastSeen = values.TimeToPb(favorite.LastSeen.Get())
	}
	if favorite.BatteryLevel.Has() {
		srv.BatteryLevel = proto.Int64(favorite.BatteryLevel.Get())
	}

	return srv
}
