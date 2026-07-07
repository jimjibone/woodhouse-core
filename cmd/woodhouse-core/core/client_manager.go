package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jimjibone/log"
	"github.com/jimjibone/queue/v2"
	"github.com/jimjibone/woodhouse-core/shared/random"
	"github.com/jimjibone/woodhouse-core/shared/stores"
)

var (
	ErrClientNotFound      = errors.New("client not found")
	ErrClientAlreadyExists = errors.New("client already exists")
	ErrPairingNotFound     = errors.New("pairing request not found")
	ErrPairingInProgress   = errors.New("a pairing request is already in progress for this client")
	ErrTooManyPairings     = errors.New("too many pending pairing requests")
)

// maxPendingPairings caps the number of concurrent in-flight pairing requests
// to bound resource use and SAS guessing.
const maxPendingPairings = 32

type ClientUpdate struct {
	Updated *Client
	Removed *string
}

type PairingUpdate struct {
	Updated *PairingRequest
	Removed *string // request_id of a removed pairing request
}

// pendingPairing tracks an in-flight pairing request and the channel used to
// signal the waiting AuthService.Pair goroutine when the user confirms or
// denies it.
type pendingPairing struct {
	req      *PairingRequest
	result   chan bool // buffered(1): true=confirmed, false=denied
	resolved bool
}

type ClientManager struct {
	log *log.Context

	wg    sync.WaitGroup
	close func()
	ctx   context.Context

	store stores.Store

	mu              sync.RWMutex
	clients         map[string]*Client
	pairingRequests map[string]*pendingPairing // key = request_id

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
		pairingRequests:    make(map[string]*pendingPairing),
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

// FindClient returns a client by ID, or nil if not found. Note that this
// returns a copy of the client,
func (manager *ClientManager) FindClient(id string) *Client {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	client, found := manager.clients[id]
	if !found {
		return nil
	}
	return client.Clone()
}

// UpdateClient updates an existing client, or creates a new client if one
// doesn't already exist.
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

// DeleteClient deletes a client by ID. Note that this does not automatically
// remove any pending pairing requests for the client, so if the client tries to
// pair again with the same ID, it will be recreated as a new client.
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

// SetClientOnline sets the online status of an existing client, and updates the last seen time.
func (manager *ClientManager) SetClientOnline(id string, online bool, lastSeen time.Time) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	client := manager.clients[id]
	if client == nil {
		return ErrClientNotFound
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

	manager.clients[id] = client.Clone()
	manager.changed = true

	manager.clientPublisher.Pub(ClientUpdate{Updated: client.Clone()})
	return nil
}

// SetClientPaired sets the paired status of an existing client.
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

// AddPairingRequest registers a new pending pairing request, assigning it a
// unique request id, and returns that id together with a channel that receives
// the user's decision (true=confirmed, false=denied). It rejects a second
// concurrent, unresolved attempt for the same client id (so an attacker cannot
// clobber a legitimate request) and caps the number of concurrent attempts.
func (manager *ClientManager) AddPairingRequest(req *PairingRequest) (string, <-chan bool, error) {
	if req == nil || req.ClientID == "" {
		return "", nil, fmt.Errorf("client id not set")
	}

	requestID, err := random.GenerateRandomString(16)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate request id: %w", err)
	}

	manager.mu.Lock()
	defer manager.mu.Unlock()

	if len(manager.pairingRequests) >= maxPendingPairings {
		return "", nil, ErrTooManyPairings
	}
	for _, p := range manager.pairingRequests {
		if p.req.ClientID == req.ClientID && !p.resolved {
			return "", nil, ErrPairingInProgress
		}
	}

	if req.RequestedAt.IsZero() {
		req.RequestedAt = time.Now()
	}
	req.RequestID = requestID
	req.Confirmed = false
	req.SAS = ""

	pending := &pendingPairing{
		req:    req.Clone(),
		result: make(chan bool, 1),
	}
	manager.pairingRequests[requestID] = pending
	manager.pairingPublisher.Pub(PairingUpdate{Updated: pending.req.Clone()})

	return requestID, pending.result, nil
}

// RemovePairingRequest removes a pending pairing request by request id. If the
// request has not yet been resolved it is signalled as denied, unblocking the
// waiting AuthService.Pair goroutine. Safe to call for cleanup after a request
// has already been confirmed or denied.
func (manager *ClientManager) RemovePairingRequest(requestID string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	pending, found := manager.pairingRequests[requestID]
	if !found {
		return ErrPairingNotFound
	}

	if !pending.resolved {
		pending.resolved = true
		pending.result <- false
	}
	delete(manager.pairingRequests, requestID)
	manager.pairingPublisher.Pub(PairingUpdate{Removed: &requestID})

	return nil
}

// SetPairingSAS records the computed SAS on a pending request and publishes the
// update so the web UI can display it for the user to compare.
func (manager *ClientManager) SetPairingSAS(requestID, sas string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	pending, found := manager.pairingRequests[requestID]
	if !found {
		return ErrPairingNotFound
	}
	pending.req.SAS = sas
	manager.pairingPublisher.Pub(PairingUpdate{Updated: pending.req.Clone()})

	return nil
}

// ConfirmPairingRequest marks a pending request as confirmed by the user and
// signals the waiting AuthService.Pair goroutine to release the credentials.
func (manager *ClientManager) ConfirmPairingRequest(requestID string) error {
	if requestID == "" {
		return fmt.Errorf("request id not set")
	}

	manager.mu.Lock()
	defer manager.mu.Unlock()

	pending, found := manager.pairingRequests[requestID]
	if !found {
		return ErrPairingNotFound
	}
	if pending.resolved {
		return ErrPairingNotFound
	}

	pending.resolved = true
	pending.req.Confirmed = true
	pending.result <- true

	return nil
}

// FinalisePairingRequest marks the pairing request as complete and creates a
// new paired client if one doesn't already exist.
func (manager *ClientManager) FinalisePairingRequest(req *PairingRequest) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	client := manager.clients[req.ClientID]
	if client == nil {
		now := time.Now()
		client = &Client{
			ID:          req.ClientID,
			Name:        req.Name,
			Description: req.Description,
			FirstSeen:   now,
			LastSeen:    now,
			Paired:      true,
		}
		manager.log.Infof("client %q added", client.ID)
		manager.clients[req.ClientID] = client.Clone()
		manager.changed = true
		manager.clientPublisher.Pub(ClientUpdate{Updated: client.Clone()})
	} else {
		manager.log.Infof("client %q added", client.ID)
		client.Paired = true
		manager.changed = true
		manager.clientPublisher.Pub(ClientUpdate{Updated: client.Clone()})
	}

}

func (manager *ClientManager) ForgetClient(id string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	client := manager.clients[id]
	if client == nil {
		return ErrClientNotFound
	}

	delete(manager.clients, id)
	manager.changed = true

	manager.clientPublisher.Pub(ClientUpdate{Removed: &id})
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
	manager.pairingRequests = make(map[string]*pendingPairing)

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
			for _, p := range manager.pairingRequests {
				requests = append(requests, p.req.Clone())
			}
			manager.mu.RUnlock()

			for _, req := range requests {
				manager.pairingPublisher.Send(lis, PairingUpdate{Updated: req})
			}
			manager.pairingPublisher.Send(lis, PairingUpdate{})
		}
	}
}
