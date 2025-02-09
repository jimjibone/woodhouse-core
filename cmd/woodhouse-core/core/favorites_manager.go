package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/stores"
	"gopkg.in/yaml.v3"
)

type FavoritesManager struct {
	log            *log.Context
	wg             sync.WaitGroup
	ctx            context.Context
	close          func()
	store          stores.Store
	deviceManager  *DeviceManager
	listenerAdd    chan FavoritesListener
	listenerRemove chan FavoritesListener
	favoriteAdd    chan (FavoriteID)
	favoriteRemove chan (FavoriteID)
	favorites      map[string]FavoriteID // key=key
	changed        bool
}

type FavoritesListener chan FavoriteUpdate

type FavoriteUpdate struct {
	Updated *Favorite
	Removed *FavoriteID
}

func NewFavoritesManager(store stores.Store, deviceManager *DeviceManager) (*FavoritesManager, error) {
	ctx, close := context.WithCancel(context.Background())
	manager := &FavoritesManager{
		log:            log.NewContext(log.DefaultLogger, "favorites-manager", log.DebugLevel),
		ctx:            ctx,
		close:          close,
		store:          store,
		deviceManager:  deviceManager,
		listenerAdd:    make(chan FavoritesListener),
		listenerRemove: make(chan FavoritesListener),
		favoriteAdd:    make(chan FavoriteID, 1),
		favoriteRemove: make(chan FavoriteID, 1),
		favorites:      make(map[string]FavoriteID),
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

func (manager *FavoritesManager) Close() {
	manager.close()
	manager.wg.Wait()

	err := manager.saveIfChanged()
	if err != nil {
		manager.log.Errorf("failed to save state: %s", err)
	}
}

func (manager *FavoritesManager) AddListener(lis FavoritesListener) {
	manager.listenerAdd <- lis
}

func (manager *FavoritesManager) RemoveListener(lis FavoritesListener) {
	manager.listenerRemove <- lis
}

func (manager *FavoritesManager) AddFavorite(deviceID, serviceID string) {
	manager.favoriteAdd <- FavoriteID{
		DeviceID:  deviceID,
		ServiceID: serviceID,
	}
}

func (manager *FavoritesManager) RemoveFavorite(deviceID, serviceID string) {
	manager.favoriteRemove <- FavoriteID{
		DeviceID:  deviceID,
		ServiceID: serviceID,
	}
}

func (manager *FavoritesManager) load() error {
	if manager.store.Has("favorites") {
		manager.log.Debugf("loading...")

		// Load it.
		data, err := manager.store.Get("favorites")
		if err != nil {
			return err
		}

		// Decode it.
		config := struct {
			Favorites []FavoriteID `json:"favorites"`
		}{}
		err = json.NewDecoder(bytes.NewReader(data)).Decode(&config)
		// err = yaml.NewDecoder(bytes.NewReader(data)).Decode(&config)
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

		// Read the state into the manager (convert slice to map).
		manager.favorites = make(map[string]FavoriteID)
		for _, fave := range config.Favorites {
			manager.favorites[fave.Key()] = fave
		}
	}
	return nil
}

func (manager *FavoritesManager) save() error {
	// Convert map to slice.
	config := struct {
		Favorites []FavoriteID `json:"favorites"`
	}{}
	for _, fave := range manager.favorites {
		config.Favorites = append(config.Favorites, fave)
	}
	// Sorted to maintain consistent structure between saves.
	sort.Slice(config.Favorites, func(i, j int) bool {
		return config.Favorites[i].Key() < config.Favorites[j].Key()
	})

	// Encode it.
	data := &bytes.Buffer{}
	err := json.NewEncoder(data).Encode(config)
	// err := yaml.NewEncoder(data).Encode(config)
	if err != nil {
		return err
	}

	// Save it.
	err = manager.store.Set("favorites", data.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (manager *FavoritesManager) saveIfChanged() error {
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

func (manager *FavoritesManager) run(ctx context.Context) {
	defer manager.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	deviceUpdates := manager.deviceManager.GetDeviceUpdates()
	defer deviceUpdates.Close()

	listeners := make(map[FavoritesListener]struct{})
	faves := make(map[string]*Favorite)

	// Build up faves from initial deviceManager state.
	for _, faveID := range manager.favorites {
		faves[faveID.Key()] = &Favorite{
			DeviceID:  faveID.DeviceID,
			ServiceID: faveID.ServiceID,
		}
	}
	for dev := range manager.deviceManager.GetDevices() {
		for _, fave := range faves {
			if fave.DeviceID == dev.GetId() {
				fave.Update(dev)
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			err := manager.saveIfChanged()
			if err != nil {
				manager.log.Fatalf("failed to save state: %s", err)
			}

		case update := <-deviceUpdates.Sub():
			// Update faves and publish to listeners.
			for _, fave := range faves {
				if fave.DeviceID == update.GetId() {
					if fave.Update(update) {
						// Publish the updated fave to the listeners.
						for lis := range listeners {
							lis <- FavoriteUpdate{Updated: fave.Clone()}
						}
					}
				}
			}

		case lis := <-manager.listenerAdd:
			if _, found := listeners[lis]; !found {
				listeners[lis] = struct{}{}

				// Publish all faves to the new listener.
				for _, fave := range faves {
					lis <- FavoriteUpdate{Updated: fave.Clone()}
				}
			}

		case lis := <-manager.listenerRemove:
			// Remove the listener.
			delete(listeners, lis)

		case faveID := <-manager.favoriteAdd:
			if _, found := manager.favorites[faveID.Key()]; !found {
				// Add to the settings.
				manager.favorites[faveID.Key()] = faveID
				manager.changed = true

				// Get the device state.
				dev := manager.deviceManager.GetDevice(faveID.DeviceID)

				// Create the new fave.
				fave := &Favorite{
					DeviceID:  faveID.DeviceID,
					ServiceID: faveID.ServiceID,
				}
				fave.Update(dev)
				faves[faveID.Key()] = fave

				// Publish the new fave to the listeners.
				for lis := range listeners {
					lis <- FavoriteUpdate{Updated: fave.Clone()}
				}
			}

		case faveID := <-manager.favoriteRemove:
			if _, found := manager.favorites[faveID.Key()]; found {
				// Remove from the settings.
				delete(manager.favorites, faveID.Key())
				manager.changed = true

				// Remove from the faves.
				delete(faves, faveID.Key())

				// Publish the removed fave to the listeners.
				for lis := range listeners {
					lis <- FavoriteUpdate{Removed: &faveID}
				}
			}
		}
	}
}
