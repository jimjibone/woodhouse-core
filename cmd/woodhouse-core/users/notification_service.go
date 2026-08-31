package users

import (
	"context"
	"errors"
	"strings"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/core"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/shared/random"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxNotificationTitleLength = 200
	maxNotificationBodyLength  = 2000

	// testNotificationSource marks notifications raised by SendTestNotification,
	// so a later cleanup or filter can tell them from real ones.
	testNotificationSource = "settings-test"
)

// resolveNotificationUser looks the caller up in the users store and returns
// their current username and role.
//
// It deliberately does not trust claims.Role. Role is snapshotted into the
// access token and only refreshed when the token is, so a demoted admin keeps
// an admin token for up to the access-token lifetime. That is acceptable for
// "may I call this RPC" - the interceptor's policy check is coarse and the
// window is short - but not for "which role-targeted notifications am I in",
// where being wrong means showing one user another group's messages.
//
// Looking the user up per update also closes a second hole: the interceptor
// only checks that the account still exists when a call starts, so an already
// open stream outlives a RemoveUser.
func (service *UserService) resolveNotificationUser(claims *AccessTokenClaims) (*core.User, error) {
	user := service.userManager.Find(claims.Username)
	if user == nil {
		return nil, status.Error(codes.Unauthenticated, "user no longer exists")
	}
	return user, nil
}

func (service *UserService) NotificationsStream(req *clientsapi.NotificationsStreamRequest, server clientsapi.UserService_NotificationsStreamServer) error {
	claims := server.Context().Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return status.Errorf(codes.PermissionDenied, "no claims in request")
	}

	service.log.Debugf("notification stream started for %q", claims.Username)
	defer service.log.Debugf("notification stream finished for %q", claims.Username)

	revocations := service.userJwt.SubscribeRevocations()
	defer revocations.Close()

	// Subscribe first, then snapshot. An update landing between the two is then
	// delivered twice rather than not at all, and the client upserts by ID, so
	// a duplicate converges where a miss would leave a permanent gap.
	lis := service.notificationManager.GetListener()
	defer lis.Close()

	user, err := service.resolveNotificationUser(claims)
	if err != nil {
		return err
	}

	heartbeat := func() *clientsapi.NotificationsStreamResponse {
		return &clientsapi.NotificationsStreamResponse{
			Update: &clientsapi.NotificationsStreamResponse_Heartbeat{Heartbeat: &clientsapi.Heartbeat{}},
		}
	}

	// The initial batch, oldest first so a client that appends as it reads ends
	// up in the same order as one that sorts.
	snapshot := service.notificationManager.Snapshot(user.Username, user.Role)
	for i := len(snapshot) - 1; i >= 0; i-- {
		msg := &clientsapi.NotificationsStreamResponse{
			Update: &clientsapi.NotificationsStreamResponse_NotificationUpdate{
				NotificationUpdate: snapshot[i].Pb(user.Username),
			},
		}
		if err := server.Send(msg); err != nil {
			service.log.Errorf("failed to send notification snapshot: %s", err)
			return status.Errorf(codes.Internal, "failed to send snapshot")
		}
	}

	// The first heartbeat ends the initial batch. Clients use it to prune
	// anything they were holding that the server no longer has.
	if err := server.Send(heartbeat()); err != nil {
		service.log.Errorf("failed to send notification snapshot end: %s", err)
		return status.Errorf(codes.Internal, "failed to send snapshot end")
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-server.Context().Done():
			return status.Errorf(codes.Canceled, "context canceled")

		case revoked := <-revocations.Sub():
			if revoked == claims.RefreshUUID {
				return status.Error(codes.Unauthenticated, "session revoked")
			}

		case <-ticker.C:
			// Keepalive so the client can spot a dead stream.
			if err := server.Send(heartbeat()); err != nil {
				service.log.Errorf("failed to send notification stream keepalive: %s", err)
				return status.Errorf(codes.Internal, "failed to send keepalive")
			}

		case update := <-lis.Sub():
			// The manager broadcasts every notification to every listener,
			// because only this handler knows who is on the other end. Two
			// filters turn that firehose into one user's inbox.

			// A per-user fact - a read mark, a dismissal - is nobody else's.
			if update.ForUser != "" && update.ForUser != claims.Username {
				continue
			}

			// Re-resolve on every update rather than caching: a role change or
			// an account deletion mid-stream has to take effect now, not at the
			// next reconnect.
			user, err := service.resolveNotificationUser(claims)
			if err != nil {
				return err
			}

			msg := &clientsapi.NotificationsStreamResponse{}
			switch {
			case update.Updated != nil:
				if !update.Updated.Audience.Matches(user.Username, user.Role) ||
					update.Updated.IsDismissedBy(user.Username) {
					continue
				}
				msg.Update = &clientsapi.NotificationsStreamResponse_NotificationUpdate{
					NotificationUpdate: update.Updated.Pb(user.Username),
				}
			case update.Removed != nil:
				// Removals go out unfiltered. A bare opaque ID leaks nothing,
				// and a client that never held it simply no-ops.
				msg.Update = &clientsapi.NotificationsStreamResponse_RemovedId{RemovedId: *update.Removed}
			default:
				msg = heartbeat()
			}

			if err := server.Send(msg); err != nil {
				service.log.Errorf("failed to send notification stream update: %s", err)
				return status.Errorf(codes.Internal, "failed to send update")
			}
		}
	}
}

func (service *UserService) MarkNotificationRead(ctx context.Context, req *clientsapi.MarkNotificationReadRequest) (*clientsapi.MarkNotificationReadResponse, error) {
	claims := ctx.Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return nil, status.Errorf(codes.PermissionDenied, "no claims in request")
	}
	if req.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "id not defined")
	}

	err := service.notificationManager.MarkRead(req.GetId(), claims.Username)
	if err != nil {
		return nil, notificationError("failed to mark notification read", err)
	}

	return &clientsapi.MarkNotificationReadResponse{}, nil
}

func (service *UserService) MarkAllNotificationsRead(ctx context.Context, req *clientsapi.MarkAllNotificationsReadRequest) (*clientsapi.MarkAllNotificationsReadResponse, error) {
	claims := ctx.Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return nil, status.Errorf(codes.PermissionDenied, "no claims in request")
	}

	user, err := service.resolveNotificationUser(claims)
	if err != nil {
		return nil, err
	}

	_, err = service.notificationManager.MarkAllRead(user.Username, user.Role)
	if err != nil {
		return nil, notificationError("failed to mark notifications read", err)
	}

	return &clientsapi.MarkAllNotificationsReadResponse{}, nil
}

func (service *UserService) DismissNotification(ctx context.Context, req *clientsapi.DismissNotificationRequest) (*clientsapi.DismissNotificationResponse, error) {
	claims := ctx.Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return nil, status.Errorf(codes.PermissionDenied, "no claims in request")
	}
	if req.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "id not defined")
	}

	err := service.notificationManager.Dismiss(req.GetId(), claims.Username)
	if err != nil {
		return nil, notificationError("failed to dismiss notification", err)
	}

	return &clientsapi.DismissNotificationResponse{}, nil
}

func (service *UserService) SendTestNotification(ctx context.Context, req *clientsapi.SendTestNotificationRequest) (*clientsapi.SendTestNotificationResponse, error) {
	claims := ctx.Value("claims").(*AccessTokenClaims)
	if claims == nil {
		return nil, status.Errorf(codes.PermissionDenied, "no claims in request")
	}
	if claims.Role != auth.AdminRole {
		return nil, status.Errorf(codes.PermissionDenied, "not allowed to send notifications")
	}

	title := strings.TrimSpace(req.GetTitle())
	if title == "" {
		title = "Test notification"
	}
	body := strings.TrimSpace(req.GetBody())
	if req.Body == nil {
		body = "If you can see this, notification delivery is working."
	}
	if err := validNotificationText(title, body); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", err)
	}

	level := core.LevelFromPb(req.GetLevel())
	if level == core.UnspecifiedLevel {
		level = core.InfoLevel
	}

	// Default to the caller alone. Pressing "send test" on a shared instance
	// should not notify the whole household by accident.
	audience := core.AudienceUser(claims.Username)
	if req.GetAudience() != nil && req.GetAudience().GetKind() != clientsapi.NotificationAudience_KIND_UNSPECIFIED {
		audience = core.AudienceFromPb(req.GetAudience())
	}
	if err := service.validNotificationAudience(audience); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", err)
	}

	// Generate a unique ID for the notification.
	notificationID, err := random.GenerateRandomString(10)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate notification id: %s", err)
	}

	notification := core.NewNotification(notificationID, title, body, level, audience)
	notification.Source = testNotificationSource
	notification.Category = "system"

	if err := service.notificationManager.Add(notification); err != nil {
		return nil, notificationError("failed to send notification", err)
	}

	service.log.Infof("user %q sent a test notification to %s", claims.Username, audience)

	return &clientsapi.SendTestNotificationResponse{
		Notification: notification.Pb(claims.Username),
	}, nil
}

// validNotificationText bounds the free-text fields. Notifications are stored
// and replayed to every client on connect, so an unbounded title is a way to
// bloat every reconnect, not just one record.
func validNotificationText(title, body string) error {
	if len(title) > maxNotificationTitleLength {
		return errors.New("title too long")
	}
	if len(body) > maxNotificationBodyLength {
		return errors.New("body too long")
	}
	return nil
}

// validNotificationAudience rejects audiences nobody could ever be in, so a
// typo surfaces as an error rather than as silence.
func (service *UserService) validNotificationAudience(audience core.Audience) error {
	if !audience.Valid() {
		return errors.New("invalid audience")
	}
	if audience.Kind == core.UserAudience && service.userManager.Find(audience.Username) == nil {
		return errors.New("user not found")
	}
	return nil
}

// notificationError maps a NotificationManager error onto a gRPC status, in the
// same spirit as zoneError: the webui shows these verbatim, so the code decides
// whether the user sees a retryable failure or a correctable one.
func notificationError(what string, err error) error {
	if errors.Is(err, core.ErrNotificationNotFound) {
		return status.Errorf(codes.NotFound, "%s: %s", what, err)
	}
	return status.Errorf(codes.InvalidArgument, "%s: %s", what, err)
}
