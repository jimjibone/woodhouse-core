package clients

import (
	"context"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/apitools"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-4/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ClientService struct {
	clientsapi.UnimplementedClientServiceServer
	log           *log.Context
	deviceManager *core.DeviceManager
}

func NewClientService(deviceManager *core.DeviceManager) *ClientService {
	service := &ClientService{
		log:           log.NewContext(log.DefaultLogger, "client-service", log.DebugLevel),
		deviceManager: deviceManager,
	}
	return service
}

func (service *ClientService) StatusStream(server clientsapi.ClientService_StatusStreamServer) error {
	claims, ok := server.Context().Value("claims").(*AccessTokenClaims)
	if !ok {
		return status.Errorf(codes.Internal, "invalid claims")
	}

	service.log.Debugf("%q status stream started", claims.ClientID)
	defer service.log.Debugf("%q status stream finished", claims.ClientID)

	defer service.deviceManager.SetClientOffline(claims.ClientID)

	for {
		update, err := server.Recv()
		if err != nil {
			code := status.Code(err)
			if code != codes.Canceled {
				service.log.Errorf("%q status stream error: %s", claims.ClientID, err)
			}
			return status.Errorf(codes.InvalidArgument, "recv failed")
		}

		// Validate updates.
		for _, dev := range update.DeviceInfo {
			// The device ID must be set.
			if dev.Id == "" {
				return status.Errorf(codes.InvalidArgument, "device has empty id")
			}

			// A full device state must contain the Info and Online services.
			if dev.FullState {
				foundInfo := false
				foundOnline := false
				for _, srv := range dev.Services {
					switch srv.Typ {
					case clientsapi.Service_INFO:
						foundInfo = true
					case clientsapi.Service_ONLINE:
						foundOnline = true
					}
				}
				if !foundInfo || !foundOnline {
					service.log.Errorf("client %q device %q does not have info or online services", claims.ClientID, dev.Id)
					return status.Errorf(codes.InvalidArgument, "device %q does not have info or online services", dev.Id)
				}
			}
		}

		for _, dev := range update.DeviceInfo {
			service.deviceManager.PushDeviceUpdate(claims.ClientID, dev)
		}
	}
}

func (service *ClientService) ActionStream(server clientsapi.ClientService_ActionStreamServer) error {
	claims, ok := server.Context().Value("claims").(*AccessTokenClaims)
	if !ok {
		return status.Errorf(codes.Internal, "invalid claims")
	}

	service.log.Debugf("%q action stream started", claims.ClientID)
	defer service.log.Debugf("%q action stream finished", claims.ClientID)

	defer service.deviceManager.PushActionResponse(claims.ClientID, nil, true)

	requests := service.deviceManager.GetActionRequests()
	defer requests.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		defer cancel()
		for {
			res, err := server.Recv()
			if err != nil {
				code := status.Code(err)
				if code != codes.Canceled {
					service.log.Errorf("%q action stream error: %s", claims.ClientID, err)
				}
				return
			}

			service.deviceManager.PushActionResponse(claims.ClientID, res, false)

			select {
			case <-ctx.Done():
				service.log.Debugf("%q action stream recv done", claims.ClientID)
				return
			default:
			}
		}
	}()

	for {
		select {
		case req := <-requests.Sub():
			if req.ClientID == claims.ClientID {
				err := server.Send(req.Request)
				if err != nil {
					code := status.Code(err)
					if code != codes.Canceled {
						service.log.Errorf("%q action %q stream error: %s", claims.ClientID, req.Request.GetActionId(), err)
					}
					service.log.Warnf("%q action stream send err: %s", claims.ClientID, err)
					return status.Errorf(codes.InvalidArgument, "send failed")
				}
			}

		case <-ctx.Done():
			service.log.Debugf("%q action stream send done", claims.ClientID)
			return nil
		}
	}
}

func (service *ClientService) ImageStream(server clientsapi.ClientService_ImageStreamServer) error {
	claims, ok := server.Context().Value("claims").(*AccessTokenClaims)
	if !ok {
		return status.Errorf(codes.Internal, "invalid claims")
	}

	service.log.Debugf("%q images stream started", claims.ClientID)
	defer service.log.Debugf("%q images stream finished", claims.ClientID)

	defer service.deviceManager.PushImageResponse(claims.ClientID, nil, true)

	requests := service.deviceManager.GetImageRequests()
	defer requests.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		defer cancel()
		for {
			res, err := server.Recv()
			if err != nil {
				code := status.Code(err)
				if code != codes.Canceled {
					service.log.Errorf("%q images stream error: %s", claims.ClientID, err)
				}
				return
			}

			service.deviceManager.PushImageResponse(claims.ClientID, res, false)

			select {
			case <-ctx.Done():
				service.log.Debugf("%q images stream recv done", claims.ClientID)
				return
			default:
			}
		}
	}()

	for {
		select {
		case req := <-requests.Sub():
			if req.ClientID == claims.ClientID {
				err := server.Send(req.Request)
				if err != nil {
					code := status.Code(err)
					if code != codes.Canceled {
						service.log.Errorf("%q image request %q stream error: %s", claims.ClientID, req.Request.GetRequestId(), err)
					}
					service.log.Warnf("%q images stream send err: %s", claims.ClientID, err)
					return status.Errorf(codes.InvalidArgument, "send failed")
				}
			}

		case <-ctx.Done():
			service.log.Debugf("%q images stream send done", claims.ClientID)
			return nil
		}
	}
}

func (service *ClientService) DeviceStream(server clientsapi.ClientService_DeviceStreamServer) error {
	service.log.Infof("device stream started")
	defer service.log.Infof("device stream finished")

	sub := service.deviceManager.GetDeviceUpdates()
	defer sub.Close()

	// isInFilter := func(deviceID string, filter []string) bool {
	// 	if len(filter) == 0 {
	// 		return true
	// 	}
	// 	for _, id := range filter {
	// 		if deviceID == id {
	// 			return true
	// 		}
	// 	}
	// 	return false
	// }

	// Start by sending the full list of devices.
	for dev := range service.deviceManager.GetDevices() {
		// if isInFilter(dev.GetId(), req.IncludeDeviceIds) {
		err := server.Send(dev)
		if err != nil {
			service.log.Errorf("failed to send device stream: %s", err)
			return status.Errorf(codes.Internal, "failed to send device")
		}
		// }
	}

	// Read requests for full device status.
	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			req, err := server.Recv()
			if err != nil {
				service.log.Errorf("failed to receive device stream request: %s", err)
				return
			}
			if req.GetDeviceId != "" {
				// Send the requested device full state.
				dev := service.deviceManager.GetDevice(req.GetDeviceId)
				if dev != nil {
					err := server.Send(dev)
					if err != nil {
						service.log.Errorf("failed to send device stream: %s", err)
						return
					}
				}
			}
		}
	}()

	// Send an empty Device message to indicate the end of current device states.
	err := server.Send(&clientsapi.Device{})
	if err != nil {
		service.log.Errorf("failed to send device stream keepalive: %s", err)
		return status.Errorf(codes.Internal, "failed to send device keepalive")
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
			// if isInFilter(update.GetId(), req.IncludeDeviceIds) {
			err := server.Send(update)
			if err != nil {
				service.log.Errorf("failed to send device stream update: %s", err)
				return status.Errorf(codes.Internal, "failed to send device update")
			}
			// }
		}
	}
}

func (service *ClientService) SendAction(req *clientsapi.ActionRequest, server clientsapi.ClientService_SendActionServer) error {
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

func (service *ClientService) SendImageRequest(req *clientsapi.ImageRequest, server clientsapi.ClientService_SendImageRequestServer) error {
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
