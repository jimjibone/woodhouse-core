package clients

import (
	"context"
	"slices"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/cert"
	"github.com/jimjibone/woodhouse-4/shared/crypt"
	"github.com/jimjibone/woodhouse-4/shared/random"
	"github.com/schollz/pake/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthService struct {
	clientsapi.UnimplementedAuthServiceServer
	log *log.Context
	cm  *cert.CertManager
	jwt *JWTManager
}

func NewAuthService(cm *cert.CertManager, ba *JWTManager) *AuthService {
	return &AuthService{
		log: log.NewContext(log.DefaultLogger, "clients-auth", log.DebugLevel),
		cm:  cm,
		jwt: ba,
	}
}

func (as *AuthService) Pair(server clientsapi.AuthService_PairServer) error {
	// 1. Get the client ID from the client.
	req, err := server.Recv()
	if err != nil {
		as.log.Warnf("pairing client failed to receive client id: %s", err)
		return status.Errorf(codes.Unknown, "failed to receive client id")
	}
	if req.ClientId == "" {
		return status.Errorf(codes.InvalidArgument, "client_id must be set")
	}

	clientID := req.ClientId
	as.log.Infof("pairing client %q started", clientID)

	// 2. Send the pairing state to the client until the user has accepted the
	// pairing request.
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	start := time.Now()
	pending := true
	for pending {
		select {
		case <-ticker.C:
			as.log.Debugf("pairing client %q pending...", clientID)
			err = server.Send(&clientsapi.PairResponse{
				State: clientsapi.PairResponse_Pending,
			})
			if err != nil {
				code := status.Code(err)
				switch code {
				case codes.Unavailable:
					as.log.Infof("pairing client %q went offline", clientID)
					return status.Errorf(code, "client offline")

				default:
					as.log.Warnf("pairing client %q error when sending pending: %s", clientID, err)
					return status.Errorf(code, "failed to send pending")
				}
			}

			if time.Since(start) >= 1*time.Second {
				pending = false
			}
		}
	}

	// Start the PAKE handshake using the key provided by the user.
	as.log.Debugf("pairing client %q initialising pake", clientID)
	pakep, err := pake.InitCurve([]byte("redacted"), 0, "p521")
	if err != nil {
		as.log.Warnf("pairing client %q failed to init pake: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to init pake: %s", err)
	}

	// 3. Send the first PAKE handshake blob to the client.
	as.log.Debugf("pairing client %q sending first handshake blob", clientID)
	err = server.Send(&clientsapi.PairResponse{
		State: clientsapi.PairResponse_Handshake,
		Data:  pakep.Bytes(),
	})
	if err != nil {
		as.log.Warnf("pairing client %q error when sending handshake start: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to send handshake")
	}

	// 4. Receive the second handshake blob from the client.
	as.log.Debugf("pairing client %q waiting for second handshake blob", clientID)
	req, err = server.Recv()
	if err != nil {
		as.log.Warnf("pairing client %q failed to receive handshake: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to receive handshake")
	}
	as.log.Debugf("pairing client %q received second handshake blob", clientID)
	err = pakep.Update(req.Data)
	if err != nil {
		as.log.Warnf("pairing client %q failed to update handshake: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to update handshake")
	}

	// 5. We should now have the session key. Let's confirm this by sending a
	// test (an encrypted blob of random bytes).
	key, err := pakep.SessionKey()
	if err != nil {
		as.log.Warnf("pairing client %q failed to get session key: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to prepare test")
	}
	as.log.Debugf("pairing client %q generated key: [%d] %x", clientID, len(key), key)
	test, err := random.GenerateRandomString(128)
	if err != nil {
		as.log.Warnf("pairing client %q failed to generate test: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to generate test")
	}
	encrypted, err := crypt.Encrypt([]byte(test), key)
	if err != nil {
		as.log.Warnf("pairing client %q failed to encrypt test: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to encrypt test")
	}
	as.log.Debugf("pairing client %q sending test", clientID)
	err = server.Send(&clientsapi.PairResponse{
		Data: encrypted,
	})
	if err != nil {
		as.log.Warnf("pairing client %q failed to send test: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to send test")
	}

	// 6. Receive the test back.
	as.log.Debugf("pairing client %q waiting for test reply", clientID)
	req, err = server.Recv()
	if err != nil {
		as.log.Warnf("pairing client %q failed to receive test: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to receive test")
	}
	decrypted, err := crypt.Decrypt(req.Data, key)
	if err != nil {
		as.log.Warnf("pairing client %q failed to decrypt test: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to decrypt test")
	}
	slices.Reverse(decrypted)
	decryptedTest := string(decrypted)
	as.log.Debugf("pairing client %q test reply was valid", clientID)
	if test != decryptedTest {
		as.log.Warnf("pairing client %q received invalid test response", clientID)
		return status.Errorf(codes.PermissionDenied, "incorrect test response")
	}

	// 7. Send the server's certificate.
	encrypted, err = crypt.Encrypt(as.cm.CertPEM(), key)
	if err != nil {
		as.log.Warnf("pairing client %q failed to encrypt cert: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to encrypt cert")
	}
	as.log.Debugf("pairing client %q sending cert", clientID)
	err = server.Send(&clientsapi.PairResponse{
		Data: encrypted,
	})
	if err != nil {
		as.log.Warnf("pairing client %q error when sending cert: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to send cert")
	}

	// 8. Generate refresh auth token for the client and send it.
	tokens, err := as.jwt.GenerateTokens(clientID)
	if err != nil {
		as.log.Warnf("pairing client %q failed to generate token: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to generate token")
	}
	encrypted, err = crypt.Encrypt([]byte(tokens.RefreshToken), key)
	if err != nil {
		as.log.Warnf("pairing client %q failed to encrypt token: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to encrypt token")
	}
	as.log.Debugf("pairing client %q sending token", clientID)
	err = server.Send(&clientsapi.PairResponse{
		Data: encrypted,
	})
	if err != nil {
		as.log.Warnf("pairing client %q error when sending token: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to send token")
	}

	as.log.Infof("pairing client %q finished", clientID)
	return nil
}

func (as *AuthService) Refresh(ctx context.Context, req *clientsapi.RefreshRequest) (*clientsapi.RefreshResponse, error) {
	claims, err := as.jwt.VerifyRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "%s", err)
	}

	// How many days has the refresh token got left before expiry?
	exp := claims.ExpiresAt.Time
	remainingDays := time.Until(exp).Hours() / 24.0
	renewDays := (refreshTokenDuration / 2).Hours() / 24.0

	// If requested, don't revoke or replace the refresh token.
	refreshToken := ""
	accessToken := ""
	if remainingDays < renewDays {
		// Generate both tokens.
		tokens, err := as.jwt.GenerateTokens(claims.ClientID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate tokens")
		}
		refreshToken = tokens.RefreshToken
		accessToken = tokens.AccessToken
	} else {
		// Generate only the access token.
		token, err := as.jwt.GenerateAccessToken(claims.ClientID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate access token")
		}
		refreshToken = req.RefreshToken
		accessToken = token
	}

	res := &clientsapi.RefreshResponse{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}

	return res, nil
}

func (as *AuthService) Logout(ctx context.Context, req *clientsapi.LogoutRequest) (*clientsapi.LogoutResponse, error) {
	claims, err := as.jwt.VerifyRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "refresh token is invalid")
	}

	as.jwt.RevokeToken(claims.RefreshUUID)

	return &clientsapi.LogoutResponse{}, nil
}

func (as *AuthService) Ping(ctx context.Context, req *clientsapi.PingRequest) (*clientsapi.PingResponse, error) {
	return &clientsapi.PingResponse{}, nil
}
