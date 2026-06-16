package core

import (
	"context"
	"sync"

	"github.com/jimjibone/log"
	"github.com/jimjibone/queue/v2"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

type FavoritesManager struct {
	log           *log.Context
	wg            sync.WaitGroup
	ctx           context.Context
	close         func()
	deviceManager *DeviceManager
	publisher     *queue.Pub[FavoriteUpdate]
	listenerAdd   chan *queue.Sub[FavoriteUpdate]
}

type FavoriteUpdate struct {
	Updated *Favorite
	Removed *FavoriteID
}

func NewFavoritesManager(store stores.Store, deviceManager *DeviceManager) *FavoritesManager {
	ctx, close := context.WithCancel(context.Background())
	manager := &FavoritesManager{
		log:           log.NewContext(log.DefaultLogger, "favorites-manager", log.DebugLevel),
		ctx:           ctx,
		close:         close,
		deviceManager: deviceManager,
		publisher:     queue.NewPub[FavoriteUpdate](),
		listenerAdd:   make(chan *queue.Sub[FavoriteUpdate], 1),
	}

	manager.wg.Add(1)
	go manager.run(ctx)
	return manager
}

func (manager *FavoritesManager) Close() {
	manager.close()
	manager.wg.Wait()
}

func (manager *FavoritesManager) GetListener() *queue.Sub[FavoriteUpdate] {
	sub := manager.publisher.NewSub()
	manager.listenerAdd <- sub
	return sub
}

func (manager *FavoritesManager) run(ctx context.Context) {
	defer manager.wg.Done()

	deviceUpdates := manager.deviceManager.GetDeviceUpdates()
	defer deviceUpdates.Close()

	faves := make(map[string]*Favorite)

	// Build up faves from initial deviceManager state.
	for dev := range manager.deviceManager.GetDevices() {
		for _, srv := range dev.Services {
			if srv.GetFavorite() {
				faveID := FavoriteID{
					DeviceID:  dev.GetId(),
					ServiceID: srv.GetId(),
				}
				// manager.log.Errorf("prior fave: %v", faveID)
				fave := &Favorite{
					DeviceID:  faveID.DeviceID,
					ServiceID: faveID.ServiceID,
				}
				fave.Update(dev)
				faves[faveID.Key()] = fave
			}
		}
	}
	// manager.log.Debugf("prior faves is: %v", len(faves))

	for {
		select {
		case <-ctx.Done():
			return

		case update := <-deviceUpdates.Sub():
			// Update faves and publish to listeners.
			for _, fave := range faves {
				if fave.DeviceID == update.RemovedID {
					// Remove from the faves.}
					faveID := FavoriteID{
						DeviceID:  fave.DeviceID,
						ServiceID: fave.ServiceID,
					}
					delete(faves, faveID.Key())

					// Publish the removed fave to the listeners.
					manager.publisher.Pub(FavoriteUpdate{Removed: &faveID})
				} else if update.Update != nil && fave.DeviceID == update.Update.GetId() {
					if fave.Update(update.Update) {
						// Publish the updated fave to the listeners.
						manager.publisher.Pub(FavoriteUpdate{Updated: fave.Clone()})
					}
				}
			}

			// Add missing faves.
			if update.Update != nil {
				for _, service := range update.Update.Services {
					faveID := FavoriteID{
						DeviceID:  update.Update.GetId(),
						ServiceID: service.GetId(),
					}
					if service.GetFavorite() {
						if _, found := faves[faveID.Key()]; !found {
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
							manager.publisher.Pub(FavoriteUpdate{Updated: fave.Clone()})
						}
					} else {
						if _, found := faves[faveID.Key()]; found {
							// Remove from the faves.
							delete(faves, faveID.Key())

							// Publish the removed fave to the listeners.
							manager.publisher.Pub(FavoriteUpdate{Removed: &faveID})
						}
					}
				}
			}

		case lis := <-manager.listenerAdd:
			// Publish all faves to the new listener.
			for _, fave := range faves {
				manager.publisher.Send(lis, FavoriteUpdate{Updated: fave.Clone()})
			}

			// Send an empty update to indicate the end of the initial list.
			manager.publisher.Send(lis, FavoriteUpdate{})
		}
	}
}
