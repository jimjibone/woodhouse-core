package auth

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// noAuthMethods are the RPCs callable without an access token: login/refresh,
// and the pairing handshake a client must complete before it has one.
var noAuthMethods = []string{
	"/woodhouse.api.v1.clients.UserAuthService/Login",
	"/woodhouse.api.v1.clients.UserAuthService/Refresh",
	"/woodhouse.api.v1.clients.UserAuthService/Logout",
	"/woodhouse.api.v1.clients.AuthService/Pair",
	"/woodhouse.api.v1.clients.AuthService/Refresh",
	"/woodhouse.api.v1.clients.AuthService/Ping",
}

func RequiresAuth(method string) bool {
	return !slices.Contains(noAuthMethods, method)
}

func IsUserMethod(method string) bool {
	return strings.HasPrefix(method, "/woodhouse.api.v1.clients.UserService/")
}

// roleMap is a map of method names (keys) and the list of roles which are allowed to access them.
var roleMap = map[string][]Role{
	"/woodhouse.api.v1.clients.UserService/GetClients":            {AdminRole, UserRole},
	"/woodhouse.api.v1.clients.UserService/ClientsStream":         {AdminRole, UserRole},
	"/woodhouse.api.v1.clients.UserService/PairingRequestsStream": {AdminRole},
	"/woodhouse.api.v1.clients.UserService/ApprovePairing":        {AdminRole},
	"/woodhouse.api.v1.clients.UserService/DenyPairing":           {AdminRole},
	"/woodhouse.api.v1.clients.UserService/UnpairClient":          {AdminRole},
	"/woodhouse.api.v1.clients.UserService/ForgetClient":          {AdminRole},
	"/woodhouse.api.v1.clients.UserService/GetDevices":            {AdminRole, UserRole},
	"/woodhouse.api.v1.clients.UserService/DevicesStream":         {AdminRole, UserRole},
	"/woodhouse.api.v1.clients.UserService/RemoveDevice":          {AdminRole},
	"/woodhouse.api.v1.clients.UserService/FavoritesStream":       {AdminRole, UserRole},
	"/woodhouse.api.v1.clients.UserService/AddFavorite":           {AdminRole, UserRole},
	"/woodhouse.api.v1.clients.UserService/RemoveFavorite":        {AdminRole, UserRole},
	"/woodhouse.api.v1.clients.UserService/GroupsStream":          {AdminRole},
	"/woodhouse.api.v1.clients.UserService/AddGroup":              {AdminRole, UserRole},
	"/woodhouse.api.v1.clients.UserService/UpdateGroup":           {AdminRole, UserRole},
	"/woodhouse.api.v1.clients.UserService/RemoveGroup":           {AdminRole, UserRole},
	"/woodhouse.api.v1.clients.UserService/SendAction":            {AdminRole, UserRole},
	"/woodhouse.api.v1.clients.UserService/SendImageRequest":      {AdminRole, UserRole},
	"/woodhouse.api.v1.clients.UserService/UsersStream":           {AdminRole, UserRole},
	"/woodhouse.api.v1.clients.UserService/AddUser":               {AdminRole},
	"/woodhouse.api.v1.clients.UserService/UpdateUser":            {AdminRole, UserRole},
	"/woodhouse.api.v1.clients.UserService/RemoveUser":            {AdminRole},
	"/woodhouse.api.v1.clients.UserService/ImagesStream":          {AdminRole, UserRole},
	"/woodhouse.api.v1.clients.UserService/GetSettings":           {AdminRole, UserRole},
	"/woodhouse.api.v1.clients.UserService/UpdateSettings":        {AdminRole},
}

func IsUserAuthorised(method string, role Role) bool {
	if roles, found := roleMap[method]; found {
		return slices.Contains(roles, role)
	}
	return false
}

// VerifyPolicyCoverage cross-checks this package's policy tables against the
// RPCs actually registered on the server, and reports every mismatch it finds.
// It catches drift in both directions:
//
//   - A policy entry naming a method that no longer exists. Harmless in itself
//     (gRPC rejects unknown methods before the interceptor runs) but it is dead
//     data that outlives the feature and reads like a live capability.
//   - A registered UserService method with no policy entry. IsUserAuthorised
//     fails closed, so this silently denies every user rather than failing
//     loudly at the point the RPC was added.
//
// methods are full gRPC method names, e.g.
// "/woodhouse.api.v1.clients.UserService/GetClients".
func VerifyPolicyCoverage(methods []string) error {
	registered := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		registered[method] = struct{}{}
	}

	var problems []string
	for method := range roleMap {
		if _, found := registered[method]; !found {
			problems = append(problems, fmt.Sprintf("role policy names unregistered method %q", method))
		}
	}
	for _, method := range noAuthMethods {
		if _, found := registered[method]; !found {
			problems = append(problems, fmt.Sprintf("unauthenticated allowlist names unregistered method %q", method))
		}
	}
	for method := range registered {
		if !IsUserMethod(method) {
			continue
		}
		if _, found := roleMap[method]; !found {
			problems = append(problems, fmt.Sprintf("registered method %q has no role policy", method))
		}
	}

	if len(problems) == 0 {
		return nil
	}
	// Map iteration order is random; sort so the error is reproducible.
	slices.Sort(problems)
	return errors.New(strings.Join(problems, "; "))
}
