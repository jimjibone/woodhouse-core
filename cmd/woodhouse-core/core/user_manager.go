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

	"github.com/jimjibone/woodhouse-4/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/stores"
	"gopkg.in/yaml.v3"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrAlreadyExists = errors.New("already exists")
)

type UserManager struct {
	log     *log.Context
	wg      sync.WaitGroup
	mu      sync.RWMutex
	close   func()
	store   stores.Store
	users   map[string]*User // key=username
	changed bool
}

func NewUserManager(store stores.Store) (*UserManager, error) {
	ctx, close := context.WithCancel(context.Background())
	manager := &UserManager{
		log:   log.NewContext(log.DefaultLogger, "user-manager", log.DebugLevel),
		close: close,
		store: store,
		users: make(map[string]*User),
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

func (store *UserManager) HasAnAdmin() bool {
	store.mu.RLock()
	defer store.mu.RUnlock()
	admins := 0
	for _, user := range store.users {
		if user.Role == auth.AdminRole {
			admins++
		}
	}
	return admins > 0
}

func (store *UserManager) Store(user *User) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if store.users[user.Username] != nil {
		return ErrAlreadyExists
	}

	store.users[user.Username] = user.Clone()
	store.changed = true

	return nil
}

func (store *UserManager) Find(username string) *User {
	store.mu.RLock()
	defer store.mu.RUnlock()

	user := store.users[username]
	if user == nil {
		return nil
	}

	return user.Clone()
}

func (store *UserManager) Delete(username string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.users, username)
	store.changed = true
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
		}
	}
}
