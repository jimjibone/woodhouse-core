package users

import (
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-4/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserService struct {
	clientsapi.UnimplementedUserServiceServer
	log           *log.Context
	deviceManager *core.DeviceManager
}

func NewUserService(deviceManager *core.DeviceManager) *UserService {
	service := &UserService{
		log:           log.NewContext(log.DefaultLogger, "user-service", log.DebugLevel),
		deviceManager: deviceManager,
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
	service.log.Infof("device stream started")
	defer service.log.Infof("device stream finished")

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

	service.log.Infof("action %q started: %s", actionID, req)
	defer service.log.Infof("action %q finished", actionID)

	req.ActionId = actionID

	sub := service.deviceManager.GetActionResponses()
	defer sub.Close()

	service.deviceManager.PushActionRequest(clientID, req)

	for {
		select {
		case update := <-sub.Sub():
			service.log.Infof("action %q update: %s", actionID, update)
			if update.ClientID == clientID {
				if update.Response != nil {
					if update.Response.GetActionId() == actionID {
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
