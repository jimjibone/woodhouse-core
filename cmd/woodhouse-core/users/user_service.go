package users

import (
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/apitools"
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
