package clients

import (
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
		log:           log.NewContext(log.DefaultLogger, "clients-service", log.DebugLevel),
		deviceManager: deviceManager,
	}
	return service
}

func (service *ClientService) StatusStream(server clientsapi.ClientService_StatusStreamServer) error {
	claims, ok := server.Context().Value("claims").(*AccessTokenClaims)
	if !ok {
		return status.Errorf(codes.Internal, "invalid claims")
	}

	service.log.Infof("%q status stream started", claims.ClientID)
	defer service.log.Infof("%q status stream finished", claims.ClientID)

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
	return status.Errorf(codes.Unimplemented, "method ActionStream not implemented")
}
