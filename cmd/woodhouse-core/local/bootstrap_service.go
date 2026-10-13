// Package local holds services for other processes on the same device,
// served on a unix socket. Access control is the socket's file permissions;
// these services must never be registered on a network listener.
package local

import (
	"context"
	"net"
	"net/netip"
	"time"

	"github.com/jimjibone/log"
	localapi "github.com/jimjibone/woodhouse-api/go/v1/local"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/limits"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// localAddr is the rate-limit key for local callers. They all share one
// budget, which is fine: the only caller is the device agent, relaying
// passwords typed on a phone during recovery.
var localAddr = netip.MustParseAddr("127.0.0.1")

// Settings is the part of core.SettingsManager the service needs.
type Settings interface {
	InstanceName() string
	SetInstanceName(name string) (string, error)
}

// BootstrapService lets the device agent create the first admin during
// adoption, and check admin credentials during recovery.
type BootstrapService struct {
	localapi.UnimplementedBootstrapServiceServer
	log      *log.Context
	users    *core.UserManager
	settings Settings
	limits   *limits.Login
	apiAddr  net.Addr
	webAddr  net.Addr
	apiCert  []byte
}

func NewBootstrapService(users *core.UserManager, settings Settings, lim *limits.Login, apiAddr, webAddr net.Addr, apiCertDER []byte) *BootstrapService {
	return &BootstrapService{
		log:      log.NewContext(log.DefaultLogger, "bootstrap", log.DebugLevel),
		users:    users,
		settings: settings,
		limits:   lim,
		apiAddr:  apiAddr,
		webAddr:  webAddr,
		apiCert:  apiCertDER,
	}
}

func (srv *BootstrapService) GetStatus(ctx context.Context, req *localapi.GetStatusRequest) (*localapi.GetStatusResponse, error) {
	return &localapi.GetStatusResponse{
		HasAdmin:       srv.users.HasAnAdmin(),
		InstanceName:   srv.settings.InstanceName(),
		ApiPort:        port(srv.apiAddr),
		WebPort:        port(srv.webAddr),
		ApiCertificate: srv.apiCert,
	}, nil
}

func (srv *BootstrapService) CreateAdmin(ctx context.Context, req *localapi.CreateAdminRequest) (*localapi.CreateAdminResponse, error) {
	if req.Username == "" || req.Password == "" {
		return nil, status.Errorf(codes.InvalidArgument, "username and password are required")
	}
	if srv.users.HasAnAdmin() {
		return nil, status.Errorf(codes.FailedPrecondition, "an admin already exists")
	}

	admin, err := core.NewUser(req.Username, "", req.Password, auth.AdminRole)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to create user: %v", err)
	}
	admin.ResetPassword = false // chosen by the user during adoption
	if err := srv.users.Store(admin); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to add user: %v", err)
	}
	srv.log.Infof("created first admin %q", req.Username)

	if req.InstanceName != "" {
		if _, err := srv.settings.SetInstanceName(req.InstanceName); err != nil {
			// The admin exists now; a bad name should not undo adoption.
			srv.log.Warnf("failed to set instance name %q: %v", req.InstanceName, err)
		}
	}

	return &localapi.CreateAdminResponse{}, nil
}

func (srv *BootstrapService) VerifyAdmin(ctx context.Context, req *localapi.VerifyAdminRequest) (*localapi.VerifyAdminResponse, error) {
	if ok, _ := srv.limits.AllowIP(localAddr); !ok {
		return nil, status.Errorf(codes.ResourceExhausted, "too many attempts, try again later")
	}
	if wait := srv.limits.AccountRetryIn(req.Username); wait > 0 {
		return nil, status.Errorf(codes.ResourceExhausted, "too many failed attempts, try again in %s", wait.Round(time.Second))
	}

	user := srv.users.Find(req.Username)
	if user == nil {
		core.DummyPasswordCheck(req.Password)
		srv.limits.RecordFailure(localAddr, req.Username)
		return nil, status.Errorf(codes.Unauthenticated, "incorrect username/password")
	}
	if !user.IsCorrectPassword(req.Password) || user.Role != auth.AdminRole {
		srv.limits.RecordFailure(localAddr, req.Username)
		return nil, status.Errorf(codes.Unauthenticated, "incorrect username/password")
	}
	srv.limits.RecordSuccess(req.Username)

	return &localapi.VerifyAdminResponse{}, nil
}

func port(addr net.Addr) uint32 {
	if tcp, ok := addr.(*net.TCPAddr); ok {
		return uint32(tcp.Port)
	}
	return 0
}
