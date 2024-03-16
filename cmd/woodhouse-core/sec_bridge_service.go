package main

import (
	"slices"
	"time"

	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/cert"
	"github.com/jimjibone/woodhouse-4/shared/crypt"
	"github.com/jimjibone/woodhouse-4/shared/random"
	"github.com/schollz/pake/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SecBridgeService struct {
	api.SecBridgeServiceServer
	cm *cert.CertManager
	ba *auth.BridgeAuth
}

func NewSecBridgeService(cm *cert.CertManager, ba *auth.BridgeAuth) *SecBridgeService {
	return &SecBridgeService{
		cm: cm,
		ba: ba,
	}
}

func (bs *SecBridgeService) DoPairing(server api.SecBridgeService_DoPairingServer) error {
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
			err = server.Send(&api.DoPairingResponse{
				State: api.DoPairingResponse_Pending,
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

	// 3. Start the PAKE handshake using the key provided by the user.
	log.Debugf("pairing client %q initialising pake", clientID)
	pakep, err := pake.InitCurve([]byte("redacted"), 0, "p521")
	if err != nil {
		log.Warnf("pairing client %q failed to init pake: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to init pake: %s", err)
	}

	// 4. Send the first handshake blob to the client.
	log.Debugf("pairing client %q sending first handshake blob", clientID)
	err = server.Send(&api.DoPairingResponse{
		State: api.DoPairingResponse_Handshake,
		Data:  pakep.Bytes(),
	})
	if err != nil {
		log.Warnf("pairing client %q error when sending handshake start: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to send handshake")
	}

	// 5. Receive the second handshake blob from the client.
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

	// 6. We should now have the session key. Let's confirm this by sending a
	// test (an encrypted blob of random bytes).
	key, err := pakep.SessionKey()
	if err != nil {
		log.Warnf("pairing client %q failed to get session key: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to prepare test")
	}
	log.Debugf("pairing client %q generated key", clientID)
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
	err = server.Send(&api.DoPairingResponse{
		Data: encrypted,
	})
	if err != nil {
		log.Warnf("pairing client %q failed to send test: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to send test")
	}

	// 7. Receive the test back.
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

	// 8. Send the server's certificate.
	encrypted, err = crypt.Encrypt(bs.cm.CertPEM(), key)
	if err != nil {
		log.Warnf("pairing client %q failed to encrypt cert: %s", clientID, err)
		return status.Errorf(codes.PermissionDenied, "failed to encrypt cert")
	}
	log.Debugf("pairing client %q sending cert", clientID)
	err = server.Send(&api.DoPairingResponse{
		Data: encrypted,
	})
	if err != nil {
		log.Warnf("pairing client %q error when sending cert: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to send cert")
	}

	// 9. Generate refresh auth token for the client and send it.
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
	err = server.Send(&api.DoPairingResponse{
		Data: encrypted,
	})
	if err != nil {
		log.Warnf("pairing client %q error when sending token: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to send token")
	}

	log.Infof("pairing client %q finished", clientID)
	return nil
}
