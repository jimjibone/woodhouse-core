package core

import (
	"fmt"
	"sort"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
)

type Device struct {
	ClientID string
	ID       string
	Typ      clientsapi.Device_DeviceType
	Services map[string]*clientsapi.Service
}

func newDevice(log *log.Context, clientID string, update *clientsapi.Device) (*Device, error) {
	dev := &Device{
		ClientID: clientID,
		ID:       update.GetId(),
		Services: make(map[string]*clientsapi.Service),
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

func (dev *Device) pb() *clientsapi.Device {
	pb := &clientsapi.Device{
		Id:        dev.ID,
		FullState: true,
		Typ:       dev.Typ,
		Services:  []*clientsapi.Service{},
	}
	for _, srv := range dev.Services {
		// Sorted to maintain consistent structure between saves.
		sort.Slice(srv.Attrs, func(i, j int) bool {
			return srv.Attrs[i].GetId() < srv.Attrs[j].GetId()
		})
		pb.Services = append(pb.Services, srv)
	}
	// Sorted to maintain consistent structure between saves.
	sort.Slice(pb.Services, func(i, j int) bool {
		return pb.Services[i].GetId() < pb.Services[j].GetId()
	})
	return pb
}

func (dev *Device) isOnline() bool {
	for _, srv := range dev.Services {
		if srv.Typ == clientsapi.Service_ONLINE {
			for _, attr := range srv.Attrs {
				if attr.GetId() == "online" && attr.GetBool() != nil {
					return attr.Bool.Value
				}
			}
		}
	}
	return false
}

func (dev *Device) setOffline(log *log.Context) *clientsapi.Device {
	for _, srv := range dev.Services {
		if srv.Typ == clientsapi.Service_ONLINE {
			for _, attr := range srv.Attrs {
				if attr.GetId() == "online" && attr.GetBool() != nil {
					attr.Bool.Value = false
				}
			}
			if log != nil {
				log.Debugf("device %q went offline\n%s", dev.ID, prettyService("  ", srv))
			}
			return &clientsapi.Device{
				Id:        dev.ID,
				FullState: false,
				Typ:       dev.Typ,
				Services: []*clientsapi.Service{
					srv,
				},
			}
		}
	}
	return nil
}

// Update the device state and return the changes or an error.
func (dev *Device) update(log *log.Context, clientID string, update *clientsapi.Device) (*clientsapi.Device, error) {
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
	if update.FullState {
		dev.gcServices(log, update)
	}
	var changes []*clientsapi.Service
	for _, srv := range update.Services {
		srvChanges, err := dev.updateService(log, update.FullState, srv)
		if err != nil {
			return nil, err
		}
		changes = append(changes, srvChanges)
	}
	update.Services = changes
	return update, nil
}

// Garbage collect services no longer reported by this device.
func (dev *Device) gcServices(log *log.Context, update *clientsapi.Device) error {
	for srvID, srv := range dev.Services {
		found := false
		for _, upd := range update.Services {
			if upd.GetId() == srvID {
				found = true
				break
			}
		}
		if !found {
			if log != nil {
				log.Debugf("device %q removed service\n%s", dev.ID, prettyService("  ", srv))
			}
			delete(dev.Services, srvID)
		}
	}
	return nil
}

// Add or update a service and return the new state or an error.
func (dev *Device) updateService(log *log.Context, fullState bool, update *clientsapi.Service) (*clientsapi.Service, error) {
	if srv, found := dev.Services[update.GetId()]; found {
		// if log != nil {
		// 	log.Debugf("device %q updated service (update)\n%s", dev.ID, prettyService("  ", update))
		// }

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
					log.Debugf("device %q service %q removed attribute %q", dev.ID, srv.GetId(), attr.GetId())
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

		if log != nil {
			log.Debugf("device %q updated service\n%s", dev.ID, prettyService("  ", srv))
		}
		return srv, nil
	}
	if log != nil {
		log.Debugf("device %q added service\n%s", dev.ID, prettyService("  ", update))
	}
	dev.Services[update.GetId()] = update
	return update, nil
}

func (dev *Device) updateFavoriteService(log *log.Context, serviceID string, favorite bool) error {
	if srv, found := dev.Services[serviceID]; found {
		if log != nil {
			if favorite {
				log.Debugf("device %q added service favorite: %s", dev.ID, serviceID)
			} else {
				log.Debugf("device %q removed service favorite: %s", dev.ID, serviceID)
			}
		}

		// Update service favorite state.
		srv.Favorite = favorite

		return nil
	}
	return nil
}

func prettyService(pad string, srv *clientsapi.Service) string {
	str := fmt.Sprintf("%s- srv id:%q, typ:%q, alias:%q, attrs:%d, fave:%t", pad, srv.GetId(), srv.GetTyp(), srv.GetAlias(), len(srv.Attrs), srv.GetFavorite())
	for _, attr := range srv.Attrs {
		str += fmt.Sprintf("\n%s  - attr %s", pad, attr)
	}
	return str
}
