package core

import (
	"time"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
)

// Invite is a pending offer of an account. The token that redeems it is never
// stored, only its hash: whoever can read the store (a backup, a stolen disk)
// then cannot redeem an invite that has not been used yet.
type Invite struct {
	ID        string    `json:"id"`
	TokenHash []byte    `json:"token-hash"` // SHA-256 of the token
	Username  string    `json:"user"`
	Fullname  string    `json:"name"`
	Role      auth.Role `json:"role"`
	CreatedBy string    `json:"created-by"`
	CreatedAt time.Time `json:"created-at"`
	ExpiresAt time.Time `json:"expires-at"`
}

func (invite *Invite) expired(now time.Time) bool {
	return !now.Before(invite.ExpiresAt)
}

func (invite *Invite) Clone() *Invite {
	clone := *invite
	clone.TokenHash = append([]byte(nil), invite.TokenHash...)
	return &clone
}

func (invite *Invite) Pb() *clientsapi.Invite {
	return &clientsapi.Invite{
		Id:        invite.ID,
		Username:  invite.Username,
		Fullname:  invite.Fullname,
		Role:      invite.Role.Pb(),
		CreatedBy: invite.CreatedBy,
		CreatedAt: invite.CreatedAt.UnixMilli(),
		ExpiresAt: invite.ExpiresAt.UnixMilli(),
	}
}
