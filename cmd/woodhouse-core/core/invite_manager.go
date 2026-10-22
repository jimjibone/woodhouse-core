package core

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/jimjibone/log"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/shared/random"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

// InviteLifetime is how long an invite can be redeemed for after it is created.
const InviteLifetime = 24 * time.Hour

// inviteTokenBytes is the entropy of an invite token. The token is the only
// thing authorizing an unauthenticated AcceptInvite, so it is as long as a
// session secret rather than something a person could be asked to type.
const inviteTokenBytes = 32

var ErrInviteNotFound = errors.New("invite not found")

// InviteManager owns the pending invites: offers of an account that an admin
// has made and nobody has accepted yet. An invite lives until it is redeemed,
// revoked or expired, and only a hash of its token is kept.
//
// Unlike the other managers it saves on every change instead of on a timer.
// Losing a revoke or a redemption to a crash would bring a dead token back to
// life, which the 60s window the others accept is too long for.
type InviteManager struct {
	log   *log.Context
	store stores.Store
	users *UserManager
	now   func() time.Time

	mu      sync.Mutex
	changed bool
	invites map[string]*Invite // key=id
	// redeeming holds the invites Redeem has taken out and not yet settled,
	// so a revoke that lands in between is not undone by the put-back, and
	// their usernames stay reserved.
	redeeming map[string]*Invite // key=id
}

func NewInviteManager(store stores.Store, users *UserManager) (*InviteManager, error) {
	return NewInviteManagerWithClock(store, users, time.Now)
}

// NewInviteManagerWithClock is NewInviteManager with its own clock, so tests
// can move through an invite's lifetime without waiting for it.
func NewInviteManagerWithClock(store stores.Store, users *UserManager, now func() time.Time) (*InviteManager, error) {
	manager := &InviteManager{
		log:       log.NewContext(log.DefaultLogger, "invite-manager", log.DebugLevel),
		store:     store,
		users:     users,
		now:       now,
		invites:   make(map[string]*Invite),
		redeeming: make(map[string]*Invite),
	}

	manager.mu.Lock()
	defer manager.mu.Unlock()

	// Load the previous state.
	err := manager.load()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %s", err)
	}

	// Save the state if changed. load() prunes, so this persists the prune.
	err = manager.saveIfChanged()
	if err != nil {
		return nil, fmt.Errorf("failed to save state: %s", err)
	}

	return manager, nil
}

func (manager *InviteManager) Close() {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	err := manager.saveIfChanged()
	if err != nil {
		manager.log.Errorf("failed to save state: %s", err)
	}
}

// Create makes an invite for username and returns it with its token. The token
// exists only in the return value: it cannot be recovered afterwards.
//
// It fails with ErrUsernameTooShort for a username NewUser would refuse, and
// with ErrAlreadyExists if the username belongs to a user or to another
// pending invite.
func (manager *InviteManager) Create(username, fullname string, role auth.Role, createdBy string) (*Invite, string, error) {
	if err := ValidateUsername(username); err != nil {
		return nil, "", err
	}

	manager.mu.Lock()
	defer manager.mu.Unlock()

	manager.prune()

	// The check and the insert share the lock so two concurrent invites for
	// one username cannot both get in. A user created in the meantime is
	// caught again when the invite is redeemed.
	if manager.users.Find(username) != nil || manager.invited(username) {
		return nil, "", ErrAlreadyExists
	}

	id, err := manager.newID()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate invite id: %w", err)
	}
	token, err := newInviteToken()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate invite token: %w", err)
	}

	now := manager.now().UTC()
	invite := &Invite{
		ID:        id,
		TokenHash: hashInviteToken(token),
		Username:  username,
		Fullname:  fullname,
		Role:      role,
		CreatedBy: createdBy,
		CreatedAt: now,
		ExpiresAt: now.Add(InviteLifetime),
	}
	manager.invites[id] = invite
	manager.changed = true

	// An invite that was not persisted would vanish on restart, leaving the
	// admin holding a token that no longer works, so don't hand it out.
	err = manager.saveIfChanged()
	if err != nil {
		delete(manager.invites, id)
		return nil, "", fmt.Errorf("failed to save invites: %w", err)
	}

	manager.log.Infof("invite %q created for %q as %s by %q", id, username, role, createdBy)

	return invite.Clone(), token, nil
}

// List returns the pending invites, oldest first.
func (manager *InviteManager) List() []*Invite {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	manager.prune()
	manager.saveLogged()

	invites := make([]*Invite, 0, len(manager.invites))
	for _, invite := range manager.invites {
		invites = append(invites, invite.Clone())
	}
	sort.Slice(invites, func(i, j int) bool {
		if invites[i].CreatedAt.Equal(invites[j].CreatedAt) {
			return invites[i].ID < invites[j].ID
		}
		return invites[i].CreatedAt.Before(invites[j].CreatedAt)
	})
	return invites
}

// Revoke cancels a pending invite. An unknown id is not an error: the caller
// wanted the invite gone, and it is.
//
// The error is only ever a failure to save. The invite is revoked in memory
// regardless, and the next successful save makes that durable.
func (manager *InviteManager) Revoke(id string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	manager.prune()

	if invite := manager.invites[id]; invite != nil {
		delete(manager.invites, id)
		manager.changed = true
		manager.log.Infof("invite %q for %q revoked", id, invite.Username)
	}
	// If a redemption is in flight, make sure a failure does not put the
	// invite back.
	delete(manager.redeeming, id)

	err := manager.saveIfChanged()
	if err != nil {
		return fmt.Errorf("failed to save invites: %w", err)
	}
	return nil
}

// Redeem consumes the invite that token belongs to and calls accept with it,
// which should create the account. It returns ErrInviteNotFound for an
// unknown, used, revoked or expired token, all alike.
//
// The invite is taken out before accept runs, so two concurrent redemptions of
// one token cannot both reach accept. If accept fails the invite is put back
// and its error returned, so the invitee can retry (a taken username, a
// password that was too short) until the invite expires.
func (manager *InviteManager) Redeem(token string, accept func(invite *Invite) error) error {
	invite, err := manager.take(token)
	if err != nil {
		return err
	}

	err = accept(invite)
	manager.settle(invite, err == nil)
	return err
}

// take removes the invite for token and parks it in redeeming.
func (manager *InviteManager) take(token string) (*Invite, error) {
	hash := hashInviteToken(token)

	manager.mu.Lock()
	defer manager.mu.Unlock()

	manager.prune()
	defer manager.saveLogged()

	for id, invite := range manager.invites {
		if subtle.ConstantTimeCompare(invite.TokenHash, hash) != 1 {
			continue
		}
		delete(manager.invites, id)
		manager.redeeming[id] = invite
		manager.changed = true
		return invite.Clone(), nil
	}
	return nil, ErrInviteNotFound
}

// settle ends a redemption. If it did not succeed the invite goes back, unless
// it was revoked or has expired in the meantime.
func (manager *InviteManager) settle(invite *Invite, redeemed bool) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	_, found := manager.redeeming[invite.ID]
	delete(manager.redeeming, invite.ID)

	if redeemed {
		manager.log.Infof("invite %q for %q redeemed", invite.ID, invite.Username)
		return
	}
	if !found || invite.expired(manager.now()) {
		return
	}

	manager.invites[invite.ID] = invite
	manager.changed = true
	manager.saveLogged()
}

// Reserved reports whether username belongs to a pending invite, including one
// being redeemed right now. An invitee who picks another username than the
// admin did must not take one that someone else has been offered.
func (manager *InviteManager) Reserved(username string) bool {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	manager.prune()
	return manager.invited(username)
}

// invited reports whether username is reserved by a pending invite, including
// one that is being redeemed right now. The caller must hold manager.mu.
func (manager *InviteManager) invited(username string) bool {
	for _, invite := range manager.invites {
		if invite.Username == username {
			return true
		}
	}
	for _, invite := range manager.redeeming {
		if invite.Username == username {
			return true
		}
	}
	return false
}

// prune drops the expired invites. The caller must hold manager.mu.
func (manager *InviteManager) prune() {
	now := manager.now()
	for id, invite := range manager.invites {
		if invite.expired(now) {
			delete(manager.invites, id)
			manager.changed = true
			manager.log.Debugf("invite %q for %q expired", id, invite.Username)
		}
	}
}

// newID returns an id no invite is using. The caller must hold manager.mu.
func (manager *InviteManager) newID() (string, error) {
	for {
		id, err := random.GenerateRandomString(10)
		if err != nil {
			return "", err
		}
		if manager.invites[id] == nil && manager.redeeming[id] == nil {
			return id, nil
		}
	}
}

func newInviteToken() (string, error) {
	raw := make([]byte, inviteTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashInviteToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// load reads the saved invites and prunes the expired ones. The caller must
// hold manager.mu (or not have shared the manager yet).
func (manager *InviteManager) load() error {
	if manager.store.Has("invites") {
		manager.log.Debugf("loading...")

		// Load it.
		data, err := manager.store.Get("invites")
		if err != nil {
			return err
		}

		// Decode it.
		config := struct {
			Invites []*Invite `json:"invites"`
		}{}
		err = json.NewDecoder(bytes.NewReader(data)).Decode(&config)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		// Read the state into the manager (convert slice to map).
		manager.invites = make(map[string]*Invite)
		for _, invite := range config.Invites {
			manager.invites[invite.ID] = invite
		}

		// Don't carry invites that expired while the server was down.
		manager.prune()
	}
	return nil
}

// save writes the invites to the store. The caller must hold manager.mu.
func (manager *InviteManager) save() error {
	// Convert map to slice.
	config := struct {
		Invites []*Invite `json:"invites"`
	}{}
	for _, invite := range manager.invites {
		config.Invites = append(config.Invites, invite)
	}

	// Oldest first, which is also the order they are listed in, and stable
	// between saves.
	sort.Slice(config.Invites, func(i, j int) bool {
		if config.Invites[i].CreatedAt.Equal(config.Invites[j].CreatedAt) {
			return config.Invites[i].ID < config.Invites[j].ID
		}
		return config.Invites[i].CreatedAt.Before(config.Invites[j].CreatedAt)
	})

	// Encode it.
	data := &bytes.Buffer{}
	err := json.NewEncoder(data).Encode(config)
	if err != nil {
		return err
	}

	// Save it.
	return manager.store.Set("invites", data.Bytes())
}

// saveIfChanged saves if anything changed. The caller must hold manager.mu.
func (manager *InviteManager) saveIfChanged() error {
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

// saveLogged is saveIfChanged for callers with nobody to return the error to.
// changed stays set after a failure, so a later save retries. The caller must
// hold manager.mu.
func (manager *InviteManager) saveLogged() {
	err := manager.saveIfChanged()
	if err != nil {
		manager.log.Errorf("failed to save state: %s", err)
	}
}
