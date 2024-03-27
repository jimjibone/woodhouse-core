package clients

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/random"
	"github.com/jimjibone/woodhouse-4/shared/stores"
	"gopkg.in/yaml.v2"
)

const (
	refreshTokenDuration = 30 * 24 * time.Hour
	accessTokenDuration  = 15 * time.Minute
)

type JWTManager struct {
	log              *log.Context
	store            stores.Store
	wg               sync.WaitGroup
	mu               sync.RWMutex
	close            func()
	changed          bool
	refreshSecret    string
	accessSecret     string
	tokenAllocations map[string]TokenAllocation // kwy: refresh token
}

func NewJWTManager(store stores.Store) (*JWTManager, error) {
	ctx, close := context.WithCancel(context.Background())
	manager := &JWTManager{
		log:              log.NewContext(log.DefaultLogger, "clients-jwt", log.DebugLevel),
		store:            store,
		close:            close,
		tokenAllocations: make(map[string]TokenAllocation),
	}

	// Load the previous config.
	err := manager.load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %s", err)
	}

	// Generate refresh and access tokens if necessary.
	if manager.refreshSecret == "" {
		key, err := random.GenerateRandomString(64)
		if err != nil {
			return nil, fmt.Errorf("failed to generate refresh secret: %w", err)
		}
		manager.refreshSecret = key
		manager.changed = true
	}
	if manager.accessSecret == "" {
		key, err := random.GenerateRandomString(64)
		if err != nil {
			return nil, fmt.Errorf("failed to generate access secret: %w", err)
		}
		manager.accessSecret = key
		manager.changed = true
	}

	// Save the config if changed.
	if manager.changed {
		err := manager.save()
		if err != nil {
			return nil, fmt.Errorf("failed to save config: %s", err)
		}
		manager.changed = false
	}

	// Fire up a goroutine to save the config if it changes.
	manager.wg.Add(1)
	go func() {
		manager.wg.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
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
	if manager.store.Has("clients-jwt") {
		// Load it.
		data, err := manager.store.Get("clients-jwt")
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
		manager.refreshSecret = config.RefreshSecret
		manager.accessSecret = config.AccessSecret
		manager.tokenAllocations = config.TokenAllocations
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
		RefreshSecret:    manager.refreshSecret,
		AccessSecret:     manager.accessSecret,
		TokenAllocations: manager.tokenAllocations,
	}
	data := &bytes.Buffer{}
	err := yaml.NewEncoder(data).Encode(config)
	if err != nil {
		return err
	}

	// Save it.
	err = manager.store.Set("clients-jwt", data.Bytes())
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
	ClientID string    `yaml:"client_id"`
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
	ClientID    string `json:"client_id"`
}

type AccessTokenClaims struct {
	jwt.RegisteredClaims
	AccessUUID string `json:"access_uuid"`
	ClientID   string `json:"client_id"`
	// Perms      []perms.Perm `json:"perms"`
}

func (manager *JWTManager) GenerateTokens(id string) (*TokenDetails, error) {
	u1, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID: %v", err)
	}
	u2, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID: %v", err)
	}

	td := &TokenDetails{}
	td.AccessExpires = time.Now().Add(accessTokenDuration)
	td.AccessUUID = u1.String()
	td.RefreshExpires = time.Now().Add(refreshTokenDuration)
	td.RefreshUUID = u2.String()

	// Create Access Token
	atClaims := AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(td.AccessExpires),
		},
		AccessUUID: td.AccessUUID,
		ClientID:   id,
		// Perms:      perms,
	}
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	td.AccessToken, err = at.SignedString([]byte(manager.accessSecret))
	if err != nil {
		return nil, err
	}

	// Create Refresh Token
	rtClaims := RefreshTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(td.RefreshExpires),
		},
		RefreshUUID: td.RefreshUUID,
		ClientID:    id,
	}
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)
	td.RefreshToken, err = rt.SignedString([]byte(manager.refreshSecret))
	if err != nil {
		return nil, err
	}

	// Add the new token allocation.
	manager.mu.Lock()
	manager.log.Debugf("generated token for %s", id)
	manager.changed = true
	manager.tokenAllocations[td.RefreshUUID] = TokenAllocation{
		ClientID: id,
		Expires:  td.RefreshExpires,
	}
	manager.mu.Unlock()

	return td, nil
}

func (manager *JWTManager) RevokeToken(refreshUUID string) {
	manager.mu.Lock()
	manager.changed = true
	delete(manager.tokenAllocations, refreshUUID)
	manager.mu.Unlock()
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

			return []byte(manager.refreshSecret), nil
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
	if _, found := manager.tokenAllocations[claims.RefreshUUID]; !found {
		return nil, fmt.Errorf("refresh token revoked")
	}
	manager.mu.RUnlock()

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

			return []byte(manager.accessSecret), nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	claims, ok := token.Claims.(*AccessTokenClaims)
	if !ok {
		return nil, fmt.Errorf("invalid access token claims")
	}

	return claims, nil
}
