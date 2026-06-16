package core

import (
	"time"

	"github.com/jimjibone/log"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	"github.com/jimjibone/woodhouse-core/apitools"
	"google.golang.org/protobuf/proto"
)

type FavoriteID struct {
	DeviceID  string `json:"device_id"`
	ServiceID string `json:"service_id"`
}

func (f FavoriteID) Key() string {
	return f.DeviceID + "." + f.ServiceID
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
	DeviceID     string
	ServiceID    string
	Name         Optional[string]
	Online       Optional[bool]
	LastSeen     Optional[time.Time]
	BatteryLevel Optional[int64]
	Service      *clientsapi.Service
}

func (favorite *Favorite) Update(update *clientsapi.Device) bool {
	changed := false
	if update != nil {
		// Get the fave service.
		gotInfo := false
		gotOnline := false
		gotService := false
		fullState := update.GetFullState()

		for _, update := range update.GetServices() {
			if update.GetTyp() == clientsapi.Service_INFO {
				gotInfo = true
				for _, attr := range update.GetAttrs() {
					switch attr.GetId() {
					case "name":
						if favorite.Name.Get() != attr.GetText().GetValue() {
							changed = true
						}
						favorite.Name.Set(attr.GetText().GetValue())
					}
				}
			}
			if update.GetTyp() == clientsapi.Service_ONLINE {
				gotOnline = true
				for _, attr := range update.GetAttrs() {
					switch attr.GetId() {
					case "online":
						if favorite.Online.Get() != attr.GetBool().GetValue() {
							changed = true
						}
						favorite.Online.Set(attr.GetBool().GetValue())

					case "last_seen":
						attrTime := apitools.AttributeToTime(attr.GetTime())
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
						attrLevel := attr.GetInt().GetValue()
						if favorite.BatteryLevel.Get() != attrLevel {
							changed = true
						}
						favorite.BatteryLevel.Set(attrLevel)
					}
				}
			}
			if update.GetId() == favorite.ServiceID {
				changed = true
				gotService = true
				if favorite.Service == nil {
					favorite.Service = update
				} else {
					srv := favorite.Service

					// Update general service info.
					srv.Typ = update.GetTyp()
					srv.Alias = update.GetAlias()

					if fullState {
						// Remove attributes that are no longer found in the service.
						var keep []*clientsapi.Attribute
						for _, attr := range srv.Attrs {
							found := false
							for _, upd := range update.Attrs {
								if attr.GetId() == upd.GetId() {
									found = true
									keep = append(keep, attr)
									break
								}
							}
							if !found {
								log.Debugf("favorite %q service %q removed attribute %q", favorite.DeviceID, favorite.ServiceID, attr.GetId())
							}
						}
						srv.Attrs = keep
					}

					// Update attributes.
					for _, upd := range update.Attrs {
						found := false
						for i, attr := range srv.Attrs {
							if attr.GetId() == upd.GetId() {
								found = true
								srv.Attrs[i] = upd
								break
							}
						}
						if !found {
							srv.Attrs = append(srv.Attrs, upd)
						}
					}
				}
			}
			if gotInfo && gotOnline && gotService {
				break
			}
		}
	}
	return changed
}

func (favorite *Favorite) Clone() *Favorite {
	return &Favorite{
		DeviceID:     favorite.DeviceID,
		ServiceID:    favorite.ServiceID,
		Name:         favorite.Name,
		Online:       favorite.Online,
		LastSeen:     favorite.LastSeen,
		BatteryLevel: favorite.BatteryLevel,
		Service:      proto.Clone(favorite.Service).(*clientsapi.Service),
	}
}

func (favorite *Favorite) Pb() *clientsapi.DeviceService {
	srv := &clientsapi.DeviceService{
		Key:       favorite.DeviceID + "." + favorite.ServiceID,
		DeviceId:  favorite.DeviceID,
		FullState: true,
		// DeviceName:   new(string),
		// Online:       new(bool),
		// LastSeen:     &clientsapi.TimeValue{},
		// BatteryLevel: new(int64),
		Service: proto.Clone(favorite.Service).(*clientsapi.Service),
	}

	if favorite.Name.Has() {
		srv.DeviceName = proto.String(favorite.Name.Get())
	}
	if favorite.Online.Has() {
		srv.Online = proto.Bool(favorite.Online.Get())
	}
	if favorite.LastSeen.Has() {
		secs, nanos := favorite.LastSeen.Get().Unix(), int32(favorite.LastSeen.Get().Nanosecond())
		if favorite.LastSeen.Get().IsZero() {
			secs, nanos = 0, 0
		}
		srv.LastSeen = &clientsapi.TimeValue{
			Seconds: secs,
			Nanos:   nanos,
		}
	}
	if favorite.BatteryLevel.Has() {
		srv.BatteryLevel = proto.Int64(favorite.BatteryLevel.Get())
	}

	return srv
}
