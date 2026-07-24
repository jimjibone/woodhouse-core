package auth

import (
	"slices"
	"strings"
	"testing"

	clientsapi "github.com/jimjibone/woodhouse-api/go/v1/clients"
	"google.golang.org/grpc"
)

// registeredMethods registers the real services on a real server and derives
// the method list exactly as main.go does, so this covers the derivation as
// well as the policy tables. The service implementations are nil because only
// the descriptors are read.
func registeredMethods(t *testing.T) []string {
	t.Helper()

	server := grpc.NewServer()
	clientsapi.RegisterAuthServiceServer(server, nil)
	clientsapi.RegisterClientServiceServer(server, nil)
	clientsapi.RegisterUserServiceServer(server, nil)
	clientsapi.RegisterUserAuthServiceServer(server, nil)

	var methods []string
	for name, info := range server.GetServiceInfo() {
		for _, method := range info.Methods {
			methods = append(methods, "/"+name+"/"+method.Name)
		}
	}
	if len(methods) == 0 {
		t.Fatal("no methods registered")
	}
	return methods
}

func TestVerifyPolicyCoverage_MatchesRegisteredMethods(t *testing.T) {
	if err := VerifyPolicyCoverage(registeredMethods(t)); err != nil {
		t.Fatalf("policy does not match the registered API: %v", err)
	}
}

func TestVerifyPolicyCoverage_StalePolicyEntry(t *testing.T) {
	// Drop a method the role policy still names — the shape BlockClient and
	// UnblockClient were left in after the feature was removed.
	removed := "/woodhouse.api.v1.clients.UserService/UnpairClient"
	methods := slices.DeleteFunc(registeredMethods(t), func(method string) bool {
		return method == removed
	})

	err := VerifyPolicyCoverage(methods)
	if err == nil {
		t.Fatal("expected an error for a policy entry naming an unregistered method")
	}
	if !strings.Contains(err.Error(), removed) {
		t.Errorf("error %q does not mention the stale method %q", err, removed)
	}
}

func TestVerifyPolicyCoverage_StaleNoAuthEntry(t *testing.T) {
	removed := "/woodhouse.api.v1.clients.AuthService/Ping"
	methods := slices.DeleteFunc(registeredMethods(t), func(method string) bool {
		return method == removed
	})

	err := VerifyPolicyCoverage(methods)
	if err == nil {
		t.Fatal("expected an error for an allowlist entry naming an unregistered method")
	}
	if !strings.Contains(err.Error(), removed) {
		t.Errorf("error %q does not mention the stale method %q", err, removed)
	}
}

func TestVerifyPolicyCoverage_UserMethodWithoutPolicy(t *testing.T) {
	// A newly added UserService RPC that nobody wrote a policy for. Every role
	// is denied by IsUserAuthorised's fail-closed default, so without this
	// check the RPC just quietly never works.
	added := "/woodhouse.api.v1.clients.UserService/SomeNewMethod"
	methods := append(registeredMethods(t), added)

	err := VerifyPolicyCoverage(methods)
	if err == nil {
		t.Fatal("expected an error for a registered user method with no policy")
	}
	if !strings.Contains(err.Error(), added) {
		t.Errorf("error %q does not mention the unpoliced method %q", err, added)
	}
}

func TestVerifyPolicyCoverage_IgnoresNonUserMethods(t *testing.T) {
	// Client and auth RPCs are not role-gated, so they need no policy entry.
	methods := append(registeredMethods(t), "/woodhouse.api.v1.clients.ClientService/SomeNewMethod")

	if err := VerifyPolicyCoverage(methods); err != nil {
		t.Fatalf("non-user methods should not require a role policy: %v", err)
	}
}

func TestRequiresAuth(t *testing.T) {
	for _, method := range noAuthMethods {
		if RequiresAuth(method) {
			t.Errorf("%q should not require auth", method)
		}
	}
	for _, method := range []string{
		"/woodhouse.api.v1.clients.UserService/GetClients",
		"/woodhouse.api.v1.clients.ClientService/StatusStream",
		"/woodhouse.api.v1.clients.UserService/UnknownMethod",
	} {
		if !RequiresAuth(method) {
			t.Errorf("%q should require auth", method)
		}
	}
}
