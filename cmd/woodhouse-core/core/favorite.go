package core

import (
	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"google.golang.org/protobuf/proto"
)

type FavoriteID struct {
	DeviceID  string `json:"device_id"`
	ServiceID string `json:"service_id"`
}

func (f FavoriteID) Key() string {
	return f.DeviceID + "." + f.ServiceID
}

type Favorite struct {
	DeviceID  string
	ServiceID string
	HasName   bool
	Name      string
	HasOnline bool
	Online    bool
	Service   *clientsapi.Service
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
					if attr.GetId() == "name" {
						if favorite.Name != attr.GetText().GetValue() {
							changed = true
						}
						favorite.HasName = true
						favorite.Name = attr.GetText().GetValue()
					}
				}
			}
			if update.GetTyp() == clientsapi.Service_ONLINE {
				gotOnline = true
				for _, attr := range update.GetAttrs() {
					if attr.GetId() == "online" {
						if favorite.Online != attr.GetBool().GetValue() {
							changed = true
						}
						favorite.HasOnline = true
						favorite.Online = attr.GetBool().GetValue()
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
		DeviceID:  favorite.DeviceID,
		ServiceID: favorite.ServiceID,
		HasName:   favorite.HasName,
		Name:      favorite.Name,
		HasOnline: favorite.HasOnline,
		Online:    favorite.Online,
		Service:   proto.Clone(favorite.Service).(*clientsapi.Service),
	}
}

func (favorite *Favorite) Pb() *clientsapi.DeviceService {
	return &clientsapi.DeviceService{
		Key:           favorite.DeviceID + "." + favorite.ServiceID,
		DeviceId:      favorite.DeviceID,
		FullState:     true,
		HasDeviceName: favorite.HasName,
		DeviceName:    favorite.Name,
		HasOnline:     favorite.HasOnline,
		Online:        favorite.Online,
		Service:       proto.Clone(favorite.Service).(*clientsapi.Service),
	}
}
