package users

import (
	"context"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/apitools"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/clients"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/random"
	"github.com/jimjibone/woodhouse-4/wh/v1/devices/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserService struct {
	clientsapi.UnimplementedUserServiceServer
	log              *log.Context
	deviceManager    *core.DeviceManager
	favoritesManager *core.FavoritesManager
	groupManager     *core.GroupManager
	userManager      *core.UserManager
	clientManager    *core.ClientManager
	clientJwt        *clients.JWTManager
}

func NewUserService(deviceManager *core.DeviceManager, favoritesManager *core.FavoritesManager, groupManager *core.GroupManager, userManager *core.UserManager, clientManager *core.ClientManager, clientJwt *clients.JWTManager) *UserService {
	service := &UserService{
		log:              log.NewContext(log.DefaultLogger, "user-service", log.DebugLevel),
		deviceManager:    deviceManager,
		favoritesManager: favoritesManager,
		groupManager:     groupManager,
		userManager:      userManager,
		clientManager:    clientManager,
		clientJwt:        clientJwt,
	}
	return service
}

func (service *UserService) GetClients(req *clientsapi.GetClientsRequest, server clientsapi.UserService_GetClientsServer) error {
	claims := server.Context().Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return status.Errorf(codes.PermissionDenied, "no claims in request")
	}
	if claims.Role != auth.AdminRole {
		return status.Errorf(codes.PermissionDenied, "not allowed to view clients")
	}
	if service.clientManager == nil {
		return status.Errorf(codes.FailedPrecondition, "client manager not configured")
	}

	clients := service.clientManager.GetClients()
	for _, client := range clients {
		err := server.Send(client.Pb())
		if err != nil {
			service.log.Errorf("failed to send clients: %s", err)
			return status.Errorf(codes.Internal, "failed to send clients")
		}
	}
	return nil
}

func (service *UserService) ClientsStream(req *clientsapi.ClientsStreamRequest, server clientsapi.UserService_ClientsStreamServer) error {
	claims := server.Context().Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return status.Errorf(codes.PermissionDenied, "no claims in request")
	}
	if claims.Role != auth.AdminRole {
		return status.Errorf(codes.PermissionDenied, "not allowed to view clients")
	}
	if service.clientManager == nil {
		return status.Errorf(codes.FailedPrecondition, "client manager not configured")
	}

	service.log.Infof("clients stream started")
	defer service.log.Infof("clients stream finished")

	sub := service.clientManager.GetClientListener()
	defer sub.Close()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-server.Context().Done():
			return status.Errorf(codes.Canceled, "context canceled")

		case <-ticker.C:
			err := server.Send(&clientsapi.ClientsStreamResponse{})
			if err != nil {
				service.log.Errorf("failed to send clients stream keepalive: %s", err)
				return status.Errorf(codes.Internal, "failed to send keepalive")
			}

		case update := <-sub.Sub():
			if update.Updated != nil {
				err := server.Send(&clientsapi.ClientsStreamResponse{Client: update.Updated.Pb()})
				if err != nil {
					service.log.Errorf("failed to send clients stream update: %s", err)
					return status.Errorf(codes.Internal, "failed to send update")
				}
			}
			if update.Removed != nil {
				err := server.Send(&clientsapi.ClientsStreamResponse{ClientRemoved: *update.Removed})
				if err != nil {
					service.log.Errorf("failed to send clients stream removal: %s", err)
					return status.Errorf(codes.Internal, "failed to send removal")
				}
			}
		}
	}
}

func (service *UserService) PairingRequestsStream(req *clientsapi.PairingRequestsStreamRequest, server clientsapi.UserService_PairingRequestsStreamServer) error {
	claims := server.Context().Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return status.Errorf(codes.PermissionDenied, "no claims in request")
	}
	if claims.Role != auth.AdminRole {
		return status.Errorf(codes.PermissionDenied, "not allowed to view pairing requests")
	}
	if service.clientManager == nil {
		return status.Errorf(codes.FailedPrecondition, "client manager not configured")
	}

	service.log.Infof("pairing requests stream started")
	defer service.log.Infof("pairing requests stream finished")

	sub := service.clientManager.GetPairingListener()
	defer sub.Close()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-server.Context().Done():
			return status.Errorf(codes.Canceled, "context canceled")

		case <-ticker.C:
			err := server.Send(&clientsapi.PairingRequestsStreamResponse{})
			if err != nil {
				service.log.Errorf("failed to send pairing requests stream keepalive: %s", err)
				return status.Errorf(codes.Internal, "failed to send keepalive")
			}

		case update := <-sub.Sub():
			msg := &clientsapi.PairingRequestsStreamResponse{}
			if update.Updated != nil {
				msg.PairingRequest = update.Updated.Pb()
			}
			if update.Removed != nil {
				msg.PairingRemoved = *update.Removed
			}
			err := server.Send(msg)
			if err != nil {
				service.log.Errorf("failed to send pairing requests stream update: %s", err)
				return status.Errorf(codes.Internal, "failed to send update")
			}
		}
	}
}

func (service *UserService) ApprovePairing(ctx context.Context, req *clientsapi.ApprovePairingRequest) (*clientsapi.ApprovePairingResponse, error) {
	claims := ctx.Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return nil, status.Errorf(codes.PermissionDenied, "no claims in request")
	}
	if claims.Role != auth.AdminRole {
		return nil, status.Errorf(codes.PermissionDenied, "not allowed to approve pairing")
	}
	if req.GetClientId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "client_id not defined")
	}
	if req.GetPairingCode() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "pairing_code not defined")
	}
	if service.clientManager == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "client manager not configured")
	}

	if err := service.clientManager.ApprovePairingRequest(req.GetClientId(), req.GetPairingCode()); err != nil {
		if err == core.ErrPairingNotFound {
			return nil, status.Errorf(codes.NotFound, "pairing request not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to approve pairing")
	}

	return &clientsapi.ApprovePairingResponse{}, nil
}

func (service *UserService) DenyPairing(ctx context.Context, req *clientsapi.DenyPairingRequest) (*clientsapi.DenyPairingResponse, error) {
	claims := ctx.Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return nil, status.Errorf(codes.PermissionDenied, "no claims in request")
	}
	if claims.Role != auth.AdminRole {
		return nil, status.Errorf(codes.PermissionDenied, "not allowed to deny pairing")
	}
	if req.GetClientId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "client_id not defined")
	}
	if service.clientManager == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "client manager not configured")
	}

	if err := service.clientManager.RemovePairingRequest(req.GetClientId()); err != nil {
		if err == core.ErrPairingNotFound {
			return nil, status.Errorf(codes.NotFound, "pairing request not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to deny pairing")
	}

	return &clientsapi.DenyPairingResponse{}, nil
}

func (service *UserService) UnpairClient(ctx context.Context, req *clientsapi.UnpairClientRequest) (*clientsapi.UnpairClientResponse, error) {
	claims := ctx.Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return nil, status.Errorf(codes.PermissionDenied, "no claims in request")
	}
	if claims.Role != auth.AdminRole {
		return nil, status.Errorf(codes.PermissionDenied, "not allowed to unpair client")
	}
	if req.GetClientId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "client_id not defined")
	}
	if service.clientManager == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "client manager not configured")
	}

	if err := service.clientManager.SetClientPaired(req.GetClientId(), false); err != nil {
		if err == core.ErrClientNotFound {
			return nil, status.Errorf(codes.NotFound, "client not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to unpair client")
	}

	if service.clientJwt == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "client jwt manager not configured")
	}
	service.clientJwt.RevokeClient(req.GetClientId())

	return &clientsapi.UnpairClientResponse{}, nil
}

func (service *UserService) ForgetClient(ctx context.Context, req *clientsapi.ForgetClientRequest) (*clientsapi.ForgetClientResponse, error) {
	claims := ctx.Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return nil, status.Errorf(codes.PermissionDenied, "no claims in request")
	}
	if claims.Role != auth.AdminRole {
		return nil, status.Errorf(codes.PermissionDenied, "not allowed to forget client")
	}
	if req.GetClientId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "client_id not defined")
	}
	if service.clientManager == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "client manager not configured")
	}
	if service.clientJwt == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "client jwt manager not configured")
	}

	if err := service.clientManager.ForgetClient(req.GetClientId()); err != nil {
		if err == core.ErrClientNotFound {
			return nil, status.Errorf(codes.NotFound, "client not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to forget client")
	}

	service.clientJwt.RevokeClient(req.GetClientId())

	return &clientsapi.ForgetClientResponse{}, nil
}

func (service *UserService) GetDevices(req *clientsapi.GetDevicesRequest, server clientsapi.UserService_GetDevicesServer) error {
	devices := service.deviceManager.GetDevices()
	for dev := range devices {
		err := server.Send(dev)
		if err != nil {
			service.log.Errorf("failed to send devices: %s", err)
			return status.Errorf(codes.Internal, "failed to send devices")
		}
	}
	return nil
}

func (service *UserService) DevicesStream(req *clientsapi.DevicesStreamRequest, server clientsapi.UserService_DevicesStreamServer) error {
	service.log.Infof("devices stream started")
	defer service.log.Infof("devices stream finished")

	sub := service.deviceManager.GetDeviceUpdates()
	defer sub.Close()

	isInFilter := func(deviceID string, filter []string) bool {
		if len(filter) == 0 {
			return true
		}
		for _, id := range filter {
			if deviceID == id {
				return true
			}
		}
		return false
	}

	// Start by sending the full list of devices.
	for dev := range service.deviceManager.GetDevices() {
		if isInFilter(dev.GetId(), req.IncludeDeviceIds) {
			err := server.Send(&clientsapi.DevicesStreamResponse{
				Device: dev,
			})
			if err != nil {
				service.log.Errorf("failed to send device stream: %s", err)
				return status.Errorf(codes.Internal, "failed to send device")
			}
		}
	}

	// Send an empty device to indicate the end of the initial list.
	err := server.Send(&clientsapi.DevicesStreamResponse{})
	if err != nil {
		service.log.Errorf("failed to send empty device: %s", err)
		return status.Errorf(codes.Internal, "failed to send empty device")
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Now send updates.
	for {
		select {
		case <-server.Context().Done():
			return status.Errorf(codes.Canceled, "context canceled")

		case <-ticker.C:
			// Send an empty Device message as a keepalive for the client.
			err := server.Send(&clientsapi.DevicesStreamResponse{})
			if err != nil {
				service.log.Errorf("failed to send device stream keepalive: %s", err)
				return status.Errorf(codes.Internal, "failed to send device keepalive")
			}

		case update := <-sub.Sub():
			if update.Update != nil {
				if isInFilter(update.Update.GetId(), req.IncludeDeviceIds) {
					err := server.Send(&clientsapi.DevicesStreamResponse{
						Device: update.Update,
					})
					if err != nil {
						service.log.Errorf("failed to send device stream update: %s", err)
						return status.Errorf(codes.Internal, "failed to send device update")
					}
				}
			} else if update.RemovedID != "" {
				if isInFilter(update.RemovedID, req.IncludeDeviceIds) {
					err := server.Send(&clientsapi.DevicesStreamResponse{
						DeviceRemoved: update.RemovedID,
					})
					if err != nil {
						service.log.Errorf("failed to send device stream remove update: %s", err)
						return status.Errorf(codes.Internal, "failed to send device remove update")
					}
				}
			}
		}
	}
}

func (service *UserService) RemoveDevice(ctx context.Context, req *clientsapi.RemoveDeviceRequest) (*clientsapi.RemoveDeviceResponse, error) {
	claims := ctx.Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return nil, status.Errorf(codes.PermissionDenied, "no claims in request")
	}
	if claims.Role != auth.AdminRole {
		return nil, status.Errorf(codes.PermissionDenied, "not allowed to remove device")
	}
	if req.GetDeviceId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "device_id not defined")
	}
	if service.deviceManager == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "device manager not configured")
	}

	if err := service.deviceManager.RemoveDevice(req.GetDeviceId(), false); err != nil {
		if err == core.ErrDeviceIsGroup {
			// Try using the group removal which will handle the group device.
			_, err := service.RemoveGroup(ctx, &clientsapi.RemoveGroupRequest{
				Id: req.GetDeviceId(),
			})
			if err != nil {
				return nil, err
			}
			return &clientsapi.RemoveDeviceResponse{}, nil
		}

		if err == core.ErrDeviceNotFound {
			return nil, status.Errorf(codes.NotFound, "device not found")
		}
		if err == core.ErrDeviceIsOnline {
			return nil, status.Errorf(codes.FailedPrecondition, "device is online, cannot remove")
		}
		return nil, status.Errorf(codes.Internal, "failed to remove device: %s", err)
	}

	return &clientsapi.RemoveDeviceResponse{}, nil
}

func (service *UserService) FavoritesStream(req *clientsapi.FavoritesStreamRequest, server clientsapi.UserService_FavoritesStreamServer) error {
	service.log.Infof("favorites stream started")
	defer service.log.Infof("favorites stream finished")

	lis := service.favoritesManager.GetListener()
	defer lis.Close()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-server.Context().Done():
			return status.Errorf(codes.Canceled, "context canceled")

		case <-ticker.C:
			// Send an empty Device message as a keepalive for the client.
			err := server.Send(&clientsapi.FavoritesStreamResponse{})
			if err != nil {
				service.log.Errorf("failed to send favorite stream keepalive: %s", err)
				return status.Errorf(codes.Internal, "failed to send keepalive")
			}

		case update := <-lis.Sub():
			msg := &clientsapi.FavoritesStreamResponse{}
			if update.Updated != nil {
				msg.DeviceService = update.Updated.Pb()
			}
			if update.Removed != nil {
				msg.KeyRemoved = update.Removed.Key()
			}
			err := server.Send(msg)
			if err != nil {
				service.log.Errorf("failed to send favorite stream update: %s", err)
				return status.Errorf(codes.Internal, "failed to send update")
			}
		}
	}
}

func (service *UserService) AddFavorite(ctx context.Context, req *clientsapi.AddFavoriteRequest) (*clientsapi.AddFavoriteResponse, error) {
	if req.GetDeviceId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "device_id not defined")
	}
	if req.GetServiceId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "service_id not defined")
	}
	service.deviceManager.SetFavorite(req.DeviceId, req.ServiceId, true)
	return &clientsapi.AddFavoriteResponse{}, nil
}

func (service *UserService) RemoveFavorite(ctx context.Context, req *clientsapi.RemoveFavoriteRequest) (*clientsapi.RemoveFavoriteResponse, error) {
	if req.GetDeviceId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "device_id not defined")
	}
	if req.GetServiceId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "service_id not defined")
	}
	service.deviceManager.SetFavorite(req.DeviceId, req.ServiceId, false)
	return &clientsapi.RemoveFavoriteResponse{}, nil
}

func (service *UserService) GroupsStream(req *clientsapi.GroupsStreamRequest, server clientsapi.UserService_GroupsStreamServer) error {
	service.log.Infof("group stream started")
	defer service.log.Infof("group stream finished")

	lis := service.groupManager.GetListener()
	defer lis.Close()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-server.Context().Done():
			return status.Errorf(codes.Canceled, "context canceled")

		case <-ticker.C:
			// Send an empty message as a keepalive for the client.
			err := server.Send(&clientsapi.GroupsStreamResponse{})
			if err != nil {
				service.log.Errorf("failed to send group stream keepalive: %s", err)
				return status.Errorf(codes.Internal, "failed to send keepalive")
			}

		case update := <-lis.Sub():
			msg := &clientsapi.GroupsStreamResponse{}
			if update.Updated != nil {
				msg.GroupUpdate = update.Updated.Pb()
			}
			if update.Removed != nil {
				msg.RemovedId = *update.Removed
			}
			err := server.Send(msg)
			if err != nil {
				service.log.Errorf("failed to send group stream update: %s", err)
				return status.Errorf(codes.Internal, "failed to send update")
			}
		}
	}
}

func (service *UserService) AddGroup(ctx context.Context, req *clientsapi.AddGroupRequest) (*clientsapi.AddGroupResponse, error) {
	claims := ctx.Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return nil, status.Errorf(codes.PermissionDenied, "no claims in request")
	}
	if claims.Role != auth.AdminRole {
		return nil, status.Errorf(codes.PermissionDenied, "not allowed to add groups")
	}
	if req.GetName() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "name not defined")
	}
	if req.GetType() == clientsapi.Service_UNDEFINED {
		return nil, status.Errorf(codes.InvalidArgument, "type not defined")
	}

	// Determine the service ID based on the type.
	serviceID := services.DefaultServiceID(req.GetType())
	if serviceID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "type not supported")
	}

	// Generate a unique ID for the group.
	groupID, err := random.GenerateRandomString(10)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate group id: %s", err)
	}

	// Create the group and add it to the manager.
	members := make([]*core.GroupMember, len(req.GetMembers()))
	for i, member := range req.GetMembers() {
		members[i] = &core.GroupMember{
			DeviceID:  member.GetDeviceId(),
			ServiceID: member.GetServiceId(),
		}
	}
	group := core.NewGroup(groupID, serviceID, req.GetName(), req.GetType(), members)
	err = service.groupManager.AddGroup(group)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add group: %s", err)
	}

	return &clientsapi.AddGroupResponse{
		Group: group.Pb(),
	}, nil
}

func (service *UserService) UpdateGroup(ctx context.Context, req *clientsapi.UpdateGroupRequest) (*clientsapi.UpdateGroupResponse, error) {
	claims := ctx.Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return nil, status.Errorf(codes.PermissionDenied, "no claims in request")
	}
	if claims.Role != auth.AdminRole {
		return nil, status.Errorf(codes.PermissionDenied, "not allowed to update groups")
	}
	if req.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "id not defined")
	}
	if req.Name != nil {
		err := service.groupManager.UpdateGroupName(req.GetId(), req.GetName())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update group name: %s", err)
		}
	}
	if len(req.GetMembers()) > 0 {
		members := make([]*core.GroupMember, len(req.GetMembers()))
		for i, member := range req.GetMembers() {
			members[i] = &core.GroupMember{
				DeviceID:  member.GetDeviceId(),
				ServiceID: member.GetServiceId(),
			}
		}
		err := service.groupManager.UpdateGroupMembers(req.GetId(), members)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update group members: %s", err)
		}
	}
	return &clientsapi.UpdateGroupResponse{}, nil
}

func (service *UserService) RemoveGroup(ctx context.Context, req *clientsapi.RemoveGroupRequest) (*clientsapi.RemoveGroupResponse, error) {
	claims := ctx.Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return nil, status.Errorf(codes.PermissionDenied, "no claims in request")
	}
	if claims.Role != auth.AdminRole {
		return nil, status.Errorf(codes.PermissionDenied, "not allowed to remove groups")
	}
	if req.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "id not defined")
	}

	err := service.groupManager.RemoveGroup(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove group: %s", err)
	}

	return &clientsapi.RemoveGroupResponse{}, nil
}

func (service *UserService) SendAction(req *clientsapi.ActionRequest, server clientsapi.UserService_SendActionServer) error {
	if req.DeviceId == "" {
		return status.Error(codes.InvalidArgument, "device_id required")
	}
	if req.ServiceId == "" {
		return status.Error(codes.InvalidArgument, "service_id required")
	}

	actionID, clientID, err := service.deviceManager.PrepAction(req.GetDeviceId())
	if err != nil {
		return err
	}

	service.log.Infof("action %q for %q started: %s", actionID, clientID, req)
	defer service.log.Infof("action %q finished", actionID)

	req.ActionId = actionID

	sub := service.deviceManager.GetActionResponses()
	defer sub.Close()

	service.deviceManager.PushActionRequest(clientID, req)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	lastStatus := clientsapi.ActionResponse_UNDEFINED

	for {
		select {
		case <-ticker.C:
			// Time out requests if no status has been received.
			if lastStatus == clientsapi.ActionResponse_UNDEFINED {
				// Push the update out to the user.
				err := server.Send(&clientsapi.ActionResponse{
					ActionId: actionID,
					Status:   clientsapi.ActionResponse_CANCELED,
					Details:  "client didn't respond",
				})
				if err != nil {
					service.log.Warnf("action %q send err: %s", actionID, err)
					return status.Errorf(codes.Unknown, "failed to send")
				}
				return nil
			}

		case update := <-sub.Sub():
			if update.ClientID == clientID {
				service.log.Infof("action %q update: %s", actionID, update)
				if update.Response != nil {
					if update.Response.GetActionId() == actionID {
						// Update the last status.
						lastStatus = update.Response.Status

						// Push the update out to the user.
						err := server.Send(&clientsapi.ActionResponse{
							ActionId: actionID,
							Status:   update.Response.Status,
							Details:  update.Response.Details,
						})
						if err != nil {
							service.log.Warnf("action %q send err: %s", actionID, err)
							return status.Errorf(codes.Unknown, "failed to send")
						}

						// If status is final then return.
						if update.Response.Status >= clientsapi.ActionResponse_COMPLETE {
							return nil
						}
					}
				} else if update.Offline {
					// Push the update out to the user.
					err := server.Send(&clientsapi.ActionResponse{
						ActionId: actionID,
						Status:   clientsapi.ActionResponse_CANCELED,
						Details:  "client went offline",
					})
					if err != nil {
						service.log.Warnf("action %q send err: %s", actionID, err)
						return status.Errorf(codes.Unknown, "failed to send")
					}
					return nil
				}
			}
		}
	}
}

func (service *UserService) SendImageRequest(req *clientsapi.ImageRequest, server clientsapi.UserService_SendImageRequestServer) error {
	if req.DeviceId == "" {
		return status.Error(codes.InvalidArgument, "device_id required")
	}
	if req.ServiceId == "" {
		return status.Error(codes.InvalidArgument, "service_id required")
	}
	if req.AttributeId == "" {
		return status.Error(codes.InvalidArgument, "attribute_id required")
	}

	// We can use this for image requests too.
	requestID, clientID, err := service.deviceManager.PrepAction(req.GetDeviceId())
	if err != nil {
		return err
	}

	service.log.Infof("image request %q for %q started: %s", requestID, clientID, req)
	defer service.log.Infof("image request %q finished", requestID)

	req.RequestId = requestID

	sub := service.deviceManager.GetImageResponses()
	defer sub.Close()

	service.deviceManager.PushImageRequest(clientID, req)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	lastStatus := clientsapi.ImageResponse_UNDEFINED

	for {
		select {
		case <-ticker.C:
			// Time out requests if no status has been received.
			if lastStatus == clientsapi.ImageResponse_UNDEFINED {
				// Push the update out to the user.
				err := server.Send(&clientsapi.ImageResponse{
					RequestId: requestID,
					Status:    clientsapi.ImageResponse_CANCELED,
					Details:   "client didn't respond",
				})
				if err != nil {
					service.log.Warnf("image response %q send err: %s", requestID, err)
					return status.Errorf(codes.Unknown, "failed to send")
				}
				return nil
			}

		case update := <-sub.Sub():
			if update.ClientID == clientID {
				service.log.Infof("image request %q update: %s", requestID, apitools.ImageResponseString(update.Response))
				if update.Response != nil {
					if update.Response.GetRequestId() == requestID {
						// Update the last status.
						lastStatus = update.Response.Status

						// Push the update out to the user.
						err := server.Send(&clientsapi.ImageResponse{
							RequestId: requestID,
							Status:    update.Response.Status,
							Details:   update.Response.Details,
							Data:      update.Response.Data,
						})
						if err != nil {
							service.log.Warnf("image response %q send err: %s", requestID, err)
							return status.Errorf(codes.Unknown, "failed to send")
						}

						// If status is final then return.
						if update.Response.Status >= clientsapi.ImageResponse_COMPLETE {
							return nil
						}
					}
				} else if update.Offline {
					// Push the update out to the user.
					err := server.Send(&clientsapi.ImageResponse{
						RequestId: requestID,
						Status:    clientsapi.ImageResponse_CANCELED,
						Details:   "client went offline",
					})
					if err != nil {
						service.log.Warnf("image response %q send err: %s", requestID, err)
						return status.Errorf(codes.Unknown, "failed to send")
					}
					return nil
				}
			}
		}
	}
}

func (service *UserService) UsersStream(req *clientsapi.UsersStreamRequest, server clientsapi.UserService_UsersStreamServer) error {
	claims := server.Context().Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return status.Errorf(codes.PermissionDenied, "no claims in request")
	}

	service.log.Infof("users stream started")
	defer service.log.Infof("users stream finished")

	lis := service.userManager.GetListener()
	defer lis.Close()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-server.Context().Done():
			return status.Errorf(codes.Canceled, "context canceled")

		case <-ticker.C:
			// Send an empty message as a keepalive for the client.
			err := server.Send(&clientsapi.UsersStreamResponse{})
			if err != nil {
				service.log.Errorf("failed to send users stream keepalive: %s", err)
				return status.Errorf(codes.Internal, "failed to send keepalive")
			}

		case update := <-lis.Sub():
			// Admins get everyone, users only get themselves.
			sendIt := false
			if claims.Role == auth.AdminRole {
				sendIt = true
			} else if update.Updated != nil && update.Updated.Username == claims.Username {
				sendIt = true
			}

			if sendIt {
				msg := &clientsapi.UsersStreamResponse{}
				if update.Updated != nil {
					msg.User = update.Updated.Pb()
				}
				if update.Removed != nil {
					msg.UserRemoved = *update.Removed
				}
				err := server.Send(msg)
				if err != nil {
					service.log.Errorf("failed to send users stream update: %s", err)
					return status.Errorf(codes.Internal, "failed to send update")
				}
			}
		}
	}
}

func (service *UserService) AddUser(ctx context.Context, req *clientsapi.AddUserRequest) (*clientsapi.AddUserResponse, error) {
	claims := ctx.Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return nil, status.Errorf(codes.PermissionDenied, "no claims in request")
	}

	// Only admins can add users.
	if claims.Role != auth.AdminRole && claims.Username != req.Username {
		return nil, status.Errorf(codes.PermissionDenied, "not allowed to add users")
	}

	// Check that the request is valid.
	if req.Username == "" {
		return nil, status.Errorf(codes.InvalidArgument, "username not defined")
	}
	if req.Fullname == "" {
		return nil, status.Errorf(codes.InvalidArgument, "fullname not defined")
	}
	if req.Role != clientsapi.UserRole_USER_ROLE_ADMIN && req.Role != clientsapi.UserRole_USER_ROLE_USER {
		return nil, status.Errorf(codes.InvalidArgument, "role not defined")
	}
	if req.InitialPassword == "" {
		return nil, status.Errorf(codes.InvalidArgument, "initial password not defined")
	}

	user, err := core.NewUser(req.Username, req.Fullname, req.InitialPassword, auth.RoleFromPb(req.Role))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to create user: %v", err)
	}

	// Force new users to reset their password on first login.
	user.ResetPassword = true

	// Add the user to the store.
	err = service.userManager.Store(user)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to add user: %v", err)
	}

	return &clientsapi.AddUserResponse{}, nil
}

func (service *UserService) UpdateUser(ctx context.Context, req *clientsapi.UpdateUserRequest) (*clientsapi.UpdateUserResponse, error) {
	claims := ctx.Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return nil, status.Errorf(codes.PermissionDenied, "no claims in request")
	}

	// Only admins can change other users.
	if claims.Role != auth.AdminRole && claims.Username != req.Username {
		return nil, status.Errorf(codes.PermissionDenied, "not allowed to modify other user")
	}

	if req.Username == "" {
		return nil, status.Errorf(codes.InvalidArgument, "username not defined")
	}

	if req.Fullname != nil {
		err := service.userManager.SetFullname(req.Username, req.GetFullname())
		if err != nil {
			return nil, status.Errorf(codes.PermissionDenied, "%s", err)
		}
	}
	if req.Role != nil {
		// Only admins are allowed to do this.
		if claims.Role != auth.AdminRole {
			return nil, status.Errorf(codes.PermissionDenied, "only admins can change user roles")
		}

		// Admins are not allowed to change their own role.
		if claims.Username == req.Username {
			return nil, status.Errorf(codes.PermissionDenied, "not allowed to change own role")
		}

		err := service.userManager.SetRole(req.Username, auth.RoleFromPb(req.GetRole()))
		if err != nil {
			return nil, status.Errorf(codes.PermissionDenied, "%s", err)
		}
	}
	// if req.Password != nil {
	// 	err := service.userManager.SetPassword(req.GetPassword())
	// 	if err != nil {
	// 		return nil, status.Errorf(codes.PermissionDenied, "%s", err)
	// 	}
	// }
	return &clientsapi.UpdateUserResponse{}, nil
}

func (service *UserService) RemoveUser(ctx context.Context, req *clientsapi.RemoveUserRequest) (*clientsapi.RemoveUserResponse, error) {
	claims := ctx.Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return nil, status.Errorf(codes.PermissionDenied, "no claims in request")
	}

	// Only admins can remove other users.
	if claims.Role != auth.AdminRole && claims.Username != req.Username {
		return nil, status.Errorf(codes.PermissionDenied, "not allowed to remove other user")
	}

	if req.Username == "" {
		return nil, status.Errorf(codes.InvalidArgument, "username not defined")
	}

	// Admins are not allowed to remove themselves.
	if claims.Username == req.Username {
		return nil, status.Errorf(codes.PermissionDenied, "not allowed to remove self")
	}

	err := service.userManager.Delete(req.Username)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "%s", err)
	}

	return &clientsapi.RemoveUserResponse{}, nil
}
