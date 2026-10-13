// Package system serves clients.SystemService by forwarding each call to
// the woodhouse-os device agent, which manages the OS. Authorization (admin
// only) happens in the gRPC interceptor before calls get here.
package system

import (
	"context"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	localapi "github.com/jimjibone/woodhouse-api/go/v1/local"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	clientsapi.UnimplementedSystemServiceServer
	agent localapi.AgentServiceClient
}

// New returns the service. A nil agent (core not running on woodhouse-os)
// makes every call fail with UNAVAILABLE.
func New(agent localapi.AgentServiceClient) *Service {
	return &Service{agent: agent}
}

var errNoAgent = status.Error(codes.Unavailable, "system management is only available on woodhouse-os")

func (s *Service) GetSystemInfo(ctx context.Context, req *clientsapi.GetSystemInfoRequest) (*clientsapi.GetSystemInfoResponse, error) {
	if s.agent == nil {
		return nil, errNoAgent
	}
	return s.agent.GetSystemInfo(ctx, req)
}

func (s *Service) GetUpdateReports(ctx context.Context, req *clientsapi.GetUpdateReportsRequest) (*clientsapi.GetUpdateReportsResponse, error) {
	if s.agent == nil {
		return nil, errNoAgent
	}
	return s.agent.GetUpdateReports(ctx, req)
}

func (s *Service) GetWifiStatus(ctx context.Context, req *clientsapi.GetWifiStatusRequest) (*clientsapi.GetWifiStatusResponse, error) {
	if s.agent == nil {
		return nil, errNoAgent
	}
	return s.agent.GetWifiStatus(ctx, req)
}

func (s *Service) ScanWifi(ctx context.Context, req *clientsapi.SystemScanWifiRequest) (*clientsapi.SystemScanWifiResponse, error) {
	if s.agent == nil {
		return nil, errNoAgent
	}
	return s.agent.ScanWifi(ctx, req)
}

func (s *Service) JoinWifi(ctx context.Context, req *clientsapi.SystemJoinWifiRequest) (*clientsapi.SystemJoinWifiResponse, error) {
	if s.agent == nil {
		return nil, errNoAgent
	}
	return s.agent.JoinWifi(ctx, req)
}

func (s *Service) SetSSH(ctx context.Context, req *clientsapi.SetSSHRequest) (*clientsapi.SetSSHResponse, error) {
	if s.agent == nil {
		return nil, errNoAgent
	}
	return s.agent.SetSSH(ctx, req)
}

func (s *Service) Reboot(ctx context.Context, req *clientsapi.RebootRequest) (*clientsapi.RebootResponse, error) {
	if s.agent == nil {
		return nil, errNoAgent
	}
	return s.agent.Reboot(ctx, req)
}

func (s *Service) FactoryReset(ctx context.Context, req *clientsapi.FactoryResetRequest) (*clientsapi.FactoryResetResponse, error) {
	if s.agent == nil {
		return nil, errNoAgent
	}
	return s.agent.FactoryReset(ctx, req)
}
