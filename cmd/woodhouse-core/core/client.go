package core

import (
	"time"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
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

// PairingRequest represents a pending client-initiated pairing request. It is
// held in memory only (never persisted) - in particular the SAS is a
// short-lived shared authentication value and must not be written to disk.
type PairingRequest struct {
	RequestID   string
	ClientID    string
	Name        string
	Description string
	Version     string
	SAS         string
	Confirmed   bool
	RequestedAt time.Time
}

func (req *PairingRequest) Clone() *PairingRequest {
	if req == nil {
		return nil
	}
	return &PairingRequest{
		RequestID:   req.RequestID,
		ClientID:    req.ClientID,
		Name:        req.Name,
		Description: req.Description,
		Version:     req.Version,
		SAS:         req.SAS,
		Confirmed:   req.Confirmed,
		RequestedAt: req.RequestedAt,
	}
}

func (req *PairingRequest) Pb() *clientsapi.PairingRequest {
	if req == nil {
		return nil
	}

	return &clientsapi.PairingRequest{
		RequestId:   req.RequestID,
		ClientId:    req.ClientID,
		Name:        req.Name,
		Description: req.Description,
		Sas:         req.SAS,
		RequestedAt: uint64(req.RequestedAt.Unix()),
	}
}
