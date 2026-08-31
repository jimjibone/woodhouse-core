package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/jimjibone/log"
	"github.com/jimjibone/queue/v2"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

var (
	ErrNotificationNotFound      = errors.New("notification not found")
	ErrNotificationAlreadyExists = errors.New("notification already exists")
	ErrNotificationInvalid       = errors.New("notification invalid")
)

const (
	// maxNotifications caps the stored history. Every connecting client is sent
	// its whole visible share of it, so this is a bound on reconnect cost as
	// much as on disk.
	maxNotifications = 200
	// maxNotificationAge drops anything older regardless of count, so a quiet
	// server does not hand a client month-old news on connect.
	maxNotificationAge = 30 * 24 * time.Hour
)

// A Deliverer carries a notification out of band - APNs, Web Push, email. The
// in-app stream is deliberately NOT a Deliverer: it reads the same publisher
// every other stream reads, so it needs no registration here.
//
// Deliver is called once per accepted notification, on the manager's run
// goroutine and off its lock. Resolving the audience to reachable users is the
// transport's job, because only it knows who it can actually reach - a device
// token, a push subscription - and it should use Audience.Matches to do it so
// the membership rule stays in one place. Errors are logged and dropped:
// out-of-band delivery is best-effort and must never fail an Add.
type Deliverer interface {
	Name() string
	Deliver(ctx context.Context, notification *Notification) error
}

// NotificationManager owns the notifications and each user's read and dismissed
// state for them.
//
// It differs from the other managers in one way worth knowing before reading
// on: every subscriber sees a *different* subset of the same data, because a
// notification is addressed to an audience and its read flag is per-user. The
// manager does not try to model that. It publishes the whole record to every
// listener and leaves the filtering and the per-user projection to the stream
// handler, which is the only place that knows who is on the other end of the
// connection. Per-user publishers were the alternative and are worse: they save
// nothing (a distinct message still has to be built per recipient) and
// queue.Pub has no last-subscriber-left signal, so they would leak one
// publisher per username that had ever connected.
//
// The consequence for callers is that subscribing and snapshotting are two
// steps rather than one - see GetListener.
type NotificationManager struct {
	log         *log.Context
	wg          sync.WaitGroup
	ctx         context.Context
	close       func()
	store       stores.Store
	userManager *UserManager
	publisher   *queue.Pub[NotificationUpdate]
	// delivery hands new notifications to run() so Deliverers are called off
	// the lock, and so a slow transport cannot stall the caller of Add.
	delivery chan *Notification

	mu            sync.RWMutex
	changed       bool
	notifications map[string]*Notification
	deliverers    []Deliverer
}

type NotificationUpdate struct {
	Updated *Notification
	Removed *string
	// ForUser scopes this update to a single username. Empty means "everyone
	// the audience matches". It is set for a read mark and for a dismissal,
	// both of which are per-user facts that must not reach anybody else's
	// stream.
	ForUser string
}

func NewNotificationManager(store stores.Store, userManager *UserManager) (*NotificationManager, error) {
	ctx, close := context.WithCancel(context.Background())
	manager := &NotificationManager{
		log:           log.NewContext(log.DefaultLogger, "notification-manager", log.DebugLevel),
		ctx:           ctx,
		close:         close,
		store:         store,
		userManager:   userManager,
		publisher:     queue.NewPub[NotificationUpdate](),
		delivery:      make(chan *Notification, 16),
		notifications: make(map[string]*Notification),
	}

	// Load the previous state.
	err := manager.load()
	if err != nil {
		close()
		return nil, fmt.Errorf("failed to load state: %s", err)
	}

	// Save the state if changed. load() trims, so this persists the trim.
	err = manager.saveIfChanged()
	if err != nil {
		close()
		return nil, fmt.Errorf("failed to save state: %s", err)
	}

	// Subscribe before the goroutine starts, for the same reason ZoneManager
	// does: subscribing inside run() leaves a window in which a user removal is
	// published to nobody, and a missed removal leaves that user's read state
	// behind for a later account of the same name to inherit.
	//
	// GetUserUpdates, not GetListener: we only care about removals, and the
	// replay GetListener performs cannot be closed safely - see its comment.
	userUpdates := userManager.GetUserUpdates()

	manager.wg.Add(1)
	go manager.run(ctx, userUpdates)
	return manager, nil
}

func (manager *NotificationManager) Close() {
	manager.close()
	manager.wg.Wait()

	// Flush on the way out. The 60s save ticker in run() has already stopped,
	// so without this a notification from the last minute of uptime is lost.
	err := manager.saveIfChanged()
	if err != nil {
		manager.log.Errorf("failed to save state: %s", err)
	}
}

// GetListener subscribes to notification changes. Unlike the other managers
// this does NOT replay current state: callers get that from Snapshot, which
// they must call after subscribing.
//
// The reason is audience filtering. A manager-side replay has no idea who the
// subscriber is, so it would have to send every subscriber the entire history -
// including notifications they are not addressed to and ones they have
// dismissed - leaving the handler to discard most of it. Snapshot takes the
// username and role and returns only what that user can see.
//
// Subscribing before snapshotting is the correct order: an update landing in
// between is then delivered twice rather than not at all, and clients upsert by
// ID, so a duplicate converges and a miss would not.
func (manager *NotificationManager) GetListener() *queue.Sub[NotificationUpdate] {
	return manager.publisher.NewSub()
}

// AddDeliverer registers an out-of-band transport. Safe to call at any time,
// though in practice everything registers during startup wiring.
func (manager *NotificationManager) AddDeliverer(deliverer Deliverer) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	manager.deliverers = append(manager.deliverers, deliverer)
	manager.log.Infof("registered notification deliverer %q", deliverer.Name())
}

// Add stores a notification and publishes it. The caller supplies the ID, as
// with AddZone, so the service layer keeps ownership of ID generation.
func (manager *NotificationManager) Add(notification *Notification) error {
	if notification.ID == "" {
		return fmt.Errorf("%w: empty id", ErrNotificationInvalid)
	}
	if notification.Title == "" {
		return fmt.Errorf("%w: empty title", ErrNotificationInvalid)
	}
	if !notification.Audience.Valid() {
		return fmt.Errorf("%w: audience %s", ErrNotificationInvalid, notification.Audience)
	}
	if notification.Created.IsZero() {
		notification.Created = time.Now().UTC()
	}

	manager.mu.Lock()

	if manager.notifications[notification.ID] != nil {
		manager.mu.Unlock()
		return ErrNotificationAlreadyExists
	}

	// A collapse key replaces the previous notification carrying it, so a
	// flapping source cannot fill the history. The removal is published too -
	// clients hold the old row and would otherwise show both.
	if notification.CollapseKey != "" {
		for id, existing := range manager.notifications {
			if existing.CollapseKey != notification.CollapseKey {
				continue
			}
			delete(manager.notifications, id)
			manager.log.Debugf("notification %q collapsed into %q on key %q", id, notification.ID, notification.CollapseKey)
			manager.publisher.Pub(NotificationUpdate{Removed: &id})
		}
	}

	manager.notifications[notification.ID] = notification
	manager.changed = true

	manager.log.Infof("notification added %s", notification)

	manager.publisher.Pub(NotificationUpdate{Updated: notification.Clone()})
	manager.trim()

	// Clone before releasing the lock: the Deliverers run later, on another
	// goroutine, and must not read a record the manager is still mutating.
	forDelivery := notification.Clone()
	manager.mu.Unlock()

	// Non-blocking. A wedged or saturated delivery pipeline must not fail or
	// delay the in-app notification, which is the delivery path that matters.
	select {
	case manager.delivery <- forDelivery:
	default:
		manager.log.Warnf("delivery queue full, notification %q not sent out of band", notification.ID)
	}

	return nil
}

// MarkRead marks one notification read for one user. Idempotent, and silent
// when nothing changes so a repeated click costs no stream traffic.
func (manager *NotificationManager) MarkRead(id, username string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	notification := manager.notifications[id]
	if notification == nil {
		return ErrNotificationNotFound
	}

	if notification.IsReadBy(username) {
		return nil
	}

	now := time.Now().UTC()
	notification.ensureState(username).Read = &now
	manager.changed = true

	manager.publisher.Pub(NotificationUpdate{Updated: notification.Clone(), ForUser: username})

	return nil
}

// MarkAllRead marks everything currently visible to a user read, returning how
// many it changed. It needs the role so it only touches what that user can
// actually see - marking a notification they are not addressed to would leave
// state that never surfaces.
func (manager *NotificationManager) MarkAllRead(username string, role auth.Role) (int, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	now := time.Now().UTC()
	marked := 0
	for _, notification := range manager.notifications {
		if !notification.visibleTo(username, role) || notification.IsReadBy(username) {
			continue
		}

		read := now
		notification.ensureState(username).Read = &read
		marked++

		manager.publisher.Pub(NotificationUpdate{Updated: notification.Clone(), ForUser: username})
	}

	if marked > 0 {
		manager.changed = true
		manager.log.Infof("user %q marked %d notifications read", username, marked)
	}

	return marked, nil
}

// Dismiss removes a notification from one user's inbox. It stays in the history
// for everybody else in its audience, which is why this is per-user state and
// not a delete.
func (manager *NotificationManager) Dismiss(id, username string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	notification := manager.notifications[id]
	if notification == nil {
		return ErrNotificationNotFound
	}

	if notification.IsDismissedBy(username) {
		return nil
	}

	now := time.Now().UTC()
	state := notification.ensureState(username)
	state.Dismissed = &now
	// Dismissing implies reading it. Without this the unread badge would count
	// a row the user can no longer see.
	if state.Read == nil {
		state.Read = &now
	}
	manager.changed = true

	manager.log.Debugf("user %q dismissed notification %q", username, id)

	manager.publisher.Pub(NotificationUpdate{Removed: &id, ForUser: username})

	return nil
}

// Snapshot returns the notifications visible to a user, newest first. The
// stream handler calls this immediately after GetListener to build its initial
// batch; see GetListener for why the replay lives here rather than in run().
func (manager *NotificationManager) Snapshot(username string, role auth.Role) []*Notification {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	var out []*Notification
	for _, notification := range manager.notifications {
		if notification.visibleTo(username, role) {
			out = append(out, notification.Clone())
		}
	}
	sortNotifications(out)
	return out
}

// Unread counts the unread notifications visible to a user.
func (manager *NotificationManager) Unread(username string, role auth.Role) int {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	count := 0
	for _, notification := range manager.notifications {
		if notification.visibleTo(username, role) && !notification.IsReadBy(username) {
			count++
		}
	}
	return count
}

// trim enforces the history bounds, publishing a removal for each notification
// it drops. Callers must hold manager.mu.
//
// Publishing matters: a browser tab left open for a week holds every row it has
// ever been sent, so a silent trim would leave it showing notifications the
// server has forgotten.
func (manager *NotificationManager) trim() {
	if len(manager.notifications) == 0 {
		return
	}

	remove := func(id, reason string) {
		delete(manager.notifications, id)
		manager.changed = true
		manager.log.Debugf("notification %q trimmed (%s)", id, reason)
		removedID := id
		manager.publisher.Pub(NotificationUpdate{Removed: &removedID})
	}

	cutoff := time.Now().UTC().Add(-maxNotificationAge)
	for id, notification := range manager.notifications {
		if notification.Created.Before(cutoff) {
			remove(id, "too old")
		}
	}

	if len(manager.notifications) <= maxNotifications {
		return
	}

	ordered := make([]*Notification, 0, len(manager.notifications))
	for _, notification := range manager.notifications {
		ordered = append(ordered, notification)
	}
	sortNotifications(ordered)

	for _, notification := range ordered[maxNotifications:] {
		remove(notification.ID, "over count")
	}
}

// pruneUser drops a deleted user's state from every notification.
//
// Without this a recreated account of the same name inherits its predecessor's
// read marks and dismissals, and never sees notifications the previous holder
// had already dealt with. Usernames are the identity here, so reuse is real.
func (manager *NotificationManager) pruneUser(username string) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	pruned := 0
	for _, notification := range manager.notifications {
		if notification.pruneUser(username) {
			pruned++
		}
	}

	if pruned > 0 {
		manager.changed = true
		manager.log.Infof("pruned removed user %q from %d notifications", username, pruned)
	}
}

// fanOutToDeliverers runs every registered transport. Best-effort by design:
// an error is logged and the next transport still runs.
func (manager *NotificationManager) fanOutToDeliverers(notification *Notification) {
	manager.mu.RLock()
	deliverers := make([]Deliverer, len(manager.deliverers))
	copy(deliverers, manager.deliverers)
	manager.mu.RUnlock()

	for _, deliverer := range deliverers {
		err := deliverer.Deliver(manager.ctx, notification)
		if err != nil {
			manager.log.Errorf("deliverer %q failed on notification %q: %s", deliverer.Name(), notification.ID, err)
		}
	}
}

// sortNotifications orders newest first, ties broken by ID so the order is
// stable across saves.
func sortNotifications(notifications []*Notification) {
	sort.Slice(notifications, func(i, j int) bool {
		if notifications[i].Created.Equal(notifications[j].Created) {
			return notifications[i].ID < notifications[j].ID
		}
		return notifications[i].Created.After(notifications[j].Created)
	})
}

func (manager *NotificationManager) load() error {
	if manager.store.Has("notifications.json") {
		manager.log.Debugf("loading...")

		data, err := manager.store.Get("notifications.json")
		if err != nil {
			return err
		}

		config := struct {
			Notifications []*Notification `json:"notifications"`
		}{}
		err = json.NewDecoder(bytes.NewReader(data)).Decode(&config)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		manager.notifications = make(map[string]*Notification)
		for _, notification := range config.Notifications {
			manager.notifications[notification.ID] = notification
		}

		// Apply the bounds on the way in, so a restart after a long downtime
		// does not replay ancient history and a file written by a looser build
		// gets normalised. Nobody is subscribed yet, so the removals it
		// publishes go nowhere - that is fine, there is no client to correct.
		manager.mu.Lock()
		manager.trim()
		manager.mu.Unlock()
	}
	return nil
}

func (manager *NotificationManager) save() error {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	config := struct {
		Notifications []*Notification `json:"notifications"`
	}{}
	for _, notification := range manager.notifications {
		config.Notifications = append(config.Notifications, notification)
	}
	// Sorted to maintain consistent structure between saves. Newest first,
	// which is also the order everything reads them in.
	sortNotifications(config.Notifications)

	data := &bytes.Buffer{}
	encoder := json.NewEncoder(data)
	encoder.SetIndent("", "\t")
	err := encoder.Encode(config)
	if err != nil {
		return err
	}

	return manager.store.Set("notifications.json", data.Bytes())
}

func (manager *NotificationManager) saveIfChanged() error {
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

func (manager *NotificationManager) run(ctx context.Context, userUpdates *queue.Sub[UserUpdate]) {
	defer manager.wg.Done()
	defer userUpdates.Close()

	manager.mu.RLock()
	manager.log.Debugf("initial notifications are: %d", len(manager.notifications))
	manager.mu.RUnlock()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case update := <-userUpdates.Sub():
			// Only a removal matters. An ordinary user update is nothing to
			// us: a notification's audience is evaluated fresh on every send,
			// so a role change needs no bookkeeping here.
			if update.Removed != nil {
				manager.pruneUser(*update.Removed)
			}

		case notification := <-manager.delivery:
			manager.fanOutToDeliverers(notification)

		case <-ticker.C:
			// Trim on the tick as well as on Add, so the age bound still
			// applies on a server that is not receiving anything.
			manager.mu.Lock()
			manager.trim()
			manager.mu.Unlock()

			err := manager.saveIfChanged()
			if err != nil {
				manager.log.Errorf("failed to save state: %s", err)
			}
		}
	}
}
