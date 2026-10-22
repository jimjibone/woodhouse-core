package users

import (
	"context"
	"testing"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/limits"
	"github.com/jimjibone/woodhouse-core/shared/stores"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// createTestInvite makes an invite directly through the manager and returns its
// token, as an admin's CreateInvite call would hand it to the invitee.
func createTestInvite(t *testing.T, srv *AuthService, username, fullname string, role auth.Role) string {
	t.Helper()
	_, token, err := srv.invites.Create(username, fullname, role, "admin")
	if err != nil {
		t.Fatalf("Create invite for %q: unexpected error: %s", username, err)
	}
	return token
}

func wantCode(t *testing.T, err error, want codes.Code) *status.Status {
	t.Helper()
	if status.Code(err) != want {
		t.Fatalf("expected %s, got %v", want, err)
	}
	return status.Convert(err)
}

func TestAcceptInvite_CreatesUserAndLogsIn(t *testing.T) {
	srv, userManager, _ := newTestAuthService(t, limits.Config{})
	token := createTestInvite(t, srv, "alice", "Alice Example", auth.UserRole)

	res, err := srv.AcceptInvite(ctxWithIP("10.1.0.1"), &clientsapi.AcceptInviteRequest{Token: token, Password: "alicepass1"})
	if err != nil {
		t.Fatalf("AcceptInvite: unexpected error: %v", err)
	}
	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Fatalf("expected tokens, got %+v", res)
	}

	user := userManager.Find("alice")
	if user == nil {
		t.Fatal("the invited user was not created")
	}
	if user.Role != auth.UserRole || user.Fullname != "Alice Example" {
		t.Errorf("user %+v does not carry the invite's role and fullname", user)
	}
	if !user.IsCorrectPassword("alicepass1") {
		t.Error("the password the invitee chose does not work")
	}
	// They picked this password themselves, so no forced change.
	if user.ResetPassword {
		t.Error("ResetPassword is set on an account whose owner chose the password")
	}

	claims, err := srv.jwt.VerifyAccessToken(res.AccessToken)
	if err != nil {
		t.Fatalf("VerifyAccessToken: %s", err)
	}
	if claims.Username != "alice" || claims.Role != auth.UserRole || claims.ResetPassword {
		t.Errorf("access token claims %+v do not describe the new user", claims)
	}

	// The invite is spent.
	if got := srv.invites.List(); len(got) != 0 {
		t.Errorf("the redeemed invite is still pending: %+v", got)
	}
}

func TestAcceptInvite_AdminRoleFromInvite(t *testing.T) {
	srv, userManager, _ := newTestAuthService(t, limits.Config{})
	token := createTestInvite(t, srv, "carol", "", auth.AdminRole)

	res, err := srv.AcceptInvite(ctxWithIP("10.1.0.2"), &clientsapi.AcceptInviteRequest{Token: token, Password: "carolpass1"})
	if err != nil {
		t.Fatalf("AcceptInvite: unexpected error: %v", err)
	}

	if user := userManager.Find("carol"); user == nil || user.Role != auth.AdminRole {
		t.Fatalf("user %+v, want an admin", user)
	}
	claims, err := srv.jwt.VerifyAccessToken(res.AccessToken)
	if err != nil {
		t.Fatalf("VerifyAccessToken: %s", err)
	}
	if claims.Role != auth.AdminRole {
		t.Errorf("access token role = %s, want admin", claims.Role)
	}
}

func TestAcceptInvite_UsernameOverride(t *testing.T) {
	srv, userManager, _ := newTestAuthService(t, limits.Config{})
	token := createTestInvite(t, srv, "alice", "Alice Example", auth.UserRole)

	res, err := srv.AcceptInvite(ctxWithIP("10.1.0.3"), &clientsapi.AcceptInviteRequest{Token: token, Username: "alicia", Password: "aliciapass1"})
	if err != nil {
		t.Fatalf("AcceptInvite: unexpected error: %v", err)
	}

	user := userManager.Find("alicia")
	if user == nil {
		t.Fatal("the user was not created under the username the invitee chose")
	}
	if user.Role != auth.UserRole || user.Fullname != "Alice Example" {
		t.Errorf("user %+v does not carry the invite's role and fullname", user)
	}
	if userManager.Find("alice") != nil {
		t.Error("a user was also created under the username the admin chose")
	}
	claims, err := srv.jwt.VerifyAccessToken(res.AccessToken)
	if err != nil {
		t.Fatalf("VerifyAccessToken: %s", err)
	}
	if claims.Username != "alicia" {
		t.Errorf("access token is for %q, want alicia", claims.Username)
	}
}

func TestAcceptInvite_OverrideCannotTakeAnotherInvitesUsername(t *testing.T) {
	srv, userManager, _ := newTestAuthService(t, limits.Config{})
	aliceToken := createTestInvite(t, srv, "alice", "", auth.UserRole)
	bobToken := createTestInvite(t, srv, "bob", "", auth.UserRole)

	_, err := srv.AcceptInvite(ctxWithIP("10.1.0.4"), &clientsapi.AcceptInviteRequest{Token: aliceToken, Username: "bob", Password: "alicepass1"})
	wantCode(t, err, codes.AlreadyExists)
	if userManager.Find("bob") != nil {
		t.Fatal("alice's invite created bob's account")
	}

	// Both invites still work as offered.
	if _, err := srv.AcceptInvite(ctxWithIP("10.1.0.4"), &clientsapi.AcceptInviteRequest{Token: bobToken, Password: "bobpass12"}); err != nil {
		t.Fatalf("bob's invite: unexpected error: %v", err)
	}
	if _, err := srv.AcceptInvite(ctxWithIP("10.1.0.4"), &clientsapi.AcceptInviteRequest{Token: aliceToken, Password: "alicepass1"}); err != nil {
		t.Fatalf("alice's invite: unexpected error: %v", err)
	}
}

func TestAcceptInvite_UsedRevokedExpiredAndUnknownTokensAlike(t *testing.T) {
	srv, _, clock := newTestAuthService(t, limits.Config{LoginBurst: 50, LoginPerMin: 50 * 60})
	ctx := ctxWithIP("10.1.0.4")
	req := func(token string) *clientsapi.AcceptInviteRequest {
		return &clientsapi.AcceptInviteRequest{Token: token, Password: "samepass1", Username: "who"}
	}

	used := createTestInvite(t, srv, "used-name", "", auth.UserRole)
	if _, err := srv.AcceptInvite(ctx, &clientsapi.AcceptInviteRequest{Token: used, Password: "samepass1"}); err != nil {
		t.Fatalf("first AcceptInvite: unexpected error: %v", err)
	}

	revoked, revokedToken, err := srv.invites.Create("revoked-name", "", auth.UserRole, "admin")
	if err != nil {
		t.Fatalf("Create: %s", err)
	}
	if err := srv.invites.Revoke(revoked.ID); err != nil {
		t.Fatalf("Revoke: %s", err)
	}

	expired := createTestInvite(t, srv, "expired-name", "", auth.UserRole)
	clock.Advance(core.InviteLifetime)

	// Each is a different reason and must read the same, or the response
	// tells a token guesser which tokens were once real.
	messages := map[string]string{}
	for name, token := range map[string]string{
		"used":    used,
		"revoked": revokedToken,
		"expired": expired,
		"unknown": "not-a-real-token",
		"empty":   "",
	} {
		_, err := srv.AcceptInvite(ctx, req(token))
		messages[name] = wantCode(t, err, codes.NotFound).Message()
	}
	for name, msg := range messages {
		if msg != messages["used"] {
			t.Errorf("%s token: message %q differs from the used token's %q", name, msg, messages["used"])
		}
	}
}

func TestAcceptInvite_RevokedTokenIsNotFound(t *testing.T) {
	srv, userManager, _ := newTestAuthService(t, limits.Config{})
	invite, token, err := srv.invites.Create("alice", "", auth.UserRole, "admin")
	if err != nil {
		t.Fatalf("Create: %s", err)
	}
	if err := srv.invites.Revoke(invite.ID); err != nil {
		t.Fatalf("Revoke: %s", err)
	}

	_, err = srv.AcceptInvite(ctxWithIP("10.1.0.5"), &clientsapi.AcceptInviteRequest{Token: token, Password: "alicepass1"})
	wantCode(t, err, codes.NotFound)
	if userManager.Find("alice") != nil {
		t.Error("a user was created from a revoked invite")
	}
}

func TestAcceptInvite_ReusedTokenIsNotFound(t *testing.T) {
	srv, userManager, _ := newTestAuthService(t, limits.Config{})
	token := createTestInvite(t, srv, "alice", "", auth.UserRole)
	ctx := ctxWithIP("10.1.0.6")

	if _, err := srv.AcceptInvite(ctx, &clientsapi.AcceptInviteRequest{Token: token, Password: "alicepass1"}); err != nil {
		t.Fatalf("first AcceptInvite: unexpected error: %v", err)
	}

	// Same token, a different username: still one account only.
	_, err := srv.AcceptInvite(ctx, &clientsapi.AcceptInviteRequest{Token: token, Username: "mallory", Password: "mallorypass1"})
	wantCode(t, err, codes.NotFound)
	if userManager.Find("mallory") != nil {
		t.Error("a second user was created from a used invite")
	}
}

func TestAcceptInvite_ExpiredTokenIsNotFound(t *testing.T) {
	srv, userManager, clock := newTestAuthService(t, limits.Config{})
	token := createTestInvite(t, srv, "alice", "", auth.UserRole)

	clock.Advance(core.InviteLifetime)

	_, err := srv.AcceptInvite(ctxWithIP("10.1.0.7"), &clientsapi.AcceptInviteRequest{Token: token, Password: "alicepass1"})
	wantCode(t, err, codes.NotFound)
	if userManager.Find("alice") != nil {
		t.Error("a user was created from an expired invite")
	}
}

func TestAcceptInvite_TakenUsernameLeavesInviteUsable(t *testing.T) {
	srv, userManager, _ := newTestAuthService(t, limits.Config{})
	seedAdmin(t, userManager, "bob", "bobpass12")
	token := createTestInvite(t, srv, "alice", "Alice Example", auth.UserRole)
	ctx := ctxWithIP("10.1.0.8")

	// The invitee picks a name that is already someone's.
	_, err := srv.AcceptInvite(ctx, &clientsapi.AcceptInviteRequest{Token: token, Username: "bob", Password: "alicepass1"})
	wantCode(t, err, codes.AlreadyExists)
	if got := userManager.Find("bob"); got == nil || got.Role != auth.AdminRole || !got.IsCorrectPassword("bobpass12") {
		t.Fatalf("the existing user was disturbed: %+v", got)
	}
	if got := srv.invites.List(); len(got) != 1 {
		t.Fatalf("the invite is not pending after a failed attempt: %+v", got)
	}

	// They can try again with the same token.
	res, err := srv.AcceptInvite(ctx, &clientsapi.AcceptInviteRequest{Token: token, Username: "alicia", Password: "alicepass1"})
	if err != nil {
		t.Fatalf("retry AcceptInvite: unexpected error: %v", err)
	}
	if res.AccessToken == "" {
		t.Error("expected tokens from the retry")
	}
	if userManager.Find("alicia") == nil {
		t.Error("the retry did not create the user")
	}
}

func TestAcceptInvite_DefaultUsernameTakenByExistingUser(t *testing.T) {
	srv, userManager, _ := newTestAuthService(t, limits.Config{})
	token := createTestInvite(t, srv, "alice", "", auth.UserRole)

	// Someone took the name between the invite and its redemption.
	seedAdmin(t, userManager, "alice", "otherpass1")

	_, err := srv.AcceptInvite(ctxWithIP("10.1.0.9"), &clientsapi.AcceptInviteRequest{Token: token, Password: "alicepass1"})
	wantCode(t, err, codes.AlreadyExists)
	if got := userManager.Find("alice"); got == nil || !got.IsCorrectPassword("otherpass1") {
		t.Fatalf("the existing user was disturbed: %+v", got)
	}
}

func TestAcceptInvite_BadPasswordLeavesInviteUsable(t *testing.T) {
	srv, userManager, _ := newTestAuthService(t, limits.Config{})
	token := createTestInvite(t, srv, "alice", "", auth.UserRole)
	ctx := ctxWithIP("10.1.0.10")

	_, err := srv.AcceptInvite(ctx, &clientsapi.AcceptInviteRequest{Token: token, Password: "short"})
	wantCode(t, err, codes.InvalidArgument)
	if userManager.Find("alice") != nil {
		t.Error("a user was created with a rejected password")
	}

	if _, err := srv.AcceptInvite(ctx, &clientsapi.AcceptInviteRequest{Token: token, Password: "alicepass1"}); err != nil {
		t.Fatalf("retry AcceptInvite: unexpected error: %v", err)
	}
	if userManager.Find("alice") == nil {
		t.Error("the retry did not create the user")
	}
}

func TestAcceptInvite_IPLimitRejectsBeforeRedeeming(t *testing.T) {
	srv, userManager, _ := newTestAuthService(t, limits.Config{LoginBurst: 2, LoginPerMin: 2 * 60})
	token := createTestInvite(t, srv, "alice", "", auth.UserRole)
	limited := ctxWithIP("10.1.0.11")

	// Drain the source's bucket with requests that do not touch the invite.
	for i := 0; i < 2; i++ {
		if _, err := srv.AcceptInvite(limited, &clientsapi.AcceptInviteRequest{Token: "wrong", Password: "alicepass1"}); status.Code(err) != codes.NotFound {
			t.Fatalf("attempt %d: expected NotFound, got %v", i, err)
		}
	}

	// A valid token from the drained source is rejected by the gate, unspent.
	_, err := srv.AcceptInvite(limited, &clientsapi.AcceptInviteRequest{Token: token, Password: "alicepass1"})
	wantCode(t, err, codes.ResourceExhausted)
	if userManager.Find("alice") != nil {
		t.Error("a rate limited request created a user")
	}
	if got := srv.invites.List(); len(got) != 1 {
		t.Fatalf("a rate limited request spent the invite: %+v", got)
	}

	// Another source has its own bucket.
	if _, err := srv.AcceptInvite(ctxWithIP("10.1.0.12"), &clientsapi.AcceptInviteRequest{Token: token, Password: "alicepass1"}); err != nil {
		t.Fatalf("AcceptInvite from another source: unexpected error: %v", err)
	}
}

func TestAcceptInvite_FailedGuessesDrainTheSourceBudget(t *testing.T) {
	srv, _, _ := newTestAuthService(t, limits.Config{LoginBurst: 10, LoginPerMin: 60})
	ctx := ctxWithIP("10.1.0.13")

	// Each wrong token costs more than the single token an attempt spends, so
	// guessing runs out of budget well before the burst size.
	guesses := 0
	for ; guesses < 10; guesses++ {
		_, err := srv.AcceptInvite(ctx, &clientsapi.AcceptInviteRequest{Token: "guess", Password: "whatever1"})
		if status.Code(err) == codes.ResourceExhausted {
			break
		}
		wantCode(t, err, codes.NotFound)
	}
	if guesses == 0 || guesses >= 10 {
		t.Fatalf("rate limit hit after %d wrong tokens, want fewer than the burst of 10 but at least 1", guesses)
	}
}

func TestAcceptInvite_FailedGuessesDoNotLockOutAccounts(t *testing.T) {
	srv, userManager, _ := newTestAuthService(t, limits.Config{LoginBurst: 50, LoginPerMin: 50 * 60})
	seedAdmin(t, userManager, "admin", "adminpass1")
	ctx := ctxWithIP("10.1.0.14")

	// A stranger guessing tokens "as" the admin must not be able to put the
	// admin's account into backoff.
	for i := 0; i < 6; i++ {
		_, err := srv.AcceptInvite(ctx, &clientsapi.AcceptInviteRequest{Token: "guess", Username: "admin", Password: "whatever1"})
		wantCode(t, err, codes.NotFound)
	}

	if wait := srv.limits.AccountRetryIn("admin"); wait != 0 {
		t.Fatalf("account backoff of %s after invite token guesses", wait)
	}
	if _, err := srv.Login(ctxWithIP("10.1.0.15"), &clientsapi.UserLoginRequest{Username: "admin", Password: "adminpass1"}); err != nil {
		t.Fatalf("the admin cannot log in after invite token guesses: %v", err)
	}
}

// claimsCtx returns a context carrying the claims of an admin or user, as the
// auth interceptor attaches them.
func claimsCtx(username string, role auth.Role) context.Context {
	return context.WithValue(context.Background(), "claims", &AccessTokenClaims{Username: username, Role: role})
}

func newInviteTestUserService(t *testing.T) (*UserService, *core.UserManager) {
	t.Helper()
	store := stores.NewMemStore()

	userManager, err := core.NewUserManager(store)
	if err != nil {
		t.Fatalf("failed to create user manager: %s", err)
	}
	t.Cleanup(userManager.Close)

	inviteManager, err := core.NewInviteManager(store, userManager)
	if err != nil {
		t.Fatalf("failed to create invite manager: %s", err)
	}
	t.Cleanup(inviteManager.Close)

	return &UserService{userManager: userManager, inviteManager: inviteManager}, userManager
}

func TestCreateInvite_ReturnsInviteAndToken(t *testing.T) {
	service, _ := newInviteTestUserService(t)

	res, err := service.CreateInvite(claimsCtx("carol", auth.AdminRole), &clientsapi.CreateInviteRequest{
		Username: "alice",
		Fullname: "Alice Example",
		Role:     clientsapi.UserRole_USER_ROLE_USER,
	})
	if err != nil {
		t.Fatalf("CreateInvite: unexpected error: %v", err)
	}

	if res.Token == "" {
		t.Error("no token in the response")
	}
	invite := res.Invite
	if invite == nil || invite.Id == "" {
		t.Fatalf("no invite in the response: %+v", res)
	}
	if invite.Username != "alice" || invite.Fullname != "Alice Example" || invite.Role != clientsapi.UserRole_USER_ROLE_USER {
		t.Errorf("invite %+v does not carry the request", invite)
	}
	// The admin who made the call, not anything from the request.
	if invite.CreatedBy != "carol" {
		t.Errorf("CreatedBy = %q, want carol", invite.CreatedBy)
	}
	if got, want := invite.ExpiresAt-invite.CreatedAt, (24 * time.Hour).Milliseconds(); got != want {
		t.Errorf("expires %d ms after creation, want %d", got, want)
	}
	if age := time.Since(time.UnixMilli(invite.CreatedAt)); age < 0 || age > time.Minute {
		t.Errorf("CreatedAt is %s ago, expected Unix milliseconds for now", age)
	}
}

func TestCreateInvite_Validation(t *testing.T) {
	service, userManager := newInviteTestUserService(t)
	seedAdmin(t, userManager, "bob", "bobpass12")
	admin := claimsCtx("carol", auth.AdminRole)

	if _, err := service.CreateInvite(admin, &clientsapi.CreateInviteRequest{Username: "alice", Role: clientsapi.UserRole_USER_ROLE_USER}); err != nil {
		t.Fatalf("CreateInvite: unexpected error: %v", err)
	}

	tests := []struct {
		name string
		req  *clientsapi.CreateInviteRequest
		want codes.Code
	}{
		{"no username", &clientsapi.CreateInviteRequest{Role: clientsapi.UserRole_USER_ROLE_USER}, codes.InvalidArgument},
		{"no role", &clientsapi.CreateInviteRequest{Username: "dave"}, codes.InvalidArgument},
		{"undefined role", &clientsapi.CreateInviteRequest{Username: "dave", Role: clientsapi.UserRole_USER_ROLE_UNDEFINED}, codes.InvalidArgument},
		{"existing user", &clientsapi.CreateInviteRequest{Username: "bob", Role: clientsapi.UserRole_USER_ROLE_USER}, codes.AlreadyExists},
		{"pending invite", &clientsapi.CreateInviteRequest{Username: "alice", Role: clientsapi.UserRole_USER_ROLE_ADMIN}, codes.AlreadyExists},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateInvite(admin, tt.req)
			wantCode(t, err, tt.want)
		})
	}

	if got, err := service.ListInvites(admin, &clientsapi.ListInvitesRequest{}); err != nil || len(got.Invites) != 1 {
		t.Fatalf("rejected requests changed the pending invites: %+v, %v", got, err)
	}
}

func TestInviteRPCs_AdminOnly(t *testing.T) {
	service, _ := newInviteTestUserService(t)
	user := claimsCtx("alice", auth.UserRole)

	_, err := service.CreateInvite(user, &clientsapi.CreateInviteRequest{Username: "dave", Role: clientsapi.UserRole_USER_ROLE_USER})
	wantCode(t, err, codes.PermissionDenied)
	_, err = service.ListInvites(user, &clientsapi.ListInvitesRequest{})
	wantCode(t, err, codes.PermissionDenied)
	_, err = service.RevokeInvite(user, &clientsapi.RevokeInviteRequest{Id: "anything"})
	wantCode(t, err, codes.PermissionDenied)
}

func TestListInvites_OldestFirstWithoutTokens(t *testing.T) {
	service, _ := newInviteTestUserService(t)
	admin := claimsCtx("carol", auth.AdminRole)

	empty, err := service.ListInvites(admin, &clientsapi.ListInvitesRequest{})
	if err != nil || len(empty.Invites) != 0 {
		t.Fatalf("ListInvites with none pending: %+v, %v", empty, err)
	}

	created := map[string]string{}
	for _, name := range []string{"alice", "bob", "dave"} {
		res, err := service.CreateInvite(admin, &clientsapi.CreateInviteRequest{Username: name, Role: clientsapi.UserRole_USER_ROLE_USER})
		if err != nil {
			t.Fatalf("CreateInvite(%q): %v", name, err)
		}
		created[name] = res.Invite.Id
		// Creation times are milliseconds; keep the order unambiguous.
		time.Sleep(2 * time.Millisecond)
	}

	res, err := service.ListInvites(admin, &clientsapi.ListInvitesRequest{})
	if err != nil {
		t.Fatalf("ListInvites: %v", err)
	}
	if len(res.Invites) != 3 {
		t.Fatalf("got %d invites, want 3: %+v", len(res.Invites), res.Invites)
	}
	for i, name := range []string{"alice", "bob", "dave"} {
		if got := res.Invites[i]; got.Username != name || got.Id != created[name] || got.CreatedBy != "carol" {
			t.Errorf("invite %d = %+v, want %s (id %s)", i, got, name, created[name])
		}
	}
}

func TestRevokeInvite(t *testing.T) {
	service, _ := newInviteTestUserService(t)
	admin := claimsCtx("carol", auth.AdminRole)

	res, err := service.CreateInvite(admin, &clientsapi.CreateInviteRequest{Username: "alice", Role: clientsapi.UserRole_USER_ROLE_USER})
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	if _, err := service.RevokeInvite(admin, &clientsapi.RevokeInviteRequest{Id: res.Invite.Id}); err != nil {
		t.Fatalf("RevokeInvite: %v", err)
	}
	if got, _ := service.ListInvites(admin, &clientsapi.ListInvitesRequest{}); len(got.Invites) != 0 {
		t.Fatalf("a revoked invite is still listed: %+v", got.Invites)
	}

	// Unknown, and already revoked: both succeed.
	for _, id := range []string{res.Invite.Id, "no-such-invite", ""} {
		if _, err := service.RevokeInvite(admin, &clientsapi.RevokeInviteRequest{Id: id}); err != nil {
			t.Errorf("RevokeInvite(%q): unexpected error: %v", id, err)
		}
	}

	// The username is free again.
	if _, err := service.CreateInvite(admin, &clientsapi.CreateInviteRequest{Username: "alice", Role: clientsapi.UserRole_USER_ROLE_USER}); err != nil {
		t.Fatalf("CreateInvite after revoke: %v", err)
	}
}
