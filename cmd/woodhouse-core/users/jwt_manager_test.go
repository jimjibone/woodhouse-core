package users

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
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

	td, err := manager.GenerateTokens("alice", "Alice Example", auth.UserRole, false)
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}

	atClaims, err := manager.VerifyAccessToken(td.AccessToken)
	if err != nil {
		t.Fatalf("VerifyAccessToken: unexpected error: %s", err)
	}
	if atClaims.Username != "alice" {
		t.Fatalf("VerifyAccessToken: unexpected username %q", atClaims.Username)
	}
	if atClaims.Fullname != "Alice Example" {
		t.Fatalf("VerifyAccessToken: unexpected fullname %q", atClaims.Fullname)
	}
	if atClaims.RefreshUUID != td.RefreshUUID {
		t.Fatalf("VerifyAccessToken: expected refresh uuid %q, got %q", td.RefreshUUID, atClaims.RefreshUUID)
	}

	rtClaims, err := manager.VerifyRefreshToken(td.RefreshToken)
	if err != nil {
		t.Fatalf("VerifyRefreshToken: unexpected error: %s", err)
	}
	if rtClaims.Username != "alice" {
		t.Fatalf("VerifyRefreshToken: unexpected username %q", rtClaims.Username)
	}
}

// TestJWTManager_GenerateTokens_EmptyFullnameRoundTrip asserts that a user
// who has never set a display name (fullname == "") round-trips through
// the access token as an empty string rather than erroring or being
// dropped - this is the normal state for a brand new user, not an edge
// case to reject.
func TestJWTManager_GenerateTokens_EmptyFullnameRoundTrip(t *testing.T) {
	manager := newTestJWTManager(t)

	td, err := manager.GenerateTokens("alice", "", auth.UserRole, false)
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}

	atClaims, err := manager.VerifyAccessToken(td.AccessToken)
	if err != nil {
		t.Fatalf("VerifyAccessToken: unexpected error: %s", err)
	}
	if atClaims.Fullname != "" {
		t.Fatalf("VerifyAccessToken: expected empty fullname, got %q", atClaims.Fullname)
	}
}

func TestJWTManager_RevokeRefreshUUID_KillsAccessAndRefreshTokens(t *testing.T) {
	manager := newTestJWTManager(t)

	td, err := manager.GenerateTokens("alice", "Alice Example", auth.UserRole, false)
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

	manager.RevokeRefreshUUID(td.RefreshUUID)

	if _, err := manager.VerifyAccessToken(td.AccessToken); err == nil {
		t.Fatalf("VerifyAccessToken after revoke: expected error, got nil")
	}
	if _, err := manager.VerifyRefreshToken(td.RefreshToken); err == nil {
		t.Fatalf("VerifyRefreshToken after revoke: expected error, got nil")
	}
}

func TestJWTManager_GenerateAccessToken_BoundToLiveRefreshUUID(t *testing.T) {
	manager := newTestJWTManager(t)

	td, err := manager.GenerateTokens("alice", "Alice Example", auth.UserRole, false)
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}

	at, err := manager.GenerateAccessToken("alice", "Alice Example", auth.UserRole, false, td.RefreshUUID)
	if err != nil {
		t.Fatalf("GenerateAccessToken: unexpected error: %s", err)
	}

	if _, err := manager.VerifyAccessToken(at.AccessToken); err != nil {
		t.Fatalf("VerifyAccessToken: expected success while refresh uuid is live, got: %s", err)
	}

	manager.RevokeRefreshUUID(td.RefreshUUID)

	if _, err := manager.VerifyAccessToken(at.AccessToken); err == nil {
		t.Fatalf("VerifyAccessToken: expected failure after refresh uuid revoked, got nil")
	}
}

func TestJWTManager_GenerateAccessToken_CrossUserBindingFails(t *testing.T) {
	manager := newTestJWTManager(t)

	aliceTokens, err := manager.GenerateTokens("alice", "Alice Example", auth.UserRole, false)
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}

	// Mint an access token for "bob" but bound to alice's refresh UUID.
	bobToken, err := manager.GenerateAccessToken("bob", "Bob Example", auth.UserRole, false, aliceTokens.RefreshUUID)
	if err != nil {
		t.Fatalf("GenerateAccessToken: unexpected error: %s", err)
	}

	if _, err := manager.VerifyAccessToken(bobToken.AccessToken); err == nil {
		t.Fatalf("VerifyAccessToken: expected failure for cross-user binding, got nil")
	}
}

func TestJWTManager_VerifyAccessToken_RejectsLegacyTokenWithoutRefreshUUID(t *testing.T) {
	manager := newTestJWTManager(t)

	// A live refresh allocation exists for alice...
	_, err := manager.GenerateTokens("alice", "Alice Example", auth.UserRole, false)
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}

	// ...but hand-sign a token using the OLD claims shape (no refresh_uuid
	// field at all), as if it had been issued before this change shipped.
	type legacyAccessTokenClaims struct {
		jwt.RegisteredClaims
		AccessUUID string    `json:"access_uuid"`
		Username   string    `json:"username"`
		Role       auth.Role `json:"role"`
	}
	legacyClaims := legacyAccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenDuration)),
		},
		AccessUUID: "legacy-access-uuid",
		Username:   "alice",
		Role:       auth.UserRole,
	}
	legacy := jwt.NewWithClaims(jwt.SigningMethodHS256, legacyClaims)
	legacyToken, err := legacy.SignedString(manager.accessSecret)
	if err != nil {
		t.Fatalf("failed to sign legacy token: %s", err)
	}

	if _, err := manager.VerifyAccessToken(legacyToken); err == nil {
		t.Fatalf("VerifyAccessToken: expected legacy token (no refresh_uuid) to be rejected, got nil")
	}
}

func TestJWTManager_SubscribeRevocations(t *testing.T) {
	manager := newTestJWTManager(t)

	td, err := manager.GenerateTokens("alice", "Alice Example", auth.UserRole, false)
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}

	sub := manager.SubscribeRevocations()
	defer sub.Close()

	manager.RevokeRefreshUUID(td.RefreshUUID)

	select {
	case revoked := <-sub.Sub():
		if revoked != td.RefreshUUID {
			t.Fatalf("expected revocation notice for %q, got %q", td.RefreshUUID, revoked)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for revocation notice")
	}
}

func TestJWTManager_RevokeUserRefreshTokens_SparesTheCallersSession(t *testing.T) {
	manager := newTestJWTManager(t)

	// Three sessions for alice (phone, laptop, the tab doing the change)
	// and one for bob, who must be left entirely alone.
	phone, err := manager.GenerateTokens("alice", "Alice Example", auth.UserRole, false)
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}
	laptop, err := manager.GenerateTokens("alice", "Alice Example", auth.UserRole, false)
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}
	current, err := manager.GenerateTokens("alice", "Alice Example", auth.UserRole, false)
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}
	bob, err := manager.GenerateTokens("bob", "Bob Example", auth.UserRole, false)
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}

	if n := manager.RevokeUserRefreshTokens("alice", current.RefreshUUID); n != 2 {
		t.Fatalf("RevokeUserRefreshTokens: expected 2 revocations, got %d", n)
	}

	if _, err := manager.VerifyRefreshToken(phone.RefreshToken); err == nil {
		t.Error("expected the phone session to be revoked")
	}
	if _, err := manager.VerifyRefreshToken(laptop.RefreshToken); err == nil {
		t.Error("expected the laptop session to be revoked")
	}
	// Changing your password must not sign you out of the tab you did it in.
	if _, err := manager.VerifyRefreshToken(current.RefreshToken); err != nil {
		t.Errorf("expected the calling session to survive, got: %s", err)
	}
	// One user's password change must not touch anybody else's sessions.
	if _, err := manager.VerifyRefreshToken(bob.RefreshToken); err != nil {
		t.Errorf("expected bob's session to survive, got: %s", err)
	}
}

func TestJWTManager_RevokeUserRefreshTokens_EmptyExceptRevokesAll(t *testing.T) {
	manager := newTestJWTManager(t)

	first, err := manager.GenerateTokens("alice", "Alice Example", auth.UserRole, false)
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}
	second, err := manager.GenerateTokens("alice", "Alice Example", auth.UserRole, false)
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}

	// This is the admin-reset path: the target keeps no session at all.
	if n := manager.RevokeUserRefreshTokens("alice", ""); n != 2 {
		t.Fatalf("RevokeUserRefreshTokens: expected 2 revocations, got %d", n)
	}

	if _, err := manager.VerifyRefreshToken(first.RefreshToken); err == nil {
		t.Error("expected the first session to be revoked")
	}
	if _, err := manager.VerifyRefreshToken(second.RefreshToken); err == nil {
		t.Error("expected the second session to be revoked")
	}
}

func TestJWTManager_RevokeUserRefreshTokens_NotifiesSubscribers(t *testing.T) {
	manager := newTestJWTManager(t)

	td, err := manager.GenerateTokens("alice", "Alice Example", auth.UserRole, false)
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}

	sub := manager.SubscribeRevocations()
	defer sub.Close()

	// Live streams are torn down off the back of this notice, so a bulk
	// revoke that skipped it would leave open streams on a dead session.
	manager.RevokeUserRefreshTokens("alice", "")

	select {
	case revoked := <-sub.Sub():
		if revoked != td.RefreshUUID {
			t.Fatalf("expected revocation notice for %q, got %q", td.RefreshUUID, revoked)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for revocation notice")
	}
}

func TestJWTManager_GenerateTokens_CarriesResetPasswordClaim(t *testing.T) {
	manager := newTestJWTManager(t)

	td, err := manager.GenerateTokens("alice", "Alice Example", auth.UserRole, true)
	if err != nil {
		t.Fatalf("GenerateTokens: unexpected error: %s", err)
	}

	claims, err := manager.VerifyAccessToken(td.AccessToken)
	if err != nil {
		t.Fatalf("VerifyAccessToken: unexpected error: %s", err)
	}
	// The webui gates the whole app on this claim, so it has to survive
	// the round trip through the token.
	if !claims.ResetPassword {
		t.Error("ResetPassword claim did not survive the token round trip")
	}

	td, err = manager.GenerateAccessToken("alice", "Alice Example", auth.UserRole, false, td.RefreshUUID)
	if err != nil {
		t.Fatalf("GenerateAccessToken: unexpected error: %s", err)
	}
	claims, err = manager.VerifyAccessToken(td.AccessToken)
	if err != nil {
		t.Fatalf("VerifyAccessToken: unexpected error: %s", err)
	}
	// A refresh after the user picks their own password is what clears it.
	if claims.ResetPassword {
		t.Error("ResetPassword claim still set after a refresh that cleared it")
	}
}
