package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/jimjibone/log"
	"github.com/jimjibone/queue/v2"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/shared/stores"
	"gopkg.in/yaml.v3"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrAlreadyExists     = errors.New("already exists")
	ErrIncorrectPassword = errors.New("incorrect password")
)

type UserManager struct {
	log         *log.Context
	wg          sync.WaitGroup
	mu          sync.RWMutex
	close       func()
	store       stores.Store
	users       map[string]*User // key=username
	changed     bool
	publisher   *queue.Pub[UserUpdate]
	listenerAdd chan *queue.Sub[UserUpdate]
}

type UserUpdate struct {
	Updated *User
	Removed *string
}

func NewUserManager(store stores.Store) (*UserManager, error) {
	ctx, close := context.WithCancel(context.Background())
	manager := &UserManager{
		log:         log.NewContext(log.DefaultLogger, "user-manager", log.DebugLevel),
		close:       close,
		store:       store,
		users:       make(map[string]*User),
		publisher:   queue.NewPub[UserUpdate](),
		listenerAdd: make(chan *queue.Sub[UserUpdate], 1),
	}

	// Load the previous state.
	err := manager.load()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %s", err)
	}

	// Save the state if changed.
	err = manager.saveIfChanged()
	if err != nil {
		return nil, fmt.Errorf("failed to save state: %s", err)
	}

	manager.wg.Add(1)
	go manager.run(ctx)
	return manager, nil
}

func (manager *UserManager) Close() {
	manager.close()
	manager.wg.Wait()

	err := manager.saveIfChanged()
	if err != nil {
		manager.log.Fatalf("failed to save state: %s", err)
	}
}

func (manager *UserManager) GetListener() *queue.Sub[UserUpdate] {
	sub := manager.publisher.NewSub()
	manager.listenerAdd <- sub
	return sub
}

func (manager *UserManager) HasAnAdmin() bool {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	admins := 0
	for _, user := range manager.users {
		if user.Role == auth.AdminRole {
			admins++
		}
	}
	return admins > 0
}

func (manager *UserManager) Store(user *User) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if manager.users[user.Username] != nil {
		return ErrAlreadyExists
	}

	manager.users[user.Username] = user.Clone()
	manager.changed = true

	manager.log.Infof("user %q added as %s", user.Username, user.Role)

	// Publish the new/updated user to the listeners.
	manager.publisher.Pub(UserUpdate{Updated: user.Clone()})

	return nil
}

func (manager *UserManager) Find(username string) *User {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	user := manager.users[username]
	if user == nil {
		return nil
	}

	return user.Clone()
}

func (manager *UserManager) Delete(username string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if _, found := manager.users[username]; found {
		delete(manager.users, username)
		manager.changed = true

		manager.log.Infof("user %q deleted", username)

		// Publish the removed user to the listeners.
		manager.publisher.Pub(UserUpdate{Removed: &username})

		return nil
	}

	return ErrUserNotFound
}

func (manager *UserManager) SetFullname(username string, fullname string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	user := manager.users[username]
	if user == nil {
		return ErrUserNotFound
	}

	user.Fullname = fullname

	manager.users[user.Username] = user.Clone()
	manager.changed = true

	manager.log.Infof("user %q changed fullname to %q", user.Username, user.Fullname)

	// Publish the new/updated user to the listeners.
	manager.publisher.Pub(UserUpdate{Updated: user.Clone()})

	return nil
}

func (manager *UserManager) SetRole(username string, role auth.Role) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	user := manager.users[username]
	if user == nil {
		return ErrUserNotFound
	}

	user.Role = role

	manager.users[user.Username] = user.Clone()
	manager.changed = true

	manager.log.Infof("user %q changed role to %q", user.Username, user.Role)

	// Publish the new/updated user to the listeners.
	manager.publisher.Pub(UserUpdate{Updated: user.Clone()})

	return nil
}

// ChangePassword replaces a user's password, but only if currentPassword
// matches the one already stored. This is the self-service path: the
// re-entered current password is what stops a stolen session from changing
// the password and locking the real owner out, so callers must not skip it
// on the grounds that the request was already authenticated.
//
// The check and the write happen under one lock so a concurrent change
// can't slip in between them.
func (manager *UserManager) ChangePassword(username string, currentPassword, newPassword string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	user := manager.users[username]
	if user == nil {
		return ErrUserNotFound
	}

	if !user.IsCorrectPassword(currentPassword) {
		manager.log.Warnf("user %q password change rejected: current password incorrect", user.Username)
		return ErrIncorrectPassword
	}

	// SetPassword clears ResetPassword, which is what retires a temporary
	// password an admin handed out.
	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	manager.users[user.Username] = user.Clone()
	manager.changed = true

	manager.log.Infof("user %q changed password", user.Username)

	// Publish the new/updated user to the listeners.
	manager.publisher.Pub(UserUpdate{Updated: user.Clone()})

	return nil
}

// ResetPassword sets a temporary password for a user without knowing their
// current one, and flags the account so they must choose a new password
// before they can use the app. This is the admin path - for an admin
// changing their own password, use ChangePassword.
func (manager *UserManager) ResetPassword(username string, newPassword string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	user := manager.users[username]
	if user == nil {
		return ErrUserNotFound
	}

	if err := user.SetPassword(newPassword); err != nil {
		return err
	}
	// SetPassword clears the flag, so raise it after: the user is holding a
	// password somebody else chose, exactly like a freshly created account.
	user.ResetPassword = true

	manager.users[user.Username] = user.Clone()
	manager.changed = true

	manager.log.Infof("user %q password reset by an admin, reset required on next login", user.Username)

	// Publish the new/updated user to the listeners.
	manager.publisher.Pub(UserUpdate{Updated: user.Clone()})

	return nil
}

// func (store *UserManager) AddUserToken(username string, uuid string, exp time.Time) error {
// 	store.mu.Lock()
// 	defer store.mu.Unlock()

// 	user := store.users[username]
// 	if user == nil {
// 		return ErrUserNotFound
// 	}

// 	user.AddToken(uuid, exp)
// 	store.changed = true

// 	return nil
// }

// func (store *UserManager) HasUserToken(username string, uuid string) (bool, error) {
// 	store.mu.RLock()
// 	defer store.mu.RUnlock()

// 	user := store.users[username]
// 	if user == nil {
// 		return false, ErrUserNotFound
// 	}

// 	return user.HasToken(uuid), nil
// }

// func (store *UserManager) RevokeUserToken(username string, uuid string) error {
// 	store.mu.Lock()
// 	defer store.mu.Unlock()

// 	user := store.users[username]
// 	if user == nil {
// 		return ErrUserNotFound
// 	}

// 	user.RevokeToken(uuid)
// 	store.changed = true

// 	return nil
// }

// func (store *UserManager) ReplaceUserToken(username string, add, remove string, exp time.Time) error {
// 	store.mu.Lock()
// 	defer store.mu.Unlock()

// 	user := store.users[username]
// 	if user == nil {
// 		return ErrUserNotFound
// 	}

// 	user.RevokeToken(remove)
// 	user.AddToken(add, exp)
// 	store.changed = true

// 	return nil
// }

// func (store *UserManager) FillGetUsersReply(reply *clientsapi.GetUsersReply, exclude string) {
// 	store.mu.RLock()
// 	defer store.mu.RUnlock()

// 	for _, user := range store.users {
// 		if user.Username != exclude {
// 			reply.Users = append(reply.Users, &clientsapi.GetUsersReplyUser{
// 				Username: user.Username,
// 				Role:     string(user.Role),
// 			})
// 		}
// 	}
// }

func (manager *UserManager) load() error {
	if manager.store.Has("users") {
		manager.log.Debugf("loading...")

		// Load it.
		data, err := manager.store.Get("users")
		if err != nil {
			return err
		}

		// Decode it.
		config := struct {
			RefreshSecret string  `json:"refresh-secret"`
			AccessSecret  string  `json:"access-secret"`
			Users         []*User `json:"users"`
		}{}
		err = json.NewDecoder(bytes.NewReader(data)).Decode(&config)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			if te, ok := err.(*yaml.TypeError); ok {
				fmt.Fprintln(os.Stderr, te.Errors)
			}
			return err
		}

		// Read the state into the manager (convert slice to map).
		manager.users = make(map[string]*User)
		for _, user := range config.Users {
			manager.users[user.Username] = user
		}
	}
	return nil
}

func (manager *UserManager) save() error {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	// Convert map to slice.
	config := struct {
		Users []*User `json:"users"`
	}{}
	for _, user := range manager.users {
		config.Users = append(config.Users, user)
	}

	// Sorted to maintain consistent structure between saves.
	sort.Slice(config.Users, func(i, j int) bool {
		return config.Users[i].Username < config.Users[j].Username
	})

	// Encode it.
	data := &bytes.Buffer{}
	err := json.NewEncoder(data).Encode(config)
	if err != nil {
		return err
	}

	// Save it.
	err = manager.store.Set("users", data.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (manager *UserManager) saveIfChanged() error {
	// Save the config if changed.
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

func (manager *UserManager) run(ctx context.Context) {
	defer manager.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		// case update := <-manager.rxDeviceUpdates.Pop():
		// 	manager.handleDeviceUpdate(update)

		// case update := <-manager.setFavourites.Pop():
		// 	manager.handleFavoriteUpdate(update)

		case <-ticker.C:
			// Clean expired tokens from users.
			// for _, user := range manager.users {
			// 	if user.CleanTokens() {
			// 		manager.changed = true
			// 	}
			// }

			// Save the config if anything changed.
			err := manager.saveIfChanged()
			if err != nil {
				manager.log.Fatalf("failed to save state: %s", err)
			}

		case lis := <-manager.listenerAdd:
			// Publish all users to the new listener.
			for _, user := range manager.users {
				manager.publisher.Send(lis, UserUpdate{Updated: user.Clone()})
			}

			// Send and empty update to indicate the end of the initial list.
			manager.publisher.Send(lis, UserUpdate{})
		}
	}
}
