package clients

import (
	"context"
	"errors"
	"time"

	"github.com/jimjibone/log"
	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-core/shared/cert"
	"github.com/jimjibone/woodhouse-core/shared/crypt"
	"github.com/jimjibone/woodhouse-core/shared/sas"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// pairingDecisionTimeout bounds how long the server waits for the user to
// confirm or deny a pairing request before aborting.
const pairingDecisionTimeout = 3 * time.Minute

type AuthService struct {
	clientsapi.UnimplementedAuthServiceServer
	log           *log.Context
	cm            *cert.CertManager
	jwt           *JWTManager
	clientManager *core.ClientManager
}

func NewAuthService(cm *cert.CertManager, ba *JWTManager, clientManager *core.ClientManager) *AuthService {
	return &AuthService{
		log:           log.NewContext(log.DefaultLogger, "clients-auth", log.DebugLevel),
		cm:            cm,
		jwt:           ba,
		clientManager: clientManager,
	}
}

func (as *AuthService) Pair(server clientsapi.AuthService_PairServer) error {
	if as.clientManager == nil {
		return status.Errorf(codes.FailedPrecondition, "client manager not configured")
	}

	// 1. Receive the client's id and ephemeral public key (PKa).
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

	pka := req.ClientPubkey
	clientPub, err := sas.ParsePublicKey(pka)
	if err != nil {
		as.log.Warnf("pairing client %q sent an invalid public key: %s", clientID, err)
		return status.Errorf(codes.InvalidArgument, "invalid client public key")
	}

	// Register the pending request, obtaining its id and the channel that
	// delivers the user's confirm/deny decision.
	pairingRequest := &core.PairingRequest{ClientID: clientID}
	requestID, decision, err := as.clientManager.AddPairingRequest(pairingRequest)
	if err != nil {
		if errors.Is(err, core.ErrPairingInProgress) {
			as.log.Infof("pairing client %q rejected: already in progress", clientID)
			return status.Errorf(codes.AlreadyExists, "a pairing request is already in progress")
		}
		if errors.Is(err, core.ErrTooManyPairings) {
			return status.Errorf(codes.ResourceExhausted, "too many pending pairing requests")
		}
		as.log.Warnf("pairing client %q failed to add pairing request: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to add pairing request")
	}
	defer func() { _ = as.clientManager.RemovePairingRequest(requestID) }()

	// 2. Generate our ephemeral key (PKb) and nonce (Nb), and commit to Nb
	// before we learn the client's nonce.
	serverPriv, err := sas.GenerateKey()
	if err != nil {
		as.log.Warnf("pairing client %q failed to generate key: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to generate key")
	}
	pkb := serverPriv.PublicKey().Bytes()
	nb, err := sas.Nonce()
	if err != nil {
		as.log.Warnf("pairing client %q failed to generate nonce: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to generate nonce")
	}

	err = server.Send(&clientsapi.PairResponse{
		State:        clientsapi.PairResponse_KeyExchange,
		ServerPubkey: pkb,
		Commitment:   sas.Commit(pkb, pka, clientID, nb),
	})
	if err != nil {
		as.log.Warnf("pairing client %q error when sending key exchange: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to send key exchange")
	}

	// 3. Receive the client's nonce (Na).
	req, err = server.Recv()
	if err != nil {
		as.log.Warnf("pairing client %q failed to receive nonce: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to receive nonce")
	}
	na := req.ClientNonce
	if len(na) != sas.NonceSize {
		return status.Errorf(codes.InvalidArgument, "invalid client nonce")
	}

	// 4. Reveal our nonce (Nb) so the client can verify the commitment.
	err = server.Send(&clientsapi.PairResponse{
		State:       clientsapi.PairResponse_Reveal,
		ServerNonce: nb,
	})
	if err != nil {
		as.log.Warnf("pairing client %q error when sending reveal: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to send reveal")
	}

	// 5. Derive the SAS (compared by the user) and the AES-256 session key.
	sasCode, key, err := sas.Derive(serverPriv, clientPub, pka, pkb, clientID, na, nb)
	if err != nil {
		as.log.Warnf("pairing client %q failed to derive sas: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to derive sas")
	}

	// 6. Publish the SAS so the web UI shows it, then wait for the user.
	if err := as.clientManager.SetPairingSAS(requestID, sasCode); err != nil {
		as.log.Warnf("pairing client %q failed to publish sas: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to publish sas")
	}
	as.log.Infof("pairing client %q awaiting user confirmation", clientID)

	// 7. Block until the user confirms or denies, sending keepalives meanwhile.
	confirmed, err := as.awaitDecision(server, clientID, decision)
	if err != nil {
		return err
	}
	if !confirmed {
		as.log.Infof("pairing client %q denied", clientID)
		return status.Errorf(codes.PermissionDenied, "pairing denied")
	}

	// 8. Send the server certificate, encrypted under the session key.
	encrypted, err := crypt.Encrypt(as.cm.CertPEM(), key)
	if err != nil {
		as.log.Warnf("pairing client %q failed to encrypt cert: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to encrypt cert")
	}
	err = server.Send(&clientsapi.PairResponse{
		State: clientsapi.PairResponse_Confirmed,
		Data:  encrypted,
	})
	if err != nil {
		as.log.Warnf("pairing client %q error when sending cert: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to send cert")
	}

	// 9. Generate and send the refresh token, encrypted under the session key.
	tokens, err := as.jwt.GenerateTokens(clientID)
	if err != nil {
		as.log.Warnf("pairing client %q failed to generate token: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to generate token")
	}
	encrypted, err = crypt.Encrypt([]byte(tokens.RefreshToken), key)
	if err != nil {
		as.log.Warnf("pairing client %q failed to encrypt token: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to encrypt token")
	}
	err = server.Send(&clientsapi.PairResponse{Data: encrypted})
	if err != nil {
		as.log.Warnf("pairing client %q error when sending token: %s", clientID, err)
		return status.Errorf(codes.Internal, "failed to send token")
	}

	as.clientManager.FinalisePairingRequest(pairingRequest)
	as.log.Infof("pairing client %q finished", clientID)
	return nil
}

// awaitDecision blocks until the user confirms (true) or denies (false) the
// pairing request, the client disconnects, or the decision times out. It sends
// periodic Pending keepalives so the stream and any intermediaries stay alive
// and a dead client is detected while waiting.
func (as *AuthService) awaitDecision(server clientsapi.AuthService_PairServer, clientID string, decision <-chan bool) (bool, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	timeout := time.NewTimer(pairingDecisionTimeout)
	defer timeout.Stop()

	for {
		select {
		case <-server.Context().Done():
			return false, status.Errorf(codes.Canceled, "context canceled")

		case ok := <-decision:
			return ok, nil

		case <-timeout.C:
			as.log.Infof("pairing client %q timed out awaiting confirmation", clientID)
			return false, status.Errorf(codes.DeadlineExceeded, "pairing timed out")

		case <-ticker.C:
			err := server.Send(&clientsapi.PairResponse{State: clientsapi.PairResponse_Pending})
			if err != nil {
				code := status.Code(err)
				if code == codes.Unavailable {
					as.log.Infof("pairing client %q went offline", clientID)
					return false, status.Errorf(code, "client offline")
				}
				as.log.Warnf("pairing client %q error when sending pending: %s", clientID, err)
				return false, status.Errorf(code, "failed to send pending")
			}
		}
	}
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
			return nil, status.Errorf(codes.Internal, "failed to generate tokens: %s", err)
		}
		refreshToken = tokens.RefreshToken
		accessToken = tokens.AccessToken
	} else {
		// Generate only the access token.
		token, err := as.jwt.GenerateAccessToken(claims.ClientID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate access token: %s", err)
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
