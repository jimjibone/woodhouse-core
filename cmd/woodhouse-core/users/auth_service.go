package users

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-4/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

type AuthService struct {
	clientsapi.UnimplementedUserAuthServiceServer
	log   *log.Context
	users *core.UserManager
	jwt   *JWTManager
}

func NewAuthService(users *core.UserManager, jwt *JWTManager) *AuthService {
	srv := &AuthService{
		log:   log.NewContext(log.DefaultLogger, "users-auth", log.DebugLevel),
		users: users,
		jwt:   jwt,
	}
	return srv
}

func (srv *AuthService) loginBase(in *clientsapi.UserLoginRequest) (*TokenDetails, error) {
	user := srv.users.Find(in.Username)

	// Special case if there are no admins and this user does not already
	// exist... add this user as an admin and log them in.
	if user == nil && !srv.users.HasAnAdmin() {
		// Add the user as an admin.
		admin, err := core.NewUser(in.Username, in.Password, auth.AdminRole)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to create user: %v", err)
		}

		// Add the user to the store.
		err = srv.users.Store(admin)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to add user: %v", err)
		}

		srv.log.Infof("added new user %q as an admin", in.Username)

		// Find the new admin user.
		user = srv.users.Find(in.Username)
	}

	if user == nil || !user.IsCorrectPassword(in.Password) {
		return nil, status.Errorf(codes.NotFound, "incorrect username/password")
	}

	tokens, err := srv.jwt.GenerateTokens(user.Username, user.Role)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot generate access token")
	}

	// err = srv.users.AddUserToken(user.Username, tokens.RefreshUUID, tokens.RefreshExpires)
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "cannot store access token: %s", err)
	// }

	return tokens, nil
}

func (srv *AuthService) Login(ctx context.Context, req *clientsapi.UserLoginRequest) (*clientsapi.UserLoginResponse, error) {
	tokens, err := srv.loginBase(req)
	if err != nil {
		return nil, err
	}

	res := &clientsapi.UserLoginResponse{
		RefreshToken: tokens.RefreshToken,
		AccessToken:  tokens.AccessToken,
	}

	return res, nil
}

func (srv *AuthService) LoginWeb(w http.ResponseWriter, r *http.Request) {
	handlePost(w, r, func(token string, w http.ResponseWriter, r *http.Request) {
		req := &clientsapi.UserLoginRequest{}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusUnprocessableEntity)
		}
		if err := protojson.Unmarshal(body, req); err != nil {
			http.Error(w, "invalid json", http.StatusUnprocessableEntity)
		} else {
			tokens, err := srv.loginBase(req)
			if err != nil {
				writeGRPCError(w, err)
			} else {
				http.SetCookie(w, &http.Cookie{
					Name:    "token",
					Value:   tokens.RefreshToken,
					Expires: tokens.RefreshExpires,
					// Secure: true,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
				resp := &clientsapi.UserLoginResponse{
					AccessToken: tokens.AccessToken,
				}
				body, err := protojson.Marshal(resp)
				if err != nil {
					http.Error(w, "failed to marshal response", http.StatusInternalServerError)
				}
				if _, err := w.Write(body); err != nil {
					http.Error(w, "failed to write response", http.StatusInternalServerError)
				}
			}
		}
	})
}

func (srv *AuthService) refreshBase(req *clientsapi.UserRefreshRequest) (*TokenDetails, error) {
	// Special case. If there are no admins then the webui needs to present the onboarding UI.
	if !srv.users.HasAnAdmin() {
		return nil, status.Errorf(codes.FailedPrecondition, "no admins registered")
	}

	// Verify and parse the JWT into claims.
	claims, err := srv.jwt.VerifyRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "refresh token is invalid: %s", err)
	}

	// Verify the token.
	// valid, err := srv.users.HasUserToken(claims.Username, claims.RefreshUUID)
	// if err != nil {
	// 	return nil, status.Errorf(codes.Unauthenticated, "refresh token revoked: %s", err)
	// }
	// if !valid {
	// 	return nil, status.Errorf(codes.Unauthenticated, "refresh token revoked")
	// }

	// Get the user, if they actually exist.
	user := srv.users.Find(claims.Username)
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	// How many days has the refresh token got left before expiry?
	exp := claims.ExpiresAt.Time
	remainingDays := time.Until(exp).Hours() / 24.0
	renewDays := (refreshTokenDuration / 2).Hours() / 24.0

	// If requested, don't revoke or replace the refresh token.
	var tokens *TokenDetails
	if remainingDays < renewDays {
		// Generate both tokens.
		tokens, err = srv.jwt.GenerateTokens(user.Username, user.Role)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate tokens: %s", err)
		}
	} else {
		// Generate only the access token.
		tokens, err = srv.jwt.GenerateAccessToken(user.Username, user.Role)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate access token: %s", err)
		}
		// We need to fill in the refresh token info.
		tokens.RefreshToken = req.RefreshToken
		tokens.RefreshUUID = claims.RefreshUUID
		tokens.RefreshExpires = exp
	}

	return tokens, nil
}

func (srv *AuthService) Refresh(ctx context.Context, req *clientsapi.UserRefreshRequest) (*clientsapi.UserRefreshResponse, error) {
	tokens, err := srv.refreshBase(req)
	if err != nil {
		return nil, err
	}

	res := &clientsapi.UserRefreshResponse{
		RefreshToken: tokens.RefreshToken,
		AccessToken:  tokens.AccessToken,
	}

	return res, nil
}

func (srv *AuthService) RefreshWeb(w http.ResponseWriter, r *http.Request) {
	handlePost(w, r, func(token string, w http.ResponseWriter, r *http.Request) {
		if token == "" {
			http.Error(w, "token not provided", http.StatusUnauthorized)
		} else {
			req := &clientsapi.UserRefreshRequest{}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read body", http.StatusUnprocessableEntity)
			}
			if err := protojson.Unmarshal(body, req); err != nil {
				http.Error(w, "invalid json", http.StatusUnprocessableEntity)
			} else {
				req.RefreshToken = token
				tokens, err := srv.refreshBase(req)
				if err != nil {
					writeGRPCError(w, err)
				} else {
					http.SetCookie(w, &http.Cookie{
						Name:    "token",
						Value:   tokens.RefreshToken,
						Expires: tokens.RefreshExpires,
						// Secure: true,
						HttpOnly: true,
						SameSite: http.SameSiteLaxMode,
					})
					resp := &clientsapi.UserLoginResponse{
						AccessToken: tokens.AccessToken,
					}
					body, err := protojson.Marshal(resp)
					if err != nil {
						http.Error(w, "failed to marshal response", http.StatusInternalServerError)
					}
					if _, err := w.Write(body); err != nil {
						http.Error(w, "failed to write response", http.StatusInternalServerError)
					}
				}
			}
		}
	})
}

func (srv *AuthService) logoutBase(req *clientsapi.UserLogoutRequest) error {
	_, err := srv.jwt.VerifyRefreshToken(req.RefreshToken)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "refresh token is invalid")
	}

	srv.jwt.RevokeRefreshToken(req.RefreshToken)

	return nil
}

func (srv *AuthService) Logout(ctx context.Context, req *clientsapi.UserLogoutRequest) (*clientsapi.UserLogoutResponse, error) {
	if err := srv.logoutBase(req); err != nil {
		return nil, err
	}

	return &clientsapi.UserLogoutResponse{}, nil
}

func (srv *AuthService) LogoutWeb(w http.ResponseWriter, r *http.Request) {
	handlePost(w, r, func(token string, w http.ResponseWriter, r *http.Request) {
		if token == "" {
			http.Error(w, "token not provided", http.StatusUnauthorized)
		} else {
			req := &clientsapi.UserLogoutRequest{}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read body", http.StatusUnprocessableEntity)
			}
			if err := protojson.Unmarshal(body, req); err != nil {
				http.Error(w, "invalid json", http.StatusUnprocessableEntity)
			} else {
				req.RefreshToken = token
				err := srv.logoutBase(req)
				if err != nil {
					writeGRPCError(w, err)
				} else {
					http.SetCookie(w, &http.Cookie{
						Name:  "token",
						Value: "",
						// Secure: true,
						HttpOnly: true,
						SameSite: http.SameSiteLaxMode,
					})
					resp := &clientsapi.UserLogoutResponse{}
					body, err := protojson.Marshal(resp)
					if err != nil {
						http.Error(w, "failed to marshal response", http.StatusInternalServerError)
					}
					if _, err := w.Write(body); err != nil {
						http.Error(w, "failed to write response", http.StatusInternalServerError)
					}
				}
			}
		}
	})
}

func handlePost(w http.ResponseWriter, r *http.Request, handler func(token string, w http.ResponseWriter, r *http.Request)) {
	if r.Method != http.MethodPost {
		http.Error(w, "only post method allowed", http.StatusMethodNotAllowed)
	} else if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "content-type must be application/json", http.StatusUnsupportedMediaType)
	} else {
		cookie := r.Header.Values("Cookie")
		token := ""
		for _, val := range cookie {
			if strings.HasPrefix(val, "token=") {
				token = strings.TrimPrefix(val, "token=")
				break
			}
		}
		handler(token, w, r)
	}
}

func writeGRPCError(w http.ResponseWriter, err error) {
	st := status.Convert(err)
	code := 0
	switch st.Code() {
	case codes.Unauthenticated:
		code = http.StatusUnauthorized
	case codes.NotFound:
		code = http.StatusBadRequest
	case codes.Internal:
		code = http.StatusInternalServerError
	case codes.InvalidArgument:
		code = http.StatusBadRequest
	case codes.FailedPrecondition:
		code = http.StatusPreconditionFailed
	default:
		code = http.StatusTeapot
	}
	http.Error(w, st.Message(), code)
}
