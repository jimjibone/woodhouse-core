package users

import (
	"context"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/apitools"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-4/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserService struct {
	clientsapi.UnimplementedUserServiceServer
	log              *log.Context
	deviceManager    *core.DeviceManager
	favoritesManager *core.FavoritesManager
	userManager      *core.UserManager
}

func NewUserService(deviceManager *core.DeviceManager, favoritesManager *core.FavoritesManager, userManager *core.UserManager) *UserService {
	service := &UserService{
		log:              log.NewContext(log.DefaultLogger, "user-service", log.DebugLevel),
		deviceManager:    deviceManager,
		favoritesManager: favoritesManager,
		userManager:      userManager,
	}
	return service
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
			err := server.Send(dev)
			if err != nil {
				service.log.Errorf("failed to send device stream: %s", err)
				return status.Errorf(codes.Internal, "failed to send device")
			}
		}
	}

	// Send an empty device to indicate the end of the initial list.
	err := server.Send(&clientsapi.Device{})
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
			err := server.Send(&clientsapi.Device{})
			if err != nil {
				service.log.Errorf("failed to send device stream keepalive: %s", err)
				return status.Errorf(codes.Internal, "failed to send device keepalive")
			}

		case update := <-sub.Sub():
			if isInFilter(update.GetId(), req.IncludeDeviceIds) {
				err := server.Send(update)
				if err != nil {
					service.log.Errorf("failed to send device stream update: %s", err)
					return status.Errorf(codes.Internal, "failed to send device update")
				}
			}
		}
	}
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
