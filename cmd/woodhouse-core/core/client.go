package core

import (
	"time"

	clientsapi "github.com/jimjibone/woodhouse-4/api/go/v1/clients"
)

// Client holds metadata and state for a known client.
type Client struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`

	Paired bool `json:"paired"`
	Online bool `json:"online"`

	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

func (client *Client) Clone() *Client {
	if client == nil {
		return nil
	}
	return &Client{
		ID:          client.ID,
		Name:        client.Name,
		Description: client.Description,
		Version:     client.Version,
		Paired:      client.Paired,
		Online:      client.Online,
		FirstSeen:   client.FirstSeen,
		LastSeen:    client.LastSeen,
	}
}

func (client *Client) Pb() *clientsapi.Client {
	if client == nil {
		return nil
	}

	return &clientsapi.Client{
		Id:          client.ID,
		Name:        client.Name,
		Description: client.Description,
		Paired:      client.Paired,
		Online:      client.Online,
		FirstSeen:   uint64(client.FirstSeen.Unix()),
		LastSeen:    uint64(client.LastSeen.Unix()),
	}
}

// PairingRequest represents a pending client-initiated pairing request.
type PairingRequest struct {
	ClientID    string    `json:"client_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Code        string    `json:"code"`
	RequestedAt time.Time `json:"requested_at"`
}

func (req *PairingRequest) Clone() *PairingRequest {
	if req == nil {
		return nil
	}
	return &PairingRequest{
		ClientID:    req.ClientID,
		Name:        req.Name,
		Description: req.Description,
		Version:     req.Version,
		Code:        req.Code,
		RequestedAt: req.RequestedAt,
	}
}

func (req *PairingRequest) Pb() *clientsapi.PairingRequest {
	if req == nil {
		return nil
	}

	return &clientsapi.PairingRequest{
		ClientId:    req.ClientID,
		Name:        req.Name,
		Description: req.Description,
		RequestedAt: uint64(req.RequestedAt.Unix()),
	}
}
