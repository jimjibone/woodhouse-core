package clients

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

// newTestJWTManager builds a JWTManager over an in-memory store, cleaning
// it up when the test finishes.
func newTestJWTManager(t *testing.T) *JWTManager {
	t.Helper()
	store := stores.NewMemStore()
	manager, err := NewJWTManager(store)
	if err != nil {
		t.Fatalf("failed to create jwt manager: %s", err)
	}
	t.Cleanup(manager.Close)
	return manager
}

func TestJWTManager_GenerateTokens_VerifyRoundTrip(t *testing.T) {
	manager := newTestJWTManager(t)

	td, err := manager.GenerateTokens("client-a")
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}

	atClaims, err := manager.VerifyAccessToken(td.AccessToken)
	if err != nil {
		t.Fatalf("VerifyAccessToken: unexpected error: %s", err)
	}
	if atClaims.ClientID != "client-a" {
		t.Fatalf("VerifyAccessToken: unexpected client id %q", atClaims.ClientID)
	}
	if atClaims.RefreshUUID != td.RefreshUUID {
		t.Fatalf("VerifyAccessToken: expected refresh uuid %q, got %q", td.RefreshUUID, atClaims.RefreshUUID)
	}

	rtClaims, err := manager.VerifyRefreshToken(td.RefreshToken)
	if err != nil {
		t.Fatalf("VerifyRefreshToken: unexpected error: %s", err)
	}
	if rtClaims.ClientID != "client-a" {
		t.Fatalf("VerifyRefreshToken: unexpected client id %q", rtClaims.ClientID)
	}
}

func TestJWTManager_RevokeToken_KillsAccessAndRefreshTokens(t *testing.T) {
	manager := newTestJWTManager(t)

	td, err := manager.GenerateTokens("client-a")
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}

	// Sanity check both verify before revocation.
	if _, err := manager.VerifyAccessToken(td.AccessToken); err != nil {
		t.Fatalf("VerifyAccessToken before revoke: unexpected error: %s", err)
	}
	if _, err := manager.VerifyRefreshToken(td.RefreshToken); err != nil {
		t.Fatalf("VerifyRefreshToken before revoke: unexpected error: %s", err)
	}

	manager.RevokeToken(td.RefreshUUID)

	if _, err := manager.VerifyAccessToken(td.AccessToken); err == nil {
		t.Fatalf("VerifyAccessToken after revoke: expected error, got nil")
	}
	if _, err := manager.VerifyRefreshToken(td.RefreshToken); err == nil {
		t.Fatalf("VerifyRefreshToken after revoke: expected error, got nil")
	}
}

func TestJWTManager_GenerateAccessToken_BoundToLiveRefreshUUID(t *testing.T) {
	manager := newTestJWTManager(t)

	td, err := manager.GenerateTokens("client-a")
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}

	at, err := manager.GenerateAccessToken("client-a", td.RefreshUUID)
	if err != nil {
		t.Fatalf("GenerateAccessToken: unexpected error: %s", err)
	}

	if _, err := manager.VerifyAccessToken(at); err != nil {
		t.Fatalf("VerifyAccessToken: expected success while refresh uuid is live, got: %s", err)
	}

	manager.RevokeToken(td.RefreshUUID)

	if _, err := manager.VerifyAccessToken(at); err == nil {
		t.Fatalf("VerifyAccessToken: expected failure after refresh uuid revoked, got nil")
	}
}

func TestJWTManager_GenerateAccessToken_CrossClientBindingFails(t *testing.T) {
	manager := newTestJWTManager(t)

	aTokens, err := manager.GenerateTokens("client-a")
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}

	// Mint an access token for "client-b" but bound to client-a's refresh UUID.
	bToken, err := manager.GenerateAccessToken("client-b", aTokens.RefreshUUID)
	if err != nil {
		t.Fatalf("GenerateAccessToken: unexpected error: %s", err)
	}

	if _, err := manager.VerifyAccessToken(bToken); err == nil {
		t.Fatalf("VerifyAccessToken: expected failure for cross-client binding, got nil")
	}
}

func TestJWTManager_RevokeClient_KillsAllAccessTokensAndNotifies(t *testing.T) {
	manager := newTestJWTManager(t)

	td1, err := manager.GenerateTokens("client-a")
	if err != nil {
		t.Fatalf("GenerateTokens (1): unexpected error: %s", err)
	}
	td2, err := manager.GenerateTokens("client-a")
	if err != nil {
		t.Fatalf("GenerateTokens (2): unexpected error: %s", err)
	}

	// Sanity check both access tokens verify before revocation.
	if _, err := manager.VerifyAccessToken(td1.AccessToken); err != nil {
		t.Fatalf("VerifyAccessToken (1) before revoke: unexpected error: %s", err)
	}
	if _, err := manager.VerifyAccessToken(td2.AccessToken); err != nil {
		t.Fatalf("VerifyAccessToken (2) before revoke: unexpected error: %s", err)
	}

	sub := manager.SubscribeRevocations()
	defer sub.Close()

	manager.RevokeClient("client-a")

	if _, err := manager.VerifyAccessToken(td1.AccessToken); err == nil {
		t.Fatalf("VerifyAccessToken (1) after RevokeClient: expected error, got nil")
	}
	if _, err := manager.VerifyAccessToken(td2.AccessToken); err == nil {
		t.Fatalf("VerifyAccessToken (2) after RevokeClient: expected error, got nil")
	}

	select {
	case revoked := <-sub.Sub():
		if revoked != "client-a" {
			t.Fatalf("expected revocation notice for %q, got %q", "client-a", revoked)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for revocation notice")
	}
}

func TestJWTManager_VerifyAccessToken_RejectsLegacyTokenWithoutRefreshUUID(t *testing.T) {
	manager := newTestJWTManager(t)

	// A live refresh allocation exists for client-a...
	_, err := manager.GenerateTokens("client-a")
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}

	// ...but hand-sign a token using the OLD claims shape (no refresh_uuid
	// field at all), as if it had been issued before this change shipped.
	type legacyAccessTokenClaims struct {
		jwt.RegisteredClaims
		AccessUUID string `json:"access_uuid"`
		ClientID   string `json:"client_id"`
	}
	legacyClaims := legacyAccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenDuration)),
		},
		AccessUUID: "legacy-access-uuid",
		ClientID:   "client-a",
	}
	legacy := jwt.NewWithClaims(jwt.SigningMethodHS256, legacyClaims)
	legacyToken, err := legacy.SignedString([]byte(manager.accessSecret))
	if err != nil {
		t.Fatalf("failed to sign legacy token: %s", err)
	}

	if _, err := manager.VerifyAccessToken(legacyToken); err == nil {
		t.Fatalf("VerifyAccessToken: expected legacy token (no refresh_uuid) to be rejected, got nil")
	}
}
