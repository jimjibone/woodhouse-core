package users

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jimjibone/log"
	"github.com/jimjibone/queue/v2"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/shared/random"
	"github.com/jimjibone/woodhouse-core/shared/stores"
	"gopkg.in/yaml.v3"
)

const (
	refreshTokenDurationDays = 30
	refreshTokenDuration     = refreshTokenDurationDays * 24 * time.Hour
	accessTokenDuration      = 15 * time.Minute
)

type JWTManager struct {
	log              *log.Context
	store            stores.Store
	wg               sync.WaitGroup
	mu               sync.RWMutex
	close            func()
	changed          bool
	refreshSecret    []byte
	accessSecret     []byte
	tokenAllocations map[string]TokenAllocation // key: refresh uuid
	revocations      *queue.Pub[string]         // refresh uuid
}

func NewJWTManager(store stores.Store) (*JWTManager, error) {
	ctx, close := context.WithCancel(context.Background())
	manager := &JWTManager{
		log:              log.NewContext(log.DefaultLogger, "users-jwt", log.DebugLevel),
		store:            store,
		close:            close,
		tokenAllocations: make(map[string]TokenAllocation),
		revocations:      queue.NewPub[string](),
	}

	// Load the previous config.
	err := manager.load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %s", err)
	}

	// Generate refresh and access tokens if necessary.
	if len(manager.refreshSecret) == 0 {
		key, err := random.GenerateRandomBytes(64)
		if err != nil {
			return nil, fmt.Errorf("failed to generate refresh secret: %w", err)
		}
		manager.refreshSecret = key
		manager.changed = true
	}
	if len(manager.accessSecret) == 0 {
		key, err := random.GenerateRandomBytes(64)
		if err != nil {
			return nil, fmt.Errorf("failed to generate access secret: %w", err)
		}
		manager.accessSecret = key
		manager.changed = true
	}

	// Save the config if changed.
	err = manager.saveIfChanged()
	if err != nil {
		return nil, fmt.Errorf("failed to save config: %s", err)
	}

	// Fire up a goroutine to save the config if it changes.
	manager.wg.Add(1)
	go func() {
		defer manager.wg.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// Clean expired tokens.
				manager.mu.Lock()
				for refreshUUID, alloc := range manager.tokenAllocations {
					if time.Now().After(alloc.Expires) {
						delete(manager.tokenAllocations, refreshUUID)
						manager.changed = true
					}
				}
				manager.mu.Unlock()

				// Save if changed.
				err := manager.saveIfChanged()
				if err != nil {
					manager.log.Fatalf("failed to save config: %s", err)
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return manager, nil
}

func (manager *JWTManager) Close() {
	manager.close()
	manager.wg.Wait()

	err := manager.saveIfChanged()
	if err != nil {
		manager.log.Errorf("failed to save config: %s", err)
	}
}

func (manager *JWTManager) load() error {
	if manager.store.Has("users-jwt") {
		// Load it.
		data, err := manager.store.Get("users-jwt")
		if err != nil {
			return err
		}

		// Decode it.
		config := struct {
			RefreshSecret    string                     `yaml:"refresh-secret"`
			AccessSecret     string                     `yaml:"access-secret"`
			TokenAllocations map[string]TokenAllocation `yaml:"token-allocations"`
		}{
			TokenAllocations: make(map[string]TokenAllocation),
		}
		err = yaml.NewDecoder(bytes.NewReader(data)).Decode(&config)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			if te, ok := err.(*yaml.TypeError); ok {
				fmt.Fprintln(os.Stderr, te.Errors)
			}
			// fmt.Println(yaml.FormatError(err, true, true))
			return err
		}

		// Read the config into the manager.
		manager.refreshSecret, err = base64.RawURLEncoding.DecodeString(config.RefreshSecret)
		if err != nil {
			return fmt.Errorf("refresh secret: %w", err)
		}
		manager.accessSecret, err = base64.RawURLEncoding.DecodeString(config.AccessSecret)
		if err != nil {
			return fmt.Errorf("access secret: %w", err)
		}
		manager.tokenAllocations = config.TokenAllocations

		// Maps may be nil after reading an empty list from the file. Why?
		if manager.tokenAllocations == nil {
			manager.tokenAllocations = make(map[string]TokenAllocation)
		}
	}
	return nil
}

func (manager *JWTManager) save() error {
	// Encode it.
	config := struct {
		RefreshSecret    string                     `yaml:"refresh-secret"`
		AccessSecret     string                     `yaml:"access-secret"`
		TokenAllocations map[string]TokenAllocation `yaml:"token-allocations"`
	}{
		RefreshSecret:    base64.RawURLEncoding.EncodeToString(manager.refreshSecret),
		AccessSecret:     base64.RawURLEncoding.EncodeToString(manager.accessSecret),
		TokenAllocations: manager.tokenAllocations,
	}
	data := &bytes.Buffer{}
	err := yaml.NewEncoder(data).Encode(config)
	if err != nil {
		return err
	}

	// Save it.
	err = manager.store.Set("users-jwt", data.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (manager *JWTManager) saveIfChanged() error {
	// Save the config if changed.
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if manager.changed {
		manager.log.Debugf("saving...")
		err := manager.save()
		if err != nil {
			return err
		}
		manager.changed = false
	}
	return nil
}

type TokenAllocation struct {
	Username string    `yaml:"username"`
	Expires  time.Time `yaml:"expires"`
}

type TokenDetails struct {
	AccessToken    string
	RefreshToken   string
	AccessUUID     string
	RefreshUUID    string
	AccessExpires  time.Time
	RefreshExpires time.Time
}

type RefreshTokenClaims struct {
	jwt.RegisteredClaims
	RefreshUUID string `json:"refresh_uuid"`
	Username    string `json:"username"`
}

type AccessTokenClaims struct {
	jwt.RegisteredClaims
	AccessUUID string `json:"access_uuid"`
	Username   string `json:"username"`
	// Fullname is carried in the access token (rather than looked up from
	// the users store) because the sidebar/profile chrome needs it on
	// every page, and holding a users stream open app-wide just for a
	// display name isn't worth it. Like Role, it's refreshed from the
	// store on every access-token refresh, so it can lag by up to one
	// refresh cycle after the user changes it.
	Fullname string    `json:"fullname"`
	Role     auth.Role `json:"role"`
	// ResetPassword travels in the access token for the same reason as
	// Fullname: the webui has to gate every page behind it, and that would
	// otherwise mean holding a users stream open app-wide just to read one
	// bool. It clears on the next access-token refresh after the user picks
	// a new password, which the webui forces rather than waiting out.
	ResetPassword bool   `json:"reset_password"`
	RefreshUUID   string `json:"refresh_uuid"`
}

func (manager *JWTManager) GenerateTokens(username, fullname string, role auth.Role, resetPassword bool) (*TokenDetails, error) {
	u1, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID: %v", err)
	}
	u2, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID: %v", err)
	}

	td := &TokenDetails{
		AccessToken:    "",
		RefreshToken:   "",
		AccessUUID:     u1.String(),
		RefreshUUID:    u2.String(),
		AccessExpires:  time.Now().Add(accessTokenDuration),
		RefreshExpires: time.Now().Add(refreshTokenDuration),
	}

	// Create Access Token
	atClaims := AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(td.AccessExpires),
		},
		AccessUUID:    td.AccessUUID,
		Username:      username,
		Fullname:      fullname,
		Role:          role,
		ResetPassword: resetPassword,
		RefreshUUID:   td.RefreshUUID,
	}
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	td.AccessToken, err = at.SignedString(manager.accessSecret)
	if err != nil {
		return nil, err
	}

	// Create Refresh Token
	rtClaims := RefreshTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(td.RefreshExpires),
		},
		RefreshUUID: td.RefreshUUID,
		Username:    username,
	}
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)
	td.RefreshToken, err = rt.SignedString(manager.refreshSecret)
	if err != nil {
		return nil, err
	}

	// Add the new token allocation.
	manager.mu.Lock()
	manager.log.Debugf("generated tokens for %s", username)
	manager.changed = true
	manager.tokenAllocations[td.RefreshUUID] = TokenAllocation{
		Username: username,
		Expires:  td.RefreshExpires,
	}
	manager.mu.Unlock()

	return td, nil
}

func (manager *JWTManager) GenerateAccessToken(username, fullname string, role auth.Role, resetPassword bool, refreshUUID string) (*TokenDetails, error) {
	u1, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID: %v", err)
	}

	accessExpires := time.Now().Add(accessTokenDuration)
	accessUUID := u1.String()

	// Create Access Token
	atClaims := AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpires),
		},
		AccessUUID:    accessUUID,
		Username:      username,
		Fullname:      fullname,
		Role:          role,
		ResetPassword: resetPassword,
		RefreshUUID:   refreshUUID,
	}
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	accessToken, err := at.SignedString(manager.accessSecret)
	if err != nil {
		return nil, err
	}

	manager.log.Debugf("generated access token for %s", username)

	td := &TokenDetails{
		AccessToken:   accessToken,
		AccessUUID:    accessUUID,
		AccessExpires: accessExpires,
	}

	return td, nil
}

func (manager *JWTManager) RevokeRefreshUUID(refreshUUID string) {
	manager.mu.Lock()
	manager.changed = true
	delete(manager.tokenAllocations, refreshUUID)
	manager.mu.Unlock()

	// Notify any active streams that this refresh UUID (and any access
	// tokens bound to it) has been revoked.
	manager.revocations.Pub(refreshUUID)
}

// RevokeUserRefreshTokens revokes every refresh token belonging to a user,
// except the one named by exceptRefreshUUID (pass "" to revoke all of them).
// It returns the number revoked.
//
// This is the eviction half of a password change: the current-password
// check stops a stolen session from locking the owner out, and this stops
// it from carrying on regardless. The caller's own session is spared so
// changing your password doesn't sign you out of the tab you're using.
func (manager *JWTManager) RevokeUserRefreshTokens(username string, exceptRefreshUUID string) int {
	manager.mu.Lock()
	revoked := make([]string, 0, len(manager.tokenAllocations))
	for refreshUUID, allocation := range manager.tokenAllocations {
		if allocation.Username != username || refreshUUID == exceptRefreshUUID {
			continue
		}
		delete(manager.tokenAllocations, refreshUUID)
		revoked = append(revoked, refreshUUID)
	}
	if len(revoked) > 0 {
		manager.changed = true
	}
	manager.mu.Unlock()

	// Published outside the lock: subscribers tear down live streams in
	// response, and doing that while holding the manager lock invites a
	// deadlock against anything that verifies a token on the way out.
	for _, refreshUUID := range revoked {
		manager.revocations.Pub(refreshUUID)
	}

	if len(revoked) > 0 {
		manager.log.Infof("revoked %d session(s) for %s", len(revoked), username)
	}

	return len(revoked)
}

// SubscribeRevocations returns a subscription that receives refresh UUIDs
// whenever their tokens are revoked via RevokeRefreshUUID. Call Close on
// the returned Sub when done.
func (manager *JWTManager) SubscribeRevocations() *queue.Sub[string] {
	return manager.revocations.NewSub()
}

func (manager *JWTManager) VerifyRefreshToken(refreshToken string) (*RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(
		refreshToken,
		&RefreshTokenClaims{},
		func(token *jwt.Token) (interface{}, error) {
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, fmt.Errorf("unexpected refresh token signing method")
			}

			return manager.refreshSecret, nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	claims, ok := token.Claims.(*RefreshTokenClaims)
	if !ok {
		return nil, fmt.Errorf("invalid refresh token claims")
	}

	// Check that the refresh UUID is in the allocated (allowed) list.
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	if _, found := manager.tokenAllocations[claims.RefreshUUID]; !found {
		return nil, fmt.Errorf("refresh token revoked")
	}

	return claims, nil
}

func (manager *JWTManager) VerifyAccessToken(accessToken string) (*AccessTokenClaims, error) {
	token, err := jwt.ParseWithClaims(
		accessToken,
		&AccessTokenClaims{},
		func(token *jwt.Token) (interface{}, error) {
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, fmt.Errorf("unexpected access token signing method")
			}

			return manager.accessSecret, nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	claims, ok := token.Claims.(*AccessTokenClaims)
	if !ok {
		return nil, fmt.Errorf("invalid access token claims")
	}

	// Bind the access token to its refresh allocation so that revoking the
	// refresh token (e.g. logout, rotation) kills any outstanding access
	// tokens immediately, rather than waiting for their own expiry.
	manager.mu.RLock()
	allocation, found := manager.tokenAllocations[claims.RefreshUUID]
	manager.mu.RUnlock()
	if !found || allocation.Username != claims.Username {
		return nil, fmt.Errorf("access token revoked")
	}

	return claims, nil
}
