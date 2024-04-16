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
	err := dev.update(log, clientID, update)
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
				FullState: true,
				Typ:       dev.Typ,
				Services: []*clientsapi.Service{
					srv,
				},
			}
		}
	}
	return nil
}

func (dev *Device) update(log *log.Context, clientID string, update *clientsapi.Device) error {
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
	for _, srv := range update.Services {
		err := dev.updateService(log, update.FullState, srv)
		if err != nil {
			return err
		}
	}
	return nil
}

// Garbage collect services no longer reported by this device.
func (dev *Device) gcServices(log *log.Context, update *clientsapi.Device) error {
	for srvID, _ := range dev.Services {
		found := false
		for _, upd := range update.Services {
			if upd.GetId() == srvID {
				found = true
				break
			}
		}
		if !found {
			if log != nil {
				log.Debugf("device %q removed service\n%s", dev.ID, prettyService("  ", dev.Services[srvID]))
			}
			delete(dev.Services, srvID)
		}
	}
	return nil
}

// Add or update a service.
func (dev *Device) updateService(log *log.Context, fullState bool, update *clientsapi.Service) error {
	if srv, found := dev.Services[update.GetId()]; found {
		if log != nil {
			log.Debugf("device %q updated service\n%s", dev.ID, prettyService("  ", update))
		}
		srv.Typ = update.GetTyp()
		srv.Alias = update.GetAlias()
		for _, upd := range update.Attrs {
			found := false
			for i, attr := range srv.Attrs {
				if attr.GetId() == upd.GetId() {
					found = true
					srv.Attrs[i] = upd
				}
			}
			if !found {
				srv.Attrs = append(srv.Attrs, upd)
			}
		}
		return nil
	}
	if log != nil {
		log.Debugf("device %q added service\n%s", dev.ID, prettyService("  ", update))
	}
	dev.Services[update.GetId()] = update
	return nil
}

func prettyServices(pad string, srvs []*clientsapi.Service) string {
	str := ""
	for i, srv := range srvs {
		if i > 0 {
			str += "\n"
		}
		str += prettyService(pad, srv)
	}
	return str
}

func prettyService(pad string, srv *clientsapi.Service) string {
	str := fmt.Sprintf("%s- srv id:%q, typ:%q, alias:%q, attrs:%d", pad, srv.GetId(), srv.GetTyp(), srv.GetAlias(), len(srv.Attrs))
	for _, attr := range srv.Attrs {
		str += fmt.Sprintf("\n%s  - attr %s", pad, attr)
	}
	return str
}
