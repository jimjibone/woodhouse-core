package core

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/jimjibone/log"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v2/clients"
	"github.com/jimjibone/woodhouse-api/values"
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
}

type GroupMember struct {
	DeviceID   string `json:"device_id"`
	EndpointID string `json:"endpoint_id"`
	ServiceID  string `json:"service_id"`
	online     bool
	lastSeen   time.Time
	attrs      map[string]*clientsapi.Attribute

	// The member service's features and commands from its last update. known is
	// false until the first update.
	known    bool
	features []string
	commands []*clientsapi.CommandSpec
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
	for i, member := range group.Members {
		fmt.Fprintf(&b, "\n  %d: device_id:%q, endpoint_id:%q, service_id:%q", i, member.DeviceID, member.EndpointID, member.ServiceID)
		for _, attr := range member.attrs {
			fmt.Fprintf(&b, "\n      attr: %s", attr.String())
		}
	}
	return b.String()
}

// ShallowClone creates a copy of the group with the same member device,
// endpoint and service IDs, but does not copy the member states or attributes.
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
			DeviceID:   member.DeviceID,
			EndpointID: member.EndpointID,
			ServiceID:  member.ServiceID,
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
			DeviceId:   member.DeviceID,
			EndpointId: member.EndpointID,
			ServiceId:  member.ServiceID,
		})
	}
	return pb
}

func (group *Group) WantsDevice(deviceID string) bool {
	for _, member := range group.Members {
		if member.DeviceID == deviceID {
			return true
		}
	}
	return false
}

// GroupInfoServiceID is the service ID of the synthetic info service emitted
// for every group.
const GroupInfoServiceID = "info"

const groupOnlineServiceID = "online"

// Returns an initialised device update for further use with updateX methods.
// A group's device has one endpoint, the root, holding every service.
func (group *Group) initUpdate(fullState bool) *clientsapi.Device {
	return &clientsapi.Device{
		ClientId:  GroupClientID,
		Id:        group.GroupID,
		FullState: fullState,
		Typ:       clientsapi.Device_GROUPED,
		Endpoints: []*clientsapi.Endpoint{{Id: RootEndpointID}},
	}
}

func addService(update *clientsapi.Device, srv *clientsapi.Service) {
	ep := update.Endpoints[0]
	ep.Services = append(ep.Services, srv)
}

// Adds the info service to the device update.
func (group *Group) updateInfo(update *clientsapi.Device) {
	if update != nil {
		addService(update, &clientsapi.Service{
			Id:  GroupInfoServiceID,
			Typ: clientsapi.Service_INFO,
			Attrs: []*clientsapi.Attribute{
				{
					Id:    "name",
					Perms: clientsapi.Permissions_PERM_READWRITE,
					Spec:  &clientsapi.Spec{Kind: &clientsapi.Spec_Text{Text: &clientsapi.TextSpec{}}},
					Value: values.Text(group.Name),
				},
			},
		})
	}
}

func onlineAttr(online bool) *clientsapi.Attribute {
	return &clientsapi.Attribute{
		Id:    "online",
		Perms: clientsapi.Permissions_PERM_READONLY,
		Spec:  &clientsapi.Spec{Kind: &clientsapi.Spec_Bool{Bool: &clientsapi.BoolSpec{}}},
		Value: values.Bool(online),
	}
}

func lastSeenAttr(lastSeen time.Time) *clientsapi.Attribute {
	return &clientsapi.Attribute{
		Id:    "last_seen",
		Perms: clientsapi.Permissions_PERM_READONLY,
		Spec:  &clientsapi.Spec{Kind: &clientsapi.Spec_Time{Time: &clientsapi.TimeSpec{}}},
		Value: values.Time(lastSeen),
	}
}

// setAttribute adds or replaces the attribute in attrs.
func setAttribute(attrs []*clientsapi.Attribute, attrUpdate *clientsapi.Attribute) []*clientsapi.Attribute {
	for i, attr := range attrs {
		if attr.GetId() == attrUpdate.GetId() {
			attrs[i] = attrUpdate
			return attrs
		}
	}
	return append(attrs, attrUpdate)
}

// onlineAttrs returns the group's online and last_seen attributes.
func (group *Group) onlineAttrs() []*clientsapi.Attribute {
	return []*clientsapi.Attribute{onlineAttr(group.online), lastSeenAttr(group.lastSeen)}
}

// Updates the member online states using an online service update and returns
// the group's online attributes that changed.
func (group *Group) updateOnline(deviceID string, srv *clientsapi.Service) []*clientsapi.Attribute {
	var changed []*clientsapi.Attribute
	for _, member := range group.Members {
		if member.DeviceID != deviceID {
			continue
		}
		for _, attr := range srv.GetAttrs() {
			switch attr.GetId() {
			case "online":
				online := attr.GetValue().GetBool()
				if member.online != online {
					member.online = online
					group.online = online
					changed = setAttribute(changed, onlineAttr(online))
				}
			case "last_seen":
				lastSeen, _ := values.AsTime(attr.GetValue())
				if !member.lastSeen.Equal(lastSeen) {
					member.lastSeen = lastSeen
					group.lastSeen = lastSeen
					changed = setAttribute(changed, lastSeenAttr(group.lastSeen))
				}
			}
		}
	}
	return changed
}

// Updates the members matching the device, endpoint and service and returns
// whether any matched and the group's attributes that changed.
func (group *Group) updateMembers(deviceID, endpointID string, srv *clientsapi.Service) (bool, []*clientsapi.Attribute) {
	matched := false
	updatedAttrs := make(map[string]bool) // key = attr ID
	for _, member := range group.Members {
		if member.DeviceID != deviceID || member.EndpointID != endpointID || member.ServiceID != srv.GetId() {
			continue
		}
		matched = true
		member.known = true
		member.features = slices.Clone(srv.GetFeatures())
		member.commands = slices.Clone(srv.GetCommands())

		// Update attributes in the member.
		for _, next := range srv.GetAttrs() {
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

	// Compute new output attributes (latest value).
	var changed []*clientsapi.Attribute
	if len(updatedAttrs) > 0 {
		lastSeen := time.Time{}
		for _, member := range group.Members {
			if lastSeen.IsZero() || member.lastSeen.After(lastSeen) {
				lastSeen = member.lastSeen

				for updated := range updatedAttrs {
					if attr, found := member.attrs[updated]; found {
						// Calculate the 'merged' attribute value, which for now is just the latest.
						changed = setAttribute(changed, proto.Clone(attr).(*clientsapi.Attribute))
					}
				}
			}
		}
	}
	return matched, changed
}

// memberAttrs returns the latest value of every member attribute.
func (group *Group) memberAttrs() []*clientsapi.Attribute {
	var attrs []*clientsapi.Attribute
	for _, member := range group.Members {
		ids := make([]string, 0, len(member.attrs))
		for id := range member.attrs {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			attrs = setAttribute(attrs, proto.Clone(member.attrs[id]).(*clientsapi.Attribute))
		}
	}
	return attrs
}

// serviceInfo returns the features and commands of the group service. Features
// are those every member has. Commands are the specs, taken from the first
// member, whose ID every member has. Members that have not reported yet are
// ignored. The group service has no events.
func (group *Group) serviceInfo() (features []string, commands []*clientsapi.CommandSpec) {
	var known []*GroupMember
	for _, member := range group.Members {
		if member.known {
			known = append(known, member)
		}
	}
	if len(known) == 0 {
		return nil, nil
	}

	for _, feature := range known[0].features {
		if slices.ContainsFunc(known[1:], func(m *GroupMember) bool { return !slices.Contains(m.features, feature) }) {
			continue
		}
		features = append(features, feature)
	}
	for _, command := range known[0].commands {
		hasCommand := func(m *GroupMember) bool {
			return slices.ContainsFunc(m.commands, func(c *clientsapi.CommandSpec) bool { return c.GetId() == command.GetId() })
		}
		if slices.ContainsFunc(known[1:], func(m *GroupMember) bool { return !hasCommand(m) }) {
			continue
		}
		commands = append(commands, command)
	}
	return features, commands
}

// service returns the group service with the attributes and the current
// features and commands.
func (group *Group) service(attrs []*clientsapi.Attribute) *clientsapi.Service {
	features, commands := group.serviceInfo()
	return &clientsapi.Service{
		Id:       group.ServiceID,
		Typ:      group.Type,
		Attrs:    attrs,
		Features: features,
		Commands: commands,
	}
}

// fullUpdate returns the full state of the group's device: the info, online
// and group services. Members must have been fed their device state first with
// Update.
func (group *Group) fullUpdate() *clientsapi.Device {
	update := group.initUpdate(true)
	group.updateInfo(update)
	addService(update, &clientsapi.Service{
		Id:    groupOnlineServiceID,
		Typ:   clientsapi.Service_ONLINE,
		Attrs: group.onlineAttrs(),
	})
	addService(update, group.service(group.memberAttrs()))
	return update
}

// Update accepts device updates from any device and, if the group has members
// on the device, updates the group and returns the group's update, or nil if
// there was no update.
func (group *Group) Update(dev *clientsapi.Device) *clientsapi.Device {
	if !group.WantsDevice(dev.GetId()) {
		return nil
	}

	var onlineAttrs, memberAttrs []*clientsapi.Attribute
	matched := false
	for _, ep := range dev.GetEndpoints() {
		for _, srv := range ep.GetServices() {
			if ep.GetId() == RootEndpointID && srv.GetTyp() == clientsapi.Service_ONLINE {
				for _, attr := range group.updateOnline(dev.GetId(), srv) {
					onlineAttrs = setAttribute(onlineAttrs, attr)
				}
			}
			if srv.GetTyp() == group.Type {
				ok, changed := group.updateMembers(dev.GetId(), ep.GetId(), srv)
				matched = matched || ok
				for _, attr := range changed {
					memberAttrs = setAttribute(memberAttrs, attr)
				}
			}
		}
	}
	if len(onlineAttrs) == 0 && !matched {
		return nil
	}

	update := group.initUpdate(false)
	if len(onlineAttrs) > 0 {
		addService(update, &clientsapi.Service{
			Id:    groupOnlineServiceID,
			Typ:   clientsapi.Service_ONLINE,
			Attrs: onlineAttrs,
		})
	}
	if matched {
		addService(update, group.service(memberAttrs))
	}
	return update
}

// removeDevice removes the device's members and returns the group's update.
func (group *Group) removeDevice(deviceID string) *clientsapi.Device {
	group.Members = slices.DeleteFunc(group.Members, func(member *GroupMember) bool {
		return member.DeviceID == deviceID
	})

	update := group.initUpdate(false)
	addService(update, group.service(group.memberAttrs()))
	return update
}

func (group *Group) HandleRequest(req *clientsapi.ActionRequest, deviceManager *DeviceManager) {
	sub := deviceManager.GetActionResponses()
	defer sub.Close()

	// Check that the request is for this group.
	if req.GetDeviceId() == group.GroupID && req.GetEndpointId() == RootEndpointID && req.GetServiceId() == group.ServiceID {
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

			// The write or command is forwarded as it is.
			memberReq := proto.Clone(req).(*clientsapi.ActionRequest)
			memberReq.ActionId = actionID
			memberReq.DeviceId = member.DeviceID
			memberReq.EndpointId = member.EndpointID
			memberReq.ServiceId = member.ServiceID

			log.Debugf("member: %q, request: %s", member.DeviceID, actionID)
			deviceManager.PushActionRequest(clientID, memberReq)
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

		// Member command results are not merged: the group completes without a result.
		deviceManager.PushActionResponse(GroupClientID, &clientsapi.ActionResponse{
			ActionId: req.GetActionId(),
			Status:   clientsapi.ActionResponse_COMPLETE,
			Details:  "",
		}, false)
	}
}
