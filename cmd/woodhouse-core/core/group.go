package core

import (
	"fmt"
	"strings"
	"time"

	"github.com/jimjibone/log"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	"github.com/jimjibone/woodhouse-core/apitools"
	"google.golang.org/protobuf/proto"
)

type Group struct {
	GroupID   string                         `json:"group_id"`
	ServiceID string                         `json:"service_id"`
	Name      string                         `json:"name"`
	Type      clientsapi.Service_ServiceType `json:"type"`
	Members   []*GroupMember                 `json:"members"`
	online    bool
	lastSeen  time.Time
	// attrs     map[string]*clientsapi.Attribute // key = id
}

type GroupMember struct {
	DeviceID  string `json:"device_id"`
	ServiceID string `json:"service_id"`
	online    bool
	lastSeen  time.Time
	attrs     map[string]*clientsapi.Attribute
}

func NewGroup(groupID, serviceID, name string, typ clientsapi.Service_ServiceType, members []*GroupMember) *Group {
	return &Group{
		GroupID:   groupID,
		ServiceID: serviceID,
		Name:      name,
		Type:      typ,
		Members:   members,
	}
}

func (group *Group) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "group_id:%q, srv_id:%q, name:%q, type:%s, members:%d", group.GroupID, group.ServiceID, group.Name, group.Type, len(group.Members))
	// fmt.Fprintf(&b, "group_id:%q, srv_id:%q, name:%q, type:%s, members:%d, attrs:%d", group.GroupID, group.ServiceID, group.Name, group.Type, len(group.Members), len(group.attrs))
	for i, member := range group.Members {
		fmt.Fprintf(&b, "\n  %d: device_id:%q, service_id:%q", i, member.DeviceID, member.ServiceID)
		for _, attr := range member.attrs {
			fmt.Fprintf(&b, "\n      attr: %s", attr.String())
		}
	}
	return b.String()
}

// func (group *Group) Clone() *Group {
// 	grp := &Group{
// 		GroupID:   group.GroupID,
// 		ServiceID: group.ServiceID,
// 		Name:      group.Name,
// 		Type:      group.Type,
// 		Members:   []GroupMember{},
// 	}
// 	for _, member := range group.Members {
// 		attrs := make(map[string]*clientsapi.Attribute)
// 		for k, v := range member.attrs {
// 			attrs[k] = proto.Clone(v).(*clientsapi.Attribute)
// 		}
// 		grp.Members = append(grp.Members, GroupMember{
// 			DeviceID:  member.DeviceID,
// 			ServiceID: member.ServiceID,
// 			online:    member.online,
// 			lastSeen:  member.lastSeen,
// 			attrs:     attrs,
// 		})
// 	}
// 	return grp
// }

// ShallowClone creates a copy of the group with the same member device and
// service IDs, but does not copy the member states or attributes.
func (group *Group) ShallowClone() *Group {
	grp := &Group{
		GroupID:   group.GroupID,
		ServiceID: group.ServiceID,
		Name:      group.Name,
		Type:      group.Type,
		Members:   []*GroupMember{},
	}
	for _, member := range group.Members {
		grp.Members = append(grp.Members, &GroupMember{
			DeviceID:  member.DeviceID,
			ServiceID: member.ServiceID,
		})
	}
	return grp
}

func (group *Group) Pb() *clientsapi.Group {
	pb := &clientsapi.Group{
		Id:        group.GroupID,
		ServiceId: group.ServiceID,
		Name:      group.Name,
		Type:      group.Type,
		Members:   []*clientsapi.GroupMember{},
	}
	for _, member := range group.Members {
		pb.Members = append(pb.Members, &clientsapi.GroupMember{
			DeviceId:  member.DeviceID,
			ServiceId: member.ServiceID,
		})
	}
	return pb
}

// func (group *Group) DevicePb() *clientsapi.Device {
// 	pb := &clientsapi.Device{
// 		ClientId:  "_group",
// 		Id:        group.GroupID,
// 		FullState: true,
// 		Typ:       clientsapi.Device_GROUPED,
// 		Services: []*clientsapi.Service{
// 			{
// 				Id:  "info",
// 				Typ: clientsapi.Service_INFO,
// 				Attrs: []*clientsapi.Attribute{
// 					&clientsapi.Attribute{
// 						Id: "name",
// 						Text: &clientsapi.TextAttribute{
// 							Value: group.Name,
// 							Perms: clientsapi.Permissions_PERM_READWRITE,
// 						},
// 					},
// 				},
// 			},
// 			{
// 				Id:    "online",
// 				Typ:   clientsapi.Service_ONLINE,
// 				Attrs: nil,
// 			},
// 		},
// 	}
// 	ourService:= &clientsapi.Service{
// 		Id:    group.ServiceID,
// 		Typ:   group.Type,
// 		Attrs: nil,
// 	}
// 	for _, member := range group.Members {
// 		// For now just use the latest value, but we could do something more complex like averaging or min/max.
// 		for _, attr := range member. {
// 			attrUpdate := proto.Clone(attr).(*clientsapi.Attribute)
// 			ourService.Attrs = append(ourService.Attrs, attrUpdate)
// 		}
// 	}
// 	pb.Services[0].Attrs = append(pb.Services[0].Attrs, ourService.Attrs...)
// 	return pb
// }
// 	found := false
// 	for i, prev := range pb.Services[0].Attrs {
// 		if prev.GetId() == attrUpdate.GetId() {
// 			found = true
// 			pb.Services[0].Attrs[i] = attrUpdate
// 		}
// 	}
// 	if !found {
// 		pb.Services[0].Attrs = append(pb.Services[0].Attrs, attrUpdate)
// 	}
// 	return pb
// }

func (group *Group) WantsDevice(deviceID string) bool {
	for _, member := range group.Members {
		if member.DeviceID == deviceID {
			return true
		}
	}
	return false
}

func (group *Group) WantsServiceUpdate(deviceID string, srv *clientsapi.Service) bool {
	if srv.GetTyp() == clientsapi.Service_ONLINE {
		for _, member := range group.Members {
			if member.DeviceID == deviceID {
				return true
			}
		}
	} else if srv.GetTyp() == group.Type {
		for _, member := range group.Members {
			if member.DeviceID == deviceID && member.ServiceID == srv.GetId() {
				return true
			}
		}
	}
	return false
}

// Update accepts service updates from any device and may update the group if
// this group includes the device and service. The method will return the
// group's update as a clientsapi.Device, or nil if there was no update.
// func (group *Group) Update(fullState bool, deviceID string, srv *clientsapi.Service) *clientsapi.Device {
// 	var update *clientsapi.Device

// 	// Add info service if fullState is set.
// 	if fullState {
// 		group.initUpdate()
// 		group.updateInfo(update)
// 	}

// 	// Update the group online states.
// 	if srv.GetTyp() == clientsapi.Service_ONLINE {
// 		initUpdate()
// 		group.updateOnline(fullState, update, deviceID, srv)
// 	}

// 	// Update the group members.
// 	if srv.GetTyp() == group.Type {
// 		initUpdate()
// 		group.updateMembers(fullState, update, deviceID, srv)
// 	}

// 	// Return the update if any.
// 	return update
// }

// Returns an initialised device update for further use with updateX methods.
func (group *Group) initUpdate(fullState bool) *clientsapi.Device {
	update := &clientsapi.Device{
		ClientId:  "_group",
		Id:        group.GroupID,
		FullState: fullState,
		Typ:       clientsapi.Device_GROUPED,
		Services:  nil,
	}
	return update
}

// Adds the info service to the device update.
func (group *Group) updateInfo(update *clientsapi.Device) {
	if update != nil {
		update.Services = append(update.Services, &clientsapi.Service{
			Id:  "info",
			Typ: clientsapi.Service_INFO,
			Attrs: []*clientsapi.Attribute{
				&clientsapi.Attribute{
					Id: "name",
					Text: &clientsapi.TextAttribute{
						Value: group.Name,
						Perms: clientsapi.Permissions_PERM_READWRITE,
					},
				},
			},
		})
	}
}

// Updates the group using online service update and adds the service to the
// device update if there was a change.
func (group *Group) updateOnline(fullState bool, update *clientsapi.Device, deviceID string, srv *clientsapi.Service) {
	// Sanity check.
	if srv.GetTyp() != clientsapi.Service_ONLINE {
		return
	}

	service := &clientsapi.Service{
		Id:    "online",
		Typ:   clientsapi.Service_ONLINE,
		Attrs: nil,
	}

	setAttribute := func(attrUpdate *clientsapi.Attribute) {
		for i, attr := range service.Attrs {
			if attr.GetId() == attrUpdate.GetId() {
				service.Attrs[i] = attrUpdate
				return
			}
		}
		service.Attrs = append(service.Attrs, attrUpdate)
	}

	// Update the member states.
	for _, member := range group.Members {
		if member.DeviceID == deviceID {
			for _, attr := range srv.Attrs {
				switch attr.Id {
				case "online":
					if member.online != attr.GetBool().GetValue() {
						member.online = attr.GetBool().GetValue()
						group.online = attr.GetBool().GetValue()
						setAttribute(&clientsapi.Attribute{
							Id: "online",
							Bool: &clientsapi.BoolAttribute{
								Value: attr.GetBool().GetValue(),
								Perms: clientsapi.Permissions_PERM_READONLY,
							},
						})
					}
				case "last_seen":
					lastSeen := apitools.AttributeToTime(attr.GetTime())
					if !member.lastSeen.Equal(lastSeen) {
						member.lastSeen = lastSeen
						group.lastSeen = lastSeen
						setAttribute(&clientsapi.Attribute{
							Id:   "last_seen",
							Time: apitools.TimeToAttribute(group.lastSeen, clientsapi.Permissions_PERM_READONLY),
						})
					}
				}
			}
		}
	}

	// If fullState is set, we need to make sure to include the online state even if it didn't change.
	if fullState {
		setAttribute(&clientsapi.Attribute{
			Id: "online",
			Bool: &clientsapi.BoolAttribute{
				Value: group.online,
				Perms: clientsapi.Permissions_PERM_READONLY,
			},
		})
		setAttribute(&clientsapi.Attribute{
			Id:   "last_seen",
			Time: apitools.TimeToAttribute(group.lastSeen, clientsapi.Permissions_PERM_READONLY),
		})
	}

	// Add the online service if there was an update.
	if update != nil && len(service.Attrs) > 0 {
		update.Services = append(update.Services, service)
	}
}

// Updates the members in the group and updates the device update if there was a change.
func (group *Group) updateMembers(fullState bool, update *clientsapi.Device, deviceID string, srv *clientsapi.Service) {
	// Sanity check.
	if srv != nil && srv.GetTyp() != group.Type {
		return
	}

	service := &clientsapi.Service{
		Id:    group.ServiceID,
		Typ:   group.Type,
		Attrs: nil,
	}

	setAttribute := func(attrUpdate *clientsapi.Attribute) {
		for i, attr := range service.Attrs {
			if attr.GetId() == attrUpdate.GetId() {
				service.Attrs[i] = attrUpdate
				return
			}
		}
		service.Attrs = append(service.Attrs, attrUpdate)
	}

	// Sometimes we call this without srv.
	if srv != nil {
		// Update attrs in matching group member, then compute new output attrs.
		updatedAttrs := make(map[string]bool) // key = attr ID
		for _, member := range group.Members {
			if member.DeviceID == deviceID && member.ServiceID == srv.GetId() {
				// Update attributes in the member.
				for _, next := range srv.Attrs {
					// Make the member attrs map if not already done.
					if member.attrs == nil {
						member.attrs = make(map[string]*clientsapi.Attribute)
					}

					// Add or update the attribute in the member.
					member.attrs[next.GetId()] = proto.Clone(next).(*clientsapi.Attribute)

					// Add the attribute ID to the list of updated attributes.
					updatedAttrs[next.GetId()] = true
				}
			}
		}

		if len(updatedAttrs) > 0 {
			// Compute new output attributes (latest value).
			lastSeen := time.Time{}
			for _, member := range group.Members {
				if lastSeen.IsZero() || member.lastSeen.After(lastSeen) {
					lastSeen = member.lastSeen

					for updated := range updatedAttrs {
						if attr, found := member.attrs[updated]; found {
							// Calculate the 'merged' attribute value, which for now is just the latest.
							attrUpdate := proto.Clone(attr).(*clientsapi.Attribute)
							// group.attrs[next.GetId()] = attrUpdate

							// Output the changes to the group.
							setAttribute(attrUpdate)
						}
					}
				}
			}
		}
	}

	// If fullState is set, we need to make sure to include the latest value of all member attributes even if they didn't change.
	if fullState {
		for _, member := range group.Members {
			for _, attr := range member.attrs {
				attrUpdate := proto.Clone(attr).(*clientsapi.Attribute)
				setAttribute(attrUpdate)
			}
		}
	}

	// Add the group service if there was an update.
	if update != nil && len(service.Attrs) > 0 {
		update.Services = append(update.Services, service)
	}
}

func (group *Group) removeDevice(update *clientsapi.Device, deviceID string) {
	// Remove the member then trigger a full state update.
	newMembers := make([]*GroupMember, 0, len(group.Members))
	for _, member := range group.Members {
		if member.DeviceID != deviceID {
			newMembers = append(newMembers, member)
		}
	}
	group.Members = newMembers

	group.updateMembers(true, update, "", nil)
}

func (group *Group) HandleRequest(req *clientsapi.ActionRequest, deviceManager *DeviceManager) {
	sub := deviceManager.GetActionResponses()
	defer sub.Close()

	// Check that the request is for this group.
	if req.GetDeviceId() == group.GroupID && req.GetServiceId() == group.ServiceID {
		log := log.NewContext(log.DefaultLogger, "group:"+group.GroupID, log.DebugLevel)
		log.Debugf("action request started: %s", req)
		defer log.Debugf("action request finished: %s", req)

		actionIDs := make(map[string]string) // key=actionID, value=clientID

		// Forward the request to all members.
		for _, member := range group.Members {
			actionID, clientID, err := deviceManager.PrepAction(member.DeviceID)
			if err != nil {
				// If there was an error cancel the action.
				log.Errorf("failed to prep action for member %q: %s", member.DeviceID, err)
				deviceManager.PushActionResponse(GroupClientID, &clientsapi.ActionResponse{
					ActionId: req.GetActionId(),
					Status:   clientsapi.ActionResponse_ERROR,
					Details:  err.Error(),
				}, false)
				return
			}

			// Add the action ID to the list.
			actionIDs[actionID] = clientID

			log.Debugf("member: %q, request: %s", member.DeviceID, actionID)
			deviceManager.PushActionRequest(clientID, &clientsapi.ActionRequest{
				ActionId:  actionID,
				DeviceId:  member.DeviceID,
				ServiceId: member.ServiceID,
				Values:    req.GetValues(),
			})
		}

		// Let the requester know the the actions were sent to members.
		deviceManager.PushActionResponse(GroupClientID, &clientsapi.ActionResponse{
			ActionId: req.GetActionId(),
			Status:   clientsapi.ActionResponse_SENT,
			Details:  "",
		}, false)

		// Wait for all member requests to finish.
		for response := range sub.Sub() {
			if response.Response != nil {
				if _, found := actionIDs[response.Response.ActionId]; found {
					log.Debugf("member response: %s", response.Response)
					if response.Response.Status >= clientsapi.ActionResponse_COMPLETE {
						delete(actionIDs, response.Response.ActionId)
						if response.Response.Status >= clientsapi.ActionResponse_TIMEOUT {
							deviceManager.PushActionResponse(GroupClientID, &clientsapi.ActionResponse{
								ActionId: req.GetActionId(),
								Status:   response.Response.Status,
								Details:  response.Response.Details,
							}, false)
						}
					}
				}
			} else if response.Offline {
				// Is the client going offline related to one of our actions?
				for actionID, clientID := range actionIDs {
					if clientID == response.ClientID {
						log.Debugf("member offline with pending action: %s", response.ClientID)
						delete(actionIDs, actionID)
						deviceManager.PushActionResponse(GroupClientID, &clientsapi.ActionResponse{
							ActionId: req.GetActionId(),
							Status:   clientsapi.ActionResponse_CANCELED,
							Details:  "client went offline",
						}, false)
					}
				}
			}
			if len(actionIDs) == 0 {
				break
			}
		}
		sub.Close()
		log.Debugf("finished waiting for responses")

		// Let the requester know the the actions were sent to members.
		deviceManager.PushActionResponse(GroupClientID, &clientsapi.ActionResponse{
			ActionId: req.GetActionId(),
			Status:   clientsapi.ActionResponse_COMPLETE,
			Details:  "",
		}, false)
	}
}
