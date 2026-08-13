package core

import (
	"bytes"
	"fmt"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	"github.com/jimjibone/woodhouse-core/cmd/woodhouse-core/internal/auth"
	"github.com/jimjibone/woodhouse-core/shared/random"
	"golang.org/x/crypto/argon2"
)

const (
	passwordMinSize = 8
	saltSize        = 64
)

// dummySalt is a fixed salt used only by DummyPasswordCheck, never for a
// real user's password.
var dummySalt = [saltSize]byte{}

type User struct {
	Username       string    `json:"user"`
	Fullname       string    `json:"name"`
	ResetPassword  bool      `json:"reset-password"`
	HashedPassword []byte    `json:"hash"`
	PasswordSalt   []byte    `json:"salt"`
	Role           auth.Role `json:"role"`
	// Tokens         map[string]time.Time `json:"tokens"` // key:UUID, val: expiration time
}

func NewUser(username, fullname string, password string, role auth.Role) (*User, error) {
	if len(username) == 0 {
		return nil, fmt.Errorf("username too short")
	}

	user := &User{
		Username: username,
		Fullname: fullname,
		Role:     role,
		// Tokens:   make(map[string]time.Time),
	}

	if err := user.SetPassword(password); err != nil {
		return nil, err
	}

	return user, nil
}

func (user *User) SetPassword(password string) error {
	if len(password) < passwordMinSize {
		return fmt.Errorf("password too short")
	}

	salt, err := random.GenerateRandomBytes(saltSize)
	if err != nil {
		return fmt.Errorf("cannot salt password: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 64)

	user.ResetPassword = false
	user.HashedPassword = hash
	user.PasswordSalt = salt

	return nil
}

func (user *User) IsCorrectPassword(password string) bool {
	hash := argon2.IDKey([]byte(password), user.PasswordSalt, 1, 64*1024, 4, 64)
	return bytes.Equal(hash, user.HashedPassword)
}

// DummyPasswordCheck runs argon2.IDKey with the same cost parameters as
// IsCorrectPassword, over a fixed dummy salt, and discards the result.
// Callers should invoke this on the "user not found" path of a login so
// that request timing costs the same whether or not the submitted
// username exists, closing the username-enumeration timing oracle.
func DummyPasswordCheck(password string) {
	_ = argon2.IDKey([]byte(password), dummySalt[:], 1, 64*1024, 4, 64)
}

func (user *User) Clone() *User {
	return &User{
		Username:       user.Username,
		Fullname:       user.Fullname,
		ResetPassword:  user.ResetPassword,
		HashedPassword: user.HashedPassword,
		PasswordSalt:   user.PasswordSalt,
		Role:           user.Role,
		// Tokens:         user.Tokens,
	}
}

func (user *User) Pb() *clientsapi.User {
	return &clientsapi.User{
		Username:      user.Username,
		Fullname:      user.Fullname,
		Role:          user.Role.Pb(),
		ResetPassword: user.ResetPassword,
	}
}

// func (user *User) AddToken(uuid string, exp time.Time) {
// 	if user.Tokens == nil {
// 		user.Tokens = map[string]time.Time{uuid: exp}
// 	} else {
// 		user.Tokens[uuid] = exp
// 	}
// }

// func (user *User) HasToken(uuid string) bool {
// 	if exp, found := user.Tokens[uuid]; found {
// 		if time.Now().Before(exp) {
// 			return true
// 		}
// 	}
// 	return false
// }

// func (user *User) RevokeToken(uuid string) {
// 	delete(user.Tokens, uuid)
// }

// func (user *User) CleanTokens() bool {
// 	changed := false
// 	now := time.Now()
// 	for uuid, exp := range user.Tokens {
// 		if !now.Before(exp) {
// 			delete(user.Tokens, uuid)
// 			changed = true
// 		}
// 	}
// 	return changed
// }
