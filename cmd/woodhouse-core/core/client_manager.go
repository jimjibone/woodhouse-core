package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jimjibone/queue/v2"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/stores"
)

var (
	ErrClientNotFound      = errors.New("client not found")
	ErrClientAlreadyExists = errors.New("client already exists")
	ErrPairingNotFound     = errors.New("pairing request not found")
)

type ClientUpdate struct {
	Updated *Client
	Removed *string
}

type PairingUpdate struct {
	Updated *PairingRequest
	Removed *string
}

type ClientManager struct {
	log *log.Context

	wg    sync.WaitGroup
	close func()
	ctx   context.Context

	store stores.Store

	mu              sync.RWMutex
	clients         map[string]*Client
	pairingRequests map[string]*PairingRequest

	changed bool

	clientPublisher   *queue.Pub[ClientUpdate]
	clientListenerAdd chan *queue.Sub[ClientUpdate]

	pairingPublisher   *queue.Pub[PairingUpdate]
	pairingListenerAdd chan *queue.Sub[PairingUpdate]
}

func NewClientManager(store stores.Store) (*ClientManager, error) {
	ctx, close := context.WithCancel(context.Background())
	manager := &ClientManager{
		log:                log.NewContext(log.DefaultLogger, "client-manager", log.DebugLevel),
		ctx:                ctx,
		close:              close,
		store:              store,
		clients:            make(map[string]*Client),
		pairingRequests:    make(map[string]*PairingRequest),
		clientPublisher:    queue.NewPub[ClientUpdate](),
		clientListenerAdd:  make(chan *queue.Sub[ClientUpdate], 1),
		pairingPublisher:   queue.NewPub[PairingUpdate](),
		pairingListenerAdd: make(chan *queue.Sub[PairingUpdate], 1),
	}

	if err := manager.load(); err != nil {
		return nil, fmt.Errorf("failed to load state: %s", err)
	}

	if err := manager.saveIfChanged(); err != nil {
		return nil, fmt.Errorf("failed to save state: %s", err)
	}

	manager.wg.Add(1)
	go manager.run(ctx)

	return manager, nil
}

func (manager *ClientManager) Close() {
	manager.close()
	manager.wg.Wait()

	if err := manager.saveIfChanged(); err != nil {
		manager.log.Fatalf("failed to save state: %s", err)
	}
}

func (manager *ClientManager) GetClientListener() *queue.Sub[ClientUpdate] {
	sub := manager.clientPublisher.NewSub()
	manager.clientListenerAdd <- sub
	return sub
}

func (manager *ClientManager) GetPairingListener() *queue.Sub[PairingUpdate] {
	sub := manager.pairingPublisher.NewSub()
	manager.pairingListenerAdd <- sub
	return sub
}

func (manager *ClientManager) GetClients() []*Client {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	clients := make([]*Client, 0, len(manager.clients))
	for _, client := range manager.clients {
		clients = append(clients, client.Clone())
	}
	return clients
}

func (manager *ClientManager) FindClient(id string) *Client {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	client, found := manager.clients[id]
	if !found {
		return nil
	}
	return client.Clone()
}

func (manager *ClientManager) StoreClient(client *Client) error {
	if client == nil || client.ID == "" {
		return fmt.Errorf("client id not set")
	}

	manager.mu.Lock()
	defer manager.mu.Unlock()

	if manager.clients[client.ID] != nil {
		return ErrClientAlreadyExists
	}

	now := time.Now()
	if client.FirstSeen.IsZero() {
		client.FirstSeen = now
	}
	if client.LastSeen.IsZero() {
		client.LastSeen = now
	}

	manager.clients[client.ID] = client.Clone()
	manager.changed = true

	manager.log.Infof("client %q added", client.ID)
	manager.clientPublisher.Pub(ClientUpdate{Updated: client.Clone()})

	return nil
}

func (manager *ClientManager) UpdateClient(client *Client) error {
	if client == nil || client.ID == "" {
		return fmt.Errorf("client id not set")
	}

	manager.mu.Lock()
	defer manager.mu.Unlock()

	existing := manager.clients[client.ID]
	if existing == nil {
		manager.clients[client.ID] = client.Clone()
		manager.changed = true
		manager.clientPublisher.Pub(ClientUpdate{Updated: client.Clone()})
		return nil
	}

	manager.clients[client.ID] = client.Clone()
	manager.changed = true
	manager.clientPublisher.Pub(ClientUpdate{Updated: client.Clone()})

	return nil
}

func (manager *ClientManager) DeleteClient(id string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if _, found := manager.clients[id]; !found {
		return ErrClientNotFound
	}

	delete(manager.clients, id)
	manager.changed = true

	manager.log.Infof("client %q deleted", id)
	manager.clientPublisher.Pub(ClientUpdate{Removed: &id})

	return nil
}

func (manager *ClientManager) SetClientOnline(id string, online bool, lastSeen time.Time, lastIP string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	client := manager.clients[id]
	if client == nil {
		client = &Client{ID: id}
	}

	if client.FirstSeen.IsZero() {
		client.FirstSeen = time.Now()
	}
	if !lastSeen.IsZero() {
		client.LastSeen = lastSeen
	} else {
		client.LastSeen = time.Now()
	}
	client.Online = online
	if lastIP != "" {
		client.LastIP = lastIP
	}

	manager.clients[id] = client.Clone()
	manager.changed = true

	manager.clientPublisher.Pub(ClientUpdate{Updated: client.Clone()})
	return nil
}

func (manager *ClientManager) SetClientPaired(id string, paired bool) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	client := manager.clients[id]
	if client == nil {
		return ErrClientNotFound
	}

	client.Paired = paired
	manager.changed = true

	manager.clientPublisher.Pub(ClientUpdate{Updated: client.Clone()})
	return nil
}

func (manager *ClientManager) SetClientBlocked(id string, blocked bool) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	client := manager.clients[id]
	if client == nil {
		return ErrClientNotFound
	}

	client.Blocked = blocked
	if blocked {
		client.Online = false
	}
	manager.changed = true

	manager.clientPublisher.Pub(ClientUpdate{Updated: client.Clone()})
	return nil
}

func (manager *ClientManager) AddPairingRequest(req *PairingRequest) error {
	if req == nil || req.ClientID == "" {
		return fmt.Errorf("client id not set")
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if req.RequestedAt.IsZero() {
		req.RequestedAt = time.Now()
	}

	client := manager.clients[req.ClientID]
	if client == nil {
		now := time.Now()
		client = &Client{
			ID:          req.ClientID,
			Name:        req.Name,
			Description: req.Description,
			FirstSeen:   now,
			LastSeen:    now,
		}
		manager.clients[req.ClientID] = client.Clone()
		manager.changed = true
		manager.clientPublisher.Pub(ClientUpdate{Updated: client.Clone()})
	}

	manager.pairingRequests[req.ClientID] = req.Clone()
	manager.pairingPublisher.Pub(PairingUpdate{Updated: req.Clone()})

	return nil
}

func (manager *ClientManager) RemovePairingRequest(clientID string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if _, found := manager.pairingRequests[clientID]; !found {
		return ErrPairingNotFound
	}

	delete(manager.pairingRequests, clientID)

	manager.pairingPublisher.Pub(PairingUpdate{Removed: &clientID})

	return nil
}

func (manager *ClientManager) FindPairingRequest(clientID string) *PairingRequest {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	req, found := manager.pairingRequests[clientID]
	if !found {
		return nil
	}
	return req.Clone()
}

func (manager *ClientManager) ApprovePairingRequest(clientID string, code string) error {
	if clientID == "" {
		return fmt.Errorf("client id not set")
	}
	if code == "" {
		return fmt.Errorf("pairing code not set")
	}

	manager.mu.Lock()
	defer manager.mu.Unlock()

	req, found := manager.pairingRequests[clientID]
	if !found {
		return ErrPairingNotFound
	}

	req.Code = code
	manager.pairingPublisher.Pub(PairingUpdate{Updated: req.Clone()})

	return nil
}

func (manager *ClientManager) load() error {
	if !manager.store.Has("clients") {
		return nil
	}

	data, err := manager.store.Get("clients")
	if err != nil {
		return err
	}

	config := struct {
		Clients map[string]*Client `json:"clients"`
	}{
		Clients: make(map[string]*Client),
	}

	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&config); err != nil {
		return err
	}

	if config.Clients == nil {
		config.Clients = make(map[string]*Client)
	}

	manager.clients = config.Clients
	manager.pairingRequests = make(map[string]*PairingRequest)

	return nil
}

func (manager *ClientManager) save() error {
	manager.mu.RLock()
	clients := make(map[string]*Client, len(manager.clients))
	for id, client := range manager.clients {
		clients[id] = client.Clone()
	}
	manager.mu.RUnlock()

	config := struct {
		Clients map[string]*Client `json:"clients"`
	}{
		Clients: clients,
	}

	data := &bytes.Buffer{}
	if err := json.NewEncoder(data).Encode(config); err != nil {
		return err
	}

	return manager.store.Set("clients", data.Bytes())
}

func (manager *ClientManager) saveIfChanged() error {
	manager.mu.Lock()
	changed := manager.changed
	if changed {
		manager.changed = false
	}
	manager.mu.Unlock()

	if changed {
		manager.log.Debugf("saving...")
		if err := manager.save(); err != nil {
			manager.mu.Lock()
			manager.changed = true
			manager.mu.Unlock()
			return err
		}
	}
	return nil
}

func (manager *ClientManager) run(ctx context.Context) {
	defer manager.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			if err := manager.saveIfChanged(); err != nil {
				manager.log.Fatalf("failed to save state: %s", err)
			}

		case lis := <-manager.clientListenerAdd:
			manager.mu.RLock()
			clients := make([]*Client, 0, len(manager.clients))
			for _, client := range manager.clients {
				clients = append(clients, client.Clone())
			}
			manager.mu.RUnlock()

			for _, client := range clients {
				manager.clientPublisher.Send(lis, ClientUpdate{Updated: client})
			}
			manager.clientPublisher.Send(lis, ClientUpdate{})

		case lis := <-manager.pairingListenerAdd:
			manager.mu.RLock()
			requests := make([]*PairingRequest, 0, len(manager.pairingRequests))
			for _, req := range manager.pairingRequests {
				requests = append(requests, req.Clone())
			}
			manager.mu.RUnlock()

			for _, req := range requests {
				manager.pairingPublisher.Send(lis, PairingUpdate{Updated: req})
			}
			manager.pairingPublisher.Send(lis, PairingUpdate{})
		}
	}
}
