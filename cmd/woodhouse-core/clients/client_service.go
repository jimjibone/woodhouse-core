package clients

import (
	"context"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
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
