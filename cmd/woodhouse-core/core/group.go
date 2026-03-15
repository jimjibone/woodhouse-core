package core

import (
	"fmt"
	"strings"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/apitools"
	"github.com/jimjibone/woodhouse-4/log"
	"google.golang.org/protobuf/proto"
)

type Group struct {
	GroupID   string                         `json:"group_id"`
	ServiceID string                         `json:"service_id"`
	Name      string                         `json:"name"`
	Type      clientsapi.Service_ServiceType `json:"type"`
	Members   []GroupMember                  `json:"members"`
	online    bool
	lastSeen  time.Time
	attrs     map[string]*clientsapi.Attribute // key = id
}

type GroupMember struct {
	DeviceID  string `json:"device_id"`
	ServiceID string `json:"service_id"`
	online    bool
	lastSeen  time.Time
	attrs     map[string]*clientsapi.Attribute
}

func (group *Group) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "group_id:%q, srv_id:%q, name:%q, type:%s, members:%d, attrs:%d", group.GroupID, group.ServiceID, group.Name, group.Type, len(group.Members), len(group.attrs))
	for i, member := range group.Members {
		fmt.Fprintf(&b, "\n  %d: device_id:%q, service_id:%q", i, member.DeviceID, member.ServiceID)
	}
	for _, attr := range group.attrs {
		fmt.Fprintf(&b, "\n  attr: %s", attr.String())
	}
	return b.String()
}

// Update accepts service updates from any device and may update the group if
// this group includes the device and service. The method will return the
// group's update as a clientsapi.Device, or nil if there was no update.
func (group *Group) Update(log *log.Context, deviceID string, srv *clientsapi.Service) *clientsapi.Device {
	// Make the group attrs map if not already done.
	if group.attrs == nil {
		group.attrs = make(map[string]*clientsapi.Attribute)
	}

	var update *clientsapi.Device
	addAttributeUpdate := func(attrUpdate *clientsapi.Attribute) {
		if update == nil {
			update = &clientsapi.Device{
				ClientId:  "_group",
				Id:        group.GroupID,
				FullState: false,
				Typ:       clientsapi.Device_GROUPED,
				Services: []*clientsapi.Service{
					{
						Id:    group.ServiceID,
						Typ:   group.Type,
						Attrs: nil,
					},
				},
			}
		}
		found := false
		for i, attr := range update.Services[0].Attrs {
			if attr.GetId() == attrUpdate.GetId() {
				found = true
				update.Services[0].Attrs[i] = attrUpdate
			}
		}
		if !found {
			update.Services[0].Attrs = append(update.Services[0].Attrs, attrUpdate)
		}
	}

	// Update attrs in matching group member, then compute new output attrs
	// (either mean or last value).
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

				// Calculate the 'merged' attribute value, which for now is just the latest.
				attrUpdate := proto.Clone(next).(*clientsapi.Attribute)
				group.attrs[next.GetId()] = attrUpdate

				// Output the changes to the group.
				addAttributeUpdate(attrUpdate)
			}
		}
	}

	// Return the update if any.
	return update
}

func (group *Group) UpdateInfo() *clientsapi.Device {
	update := &clientsapi.Device{
		ClientId:  "_group",
		Id:        group.GroupID,
		FullState: false,
		Typ:       clientsapi.Device_GROUPED,
		Services: []*clientsapi.Service{
			{
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
			},
		},
	}
	return update
}

func (group *Group) UpdateOnline(log *log.Context, deviceID string, srv *clientsapi.Service) *clientsapi.Device {
	var update *clientsapi.Device
	addAttributeUpdate := func(attrUpdate *clientsapi.Attribute) {
		if update == nil {
			update = &clientsapi.Device{
				ClientId:  "_group",
				Id:        group.GroupID,
				FullState: false,
				Typ:       clientsapi.Device_GROUPED,
				Services: []*clientsapi.Service{
					{
						Id:    "online",
						Typ:   clientsapi.Service_ONLINE,
						Attrs: nil,
					},
				},
			}
		}
		found := false
		for i, attr := range update.Services[0].Attrs {
			if attr.GetId() == attrUpdate.GetId() {
				found = true
				update.Services[0].Attrs[i] = attrUpdate
			}
		}
		if !found {
			update.Services[0].Attrs = append(update.Services[0].Attrs, attrUpdate)
		}
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
						addAttributeUpdate(&clientsapi.Attribute{
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
						addAttributeUpdate(&clientsapi.Attribute{
							Id:   "last_seen",
							Time: apitools.TimeToAttribute(group.lastSeen, clientsapi.Permissions_PERM_READONLY),
						})
					}
				}
			}
		}
	}

	// Return the update if any.
	return update
}

func (group *Group) Clone() *Group {
	grp := &Group{
		GroupID:   group.GroupID,
		ServiceID: group.ServiceID,
		Name:      group.Name,
		Type:      group.Type,
		Members:   []GroupMember{},
	}
	for _, member := range group.Members {
		grp.Members = append(grp.Members, GroupMember{
			DeviceID:  member.DeviceID,
			ServiceID: member.ServiceID,
		})
	}
	return grp
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
