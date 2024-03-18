package main

import (
	"context"
	"slices"
	"time"

	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/bridges"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/cert"
	"github.com/jimjibone/woodhouse-4/shared/crypt"
	"github.com/jimjibone/woodhouse-4/shared/random"
	"github.com/schollz/pake/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type BridgeAuthService struct {
	api.BridgeAuthServiceServer
	cm *cert.CertManager
	ba *bridges.JWTManager
}

func NewBridgeAuthService(cm *cert.CertManager, ba *bridges.JWTManager) *BridgeAuthService {
	return &BridgeAuthService{
		cm: cm,
		ba: ba,
	}
}

func (bs *BridgeAuthService) Pair(server api.BridgeAuthService_PairServer) error {
	// 1. Get the client ID from the client.
	req, err := server.Recv()
	if err != nil {
		log.Warnf("pairing client failed to receive client id: %s", err)
		return status.Errorf(codes.Unknown, "failed to receive client id")
	}
	if req.ClientId == "" {
		return status.Errorf(codes.InvalidArgument, "client_id must be set")
	}

	clientID := req.ClientId
	log.Infof("pairing client %q started", clientID)

	// 2. Send the pairing state to the client until the user has accepted the
	// pairing request.
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	start := time.Now()
	pending := true
	for pending {
		select {
		case <-ticker.C:
			log.Debugf("pairing client %q pending...", clientID)
			err = server.Send(&api.BridgePairResponse{
				State: api.BridgePairResponse_Pending,
			})
			if err != nil {
				code := status.Code(err)
				switch code {
				case codes.Unavailable:
					log.Infof("pairing client %q went offline", clientID)
					return status.Errorf(code, "client offline")

				default:
					log.Warnf("pairing client %q error when sending pending: %s", clientID, err)
					return status.Errorf(code, "failed to send pending")
				}
			}

			if time.Since(start) >= 1*time.Second {
				pending = false
			}
		}
	}

	// Start the PAKE handshake using the key provided by the user.
	log.Debugf("pairing client %q initialising pake", clientID)
	pakep, err := pake.InitCurve([]byte("redacted"), 0, "p521")
	if err != nil {
		log.Warnf("pairing client %q failed to init pake: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to init pake: %s", err)
	}

	// 3. Send the first PAKE handshake blob to the client.
	log.Debugf("pairing client %q sending first handshake blob", clientID)
	err = server.Send(&api.BridgePairResponse{
		State: api.BridgePairResponse_Handshake,
		Data:  pakep.Bytes(),
	})
	if err != nil {
		log.Warnf("pairing client %q error when sending handshake start: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to send handshake")
	}

	// 4. Receive the second handshake blob from the client.
	log.Debugf("pairing client %q waiting for second handshake blob", clientID)
	req, err = server.Recv()
	if err != nil {
		log.Warnf("pairing client %q failed to receive handshake: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to receive handshake")
	}
	log.Debugf("pairing client %q received second handshake blob", clientID)
	err = pakep.Update(req.Data)
	if err != nil {
		log.Warnf("pairing client %q failed to update handshake: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to update handshake")
	}

	// 5. We should now have the session key. Let's confirm this by sending a
	// test (an encrypted blob of random bytes).
	key, err := pakep.SessionKey()
	if err != nil {
		log.Warnf("pairing client %q failed to get session key: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to prepare test")
	}
	log.Debugf("pairing client %q generated key: [%d] %x", clientID, len(key), key)
	test, err := random.GenerateRandomString(128)
	if err != nil {
		log.Warnf("pairing client %q failed to generate test: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to generate test")
	}
	encrypted, err := crypt.Encrypt([]byte(test), key)
	if err != nil {
		log.Warnf("pairing client %q failed to encrypt test: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to encrypt test")
	}
	log.Debugf("pairing client %q sending test", clientID)
	err = server.Send(&api.BridgePairResponse{
		Data: encrypted,
	})
	if err != nil {
		log.Warnf("pairing client %q failed to send test: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to send test")
	}

	// 6. Receive the test back.
	log.Debugf("pairing client %q waiting for test reply", clientID)
	req, err = server.Recv()
	if err != nil {
		log.Warnf("pairing client %q failed to receive test: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to receive test")
	}
	decrypted, err := crypt.Decrypt(req.Data, key)
	if err != nil {
		log.Warnf("pairing client %q failed to decrypt test: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to decrypt test")
	}
	slices.Reverse(decrypted)
	decryptedTest := string(decrypted)
	log.Debugf("pairing client %q test reply was valid", clientID)
	if test != decryptedTest {
		log.Warnf("pairing client %q received invalid test response", clientID)
		return status.Errorf(codes.PermissionDenied, "incorrect test response")
	}

	// 7. Send the server's certificate.
	encrypted, err = crypt.Encrypt(bs.cm.CertPEM(), key)
	if err != nil {
		log.Warnf("pairing client %q failed to encrypt cert: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to encrypt cert")
	}
	log.Debugf("pairing client %q sending cert", clientID)
	err = server.Send(&api.BridgePairResponse{
		Data: encrypted,
	})
	if err != nil {
		log.Warnf("pairing client %q error when sending cert: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to send cert")
	}

	// 8. Generate refresh auth token for the client and send it.
	tokens, err := bs.ba.GenerateTokens(clientID)
	if err != nil {
		log.Warnf("pairing client %q failed to generate token: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to generate token")
	}
	encrypted, err = crypt.Encrypt([]byte(tokens.RefreshToken), key)
	if err != nil {
		log.Warnf("pairing client %q failed to encrypt token: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to encrypt token")
	}
	log.Debugf("pairing client %q sending token", clientID)
	err = server.Send(&api.BridgePairResponse{
		Data: encrypted,
	})
	if err != nil {
		log.Warnf("pairing client %q error when sending token: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to send token")
	}

	log.Infof("pairing client %q finished", clientID)
	return nil
}

func (bs *BridgeAuthService) RefreshTokens(ctx context.Context, req *api.BridgeRefreshRequest) (*api.BridgeRefreshResponse, error) {
	claims, err := bs.ba.VerifyRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "refresh token is invalid")
	}

	// valid, err := s.bridgeStore.HasBridgeToken(claims.BridgeID, claims.RefreshUUID)
	// if err != nil || !valid {
	// 	return nil, status.Errorf(codes.Unauthenticated, "refresh token revoked")
	// }

	// bridge := s.bridgeStore.Find(claims.BridgeID)
	// if bridge == nil {
	// 	return nil, status.Errorf(codes.Internal, "cannot find bridge")
	// }

	tokens, err := bs.ba.GenerateTokens(claims.BridgeID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate refresh token")
	}

	// // How many days has the refresh token got left before expiry?
	// exp := claims.ExpiresAt.Time
	// rem := time.Now().Sub(exp).Hours() / 24.0

	// // If requested, don't revoke or replace the refresh token.
	// if req.RenewThreshold != 0 && rem < float64(req.RenewThreshold) {
	// 	_ = s.bridgeStore.RevokeBridgeToken(bridge.ID, claims.RefreshUUID)
	// 	tokens.RefreshToken = req.RefreshToken
	// 	tokens.RefreshUUID = claims.RefreshUUID
	// 	tokens.RefreshExpires = exp
	// }

	// s.bridgeStore.AddBridgeToken(bridge.ID, tokens.RefreshUUID, tokens.RefreshExpires)

	res := &api.BridgeRefreshResponse{
		RefreshToken: tokens.RefreshToken,
		AccessToken:  tokens.AccessToken,
	}

	return res, nil
}
