package core

import (
	"fmt"
	"strings"
	"time"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
)

// Level is how loudly a Notification wants to be presented. Clients map it onto
// their own vocabulary: a toast variant on the web, nothing in particular on
// iOS yet.
type Level int

const (
	UnspecifiedLevel Level = iota
	InfoLevel
	SuccessLevel
	WarningLevel
	ErrorLevel
)

func (l Level) String() string {
	switch l {
	case UnspecifiedLevel:
		return "unspecified"
	case InfoLevel:
		return "info"
	case SuccessLevel:
		return "success"
	case WarningLevel:
		return "warning"
	case ErrorLevel:
		return "error"
	}
	return "<UNIMPLEMENTED>"
}

func (l Level) MarshalJSON() ([]byte, error) {
	switch l {
	case UnspecifiedLevel, InfoLevel, SuccessLevel, WarningLevel, ErrorLevel:
		return []byte(`"` + l.String() + `"`), nil
	}
	return nil, fmt.Errorf("unimplemented")
}

func (l *Level) UnmarshalJSON(p []byte) error {
	switch string(p) {
	case `"unspecified"`:
		*l = UnspecifiedLevel
	case `"info"`:
		*l = InfoLevel
	case `"success"`:
		*l = SuccessLevel
	case `"warning"`:
		*l = WarningLevel
	case `"error"`:
		*l = ErrorLevel
	default:
		return fmt.Errorf("unknown")
	}
	return nil
}

func (l Level) Pb() clientsapi.NotificationLevel {
	switch l {
	case InfoLevel:
		return clientsapi.NotificationLevel_NOTIFICATION_LEVEL_INFO
	case SuccessLevel:
		return clientsapi.NotificationLevel_NOTIFICATION_LEVEL_SUCCESS
	case WarningLevel:
		return clientsapi.NotificationLevel_NOTIFICATION_LEVEL_WARNING
	case ErrorLevel:
		return clientsapi.NotificationLevel_NOTIFICATION_LEVEL_ERROR
	}
	return clientsapi.NotificationLevel_NOTIFICATION_LEVEL_UNSPECIFIED
}

func LevelFromPb(pb clientsapi.NotificationLevel) Level {
	switch pb {
	case clientsapi.NotificationLevel_NOTIFICATION_LEVEL_INFO:
		return InfoLevel
	case clientsapi.NotificationLevel_NOTIFICATION_LEVEL_SUCCESS:
		return SuccessLevel
	case clientsapi.NotificationLevel_NOTIFICATION_LEVEL_WARNING:
		return WarningLevel
	case clientsapi.NotificationLevel_NOTIFICATION_LEVEL_ERROR:
		return ErrorLevel
	}
	return UnspecifiedLevel
}

// AudienceKind selects which of an Audience's other fields is meaningful.
type AudienceKind int

const (
	// UnspecifiedAudience matches nobody. It is the zero value on purpose: a
	// record that fails to decode, or one written by a future version with a
	// kind this build does not know, reaches no one rather than everyone.
	UnspecifiedAudience AudienceKind = iota
	AllAudience
	RoleAudience
	UserAudience
)

func (k AudienceKind) String() string {
	switch k {
	case UnspecifiedAudience:
		return "unspecified"
	case AllAudience:
		return "all"
	case RoleAudience:
		return "role"
	case UserAudience:
		return "user"
	}
	return "<UNIMPLEMENTED>"
}

func (k AudienceKind) MarshalJSON() ([]byte, error) {
	switch k {
	case UnspecifiedAudience, AllAudience, RoleAudience, UserAudience:
		return []byte(`"` + k.String() + `"`), nil
	}
	return nil, fmt.Errorf("unimplemented")
}

func (k *AudienceKind) UnmarshalJSON(p []byte) error {
	switch string(p) {
	case `"unspecified"`:
		*k = UnspecifiedAudience
	case `"all"`:
		*k = AllAudience
	case `"role"`:
		*k = RoleAudience
	case `"user"`:
		*k = UserAudience
	default:
		return fmt.Errorf("unknown")
	}
	return nil
}

// Audience is who a Notification is for.
type Audience struct {
	Kind     AudienceKind `json:"kind"`
	Role     auth.Role    `json:"role,omitempty"`     // Only read when Kind is RoleAudience.
	Username string       `json:"username,omitempty"` // Only read when Kind is UserAudience.
}

func AudienceAll() Audience { return Audience{Kind: AllAudience} }

func AudienceRole(role auth.Role) Audience {
	return Audience{Kind: RoleAudience, Role: role}
}

func AudienceUser(username string) Audience {
	return Audience{Kind: UserAudience, Username: username}
}

// Matches reports whether a user is in the audience. It is the single authority
// on that question: the stream handler and every future Deliverer must call
// this rather than re-deriving the rule, so there is one place to audit and one
// place to change when a new kind is added.
//
// The role must come from the users store, not from an access token. Role is
// snapshotted into the token and only refreshed when the token is, so a demoted
// admin can hold an admin token for minutes.
func (a Audience) Matches(username string, role auth.Role) bool {
	switch a.Kind {
	case AllAudience:
		return true
	case RoleAudience:
		return role == a.Role
	case UserAudience:
		return username == a.Username
	}
	return false
}

func (a Audience) Valid() bool {
	switch a.Kind {
	case AllAudience:
		return true
	case RoleAudience:
		return a.Role == auth.AdminRole || a.Role == auth.UserRole
	case UserAudience:
		return a.Username != ""
	}
	return false
}

func (a Audience) String() string {
	switch a.Kind {
	case AllAudience:
		return "all"
	case RoleAudience:
		return "role:" + a.Role.String()
	case UserAudience:
		return "user:" + a.Username
	}
	return "unspecified"
}

func (a Audience) Pb() *clientsapi.NotificationAudience {
	pb := &clientsapi.NotificationAudience{Kind: clientsapi.NotificationAudience_KIND_UNSPECIFIED}
	switch a.Kind {
	case AllAudience:
		pb.Kind = clientsapi.NotificationAudience_KIND_ALL
	case RoleAudience:
		pb.Kind = clientsapi.NotificationAudience_KIND_ROLE
		pb.Role = a.Role.Pb()
	case UserAudience:
		pb.Kind = clientsapi.NotificationAudience_KIND_USER
		pb.Username = a.Username
	}
	return pb
}

func AudienceFromPb(pb *clientsapi.NotificationAudience) Audience {
	if pb == nil {
		return Audience{}
	}
	switch pb.GetKind() {
	case clientsapi.NotificationAudience_KIND_ALL:
		return AudienceAll()
	case clientsapi.NotificationAudience_KIND_ROLE:
		return AudienceRole(auth.RoleFromPb(pb.GetRole()))
	case clientsapi.NotificationAudience_KIND_USER:
		return AudienceUser(pb.GetUsername())
	}
	return Audience{}
}

// UserState is one user's relationship to one Notification. An absent entry
// means unread and not dismissed, so a user created after a notification was
// sent still sees it as new.
type UserState struct {
	Read      *time.Time `json:"read,omitempty"`
	Dismissed *time.Time `json:"dismissed,omitempty"`
}

func (s *UserState) Clone() *UserState {
	if s == nil {
		return nil
	}
	out := &UserState{}
	if s.Read != nil {
		read := *s.Read
		out.Read = &read
	}
	if s.Dismissed != nil {
		dismissed := *s.Dismissed
		out.Dismissed = &dismissed
	}
	return out
}

// Notification is one message addressed to some subset of users. Read and
// dismissed state is per-user and lives inside the record, so trimming a
// notification takes its state with it and there is nothing left to orphan.
type Notification struct {
	ID      string    `json:"id"`
	Created time.Time `json:"created"`
	Level   Level     `json:"level"`
	Title   string    `json:"title"`
	Body    string    `json:"body,omitempty"`

	// Source is an opaque origin key, e.g. "settings-test". For grouping and
	// filtering, never for display.
	Source string `json:"source,omitempty"`
	// Category is a coarse bucket for future per-user preferences.
	Category string `json:"category,omitempty"`
	// CollapseKey, when non-empty, replaces the previous notification carrying
	// the same key, so a flapping sensor cannot accumulate a hundred rows.
	CollapseKey string `json:"collapse_key,omitempty"`
	// Link is client-relative, e.g. "/devices/<id>". Never absolute.
	Link string `json:"link,omitempty"`

	Audience Audience `json:"audience"`

	// State is keyed by username. Users are keyed by username throughout
	// (see UserManager) and cannot be renamed today - there is no rename RPC
	// and no UserManager method for it - so a username is a stable identity.
	// If renaming is ever added, these keys have to be migrated with it.
	State map[string]*UserState `json:"state,omitempty"`
}

func NewNotification(id, title, body string, level Level, audience Audience) *Notification {
	return &Notification{
		ID:       id,
		Created:  time.Now().UTC(),
		Level:    level,
		Title:    title,
		Body:     body,
		Audience: audience,
	}
}

func (n *Notification) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "id:%q, level:%s, audience:%s, title:%q", n.ID, n.Level, n.Audience, n.Title)
	if n.Source != "" {
		fmt.Fprintf(&b, ", source:%q", n.Source)
	}
	return b.String()
}

// Clone returns a copy safe to hand to listeners while the manager keeps
// mutating the original.
func (n *Notification) Clone() *Notification {
	out := *n
	out.State = make(map[string]*UserState, len(n.State))
	for username, state := range n.State {
		out.State[username] = state.Clone()
	}
	return &out
}

func (n *Notification) stateFor(username string) *UserState {
	if n.State == nil {
		return nil
	}
	return n.State[username]
}

func (n *Notification) IsReadBy(username string) bool {
	state := n.stateFor(username)
	return state != nil && state.Read != nil
}

func (n *Notification) IsDismissedBy(username string) bool {
	state := n.stateFor(username)
	return state != nil && state.Dismissed != nil
}

// Pb projects the notification for one user. Read is that user's alone, which
// is why the username is required rather than optional: there is no correct
// audience-independent value for it.
func (n *Notification) Pb(username string) *clientsapi.Notification {
	return &clientsapi.Notification{
		Id:          n.ID,
		CreatedAt:   n.Created.UnixMilli(),
		Level:       n.Level.Pb(),
		Title:       n.Title,
		Body:        n.Body,
		Source:      n.Source,
		Category:    n.Category,
		CollapseKey: n.CollapseKey,
		Audience:    n.Audience.Pb(),
		Read:        n.IsReadBy(username),
		Link:        n.Link,
	}
}

// visibleTo reports whether the notification should appear in a user's inbox:
// they are in its audience and have not dismissed it.
func (n *Notification) visibleTo(username string, role auth.Role) bool {
	return n.Audience.Matches(username, role) && !n.IsDismissedBy(username)
}

// pruneUser drops one user's state, reporting whether anything changed.
func (n *Notification) pruneUser(username string) bool {
	if n.stateFor(username) == nil {
		return false
	}
	delete(n.State, username)
	return true
}

// ensureState returns the user's state, creating it if absent.
func (n *Notification) ensureState(username string) *UserState {
	if n.State == nil {
		n.State = make(map[string]*UserState, 1)
	}
	if state := n.State[username]; state != nil {
		return state
	}
	state := &UserState{}
	n.State[username] = state
	return state
}
