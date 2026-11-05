package core

import (
	"testing"

	"github.com/jimjibone/woodhouse-core/shared/stores"
)

var untouchedKeys = []string{"users", "clients", "invites", "notifications.json", "clients-jwt", "users-jwt"}

func seedStore(t *testing.T, store stores.Store, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if err := store.Set(key, []byte("data-"+key)); err != nil {
			t.Fatalf("Set %s: %s", key, err)
		}
	}
}

func TestWipeV1State(t *testing.T) {
	store := stores.NewMemStore()
	seedStore(t, store, v1StateKeys...)
	seedStore(t, store, untouchedKeys...)

	if err := WipeV1State(store); err != nil {
		t.Fatalf("WipeV1State: %s", err)
	}
	for _, key := range v1StateKeys {
		if store.Has(key) {
			t.Errorf("%s was not wiped", key)
		}
	}
	for _, key := range untouchedKeys {
		if got, err := store.Get(key); err != nil || string(got) != "data-"+key {
			t.Errorf("%s = %q, %v, want it untouched", key, got, err)
		}
	}
	if got, err := store.Get(schemaVersionKey); err != nil || string(got) != "2" {
		t.Errorf("%s = %q, %v, want 2", schemaVersionKey, got, err)
	}

	// The second start keeps whatever the managers saved since.
	seedStore(t, store, "devices.json", "zones.json", "groups.json", endpointLabelsKey)
	if err := WipeV1State(store); err != nil {
		t.Fatalf("WipeV1State again: %s", err)
	}
	for _, key := range []string{"devices.json", "zones.json", "groups.json", endpointLabelsKey} {
		if !store.Has(key) {
			t.Errorf("%s was wiped on the second start", key)
		}
	}
	for _, key := range untouchedKeys {
		if !store.Has(key) {
			t.Errorf("%s was lost on the second start", key)
		}
	}
}

// A store with some of the keys, or none (a new install), is recorded as
// wiped too.
func TestWipeV1StatePartialAndEmpty(t *testing.T) {
	store := stores.NewMemStore()
	seedStore(t, store, "groups.json")
	if err := WipeV1State(store); err != nil {
		t.Fatalf("WipeV1State: %s", err)
	}
	if store.Has("groups.json") {
		t.Error("groups.json was not wiped")
	}

	fresh := stores.NewMemStore()
	if err := WipeV1State(fresh); err != nil {
		t.Fatalf("WipeV1State on an empty store: %s", err)
	}
	if got, err := fresh.Get(schemaVersionKey); err != nil || string(got) != "2" {
		t.Errorf("%s = %q, %v, want 2", schemaVersionKey, got, err)
	}
}

// A lower or unreadable version wipes; a higher one is left alone.
func TestWipeV1StateVersions(t *testing.T) {
	for _, tc := range []struct {
		version string
		wipe    bool
	}{
		{"1", true},
		{"0", true},
		{"garbage", true},
		{"2", false},
		{"3", false},
	} {
		store := stores.NewMemStore()
		seedStore(t, store, "devices.json")
		if err := store.Set(schemaVersionKey, []byte(tc.version)); err != nil {
			t.Fatalf("Set: %s", err)
		}
		if err := WipeV1State(store); err != nil {
			t.Fatalf("version %q: %s", tc.version, err)
		}
		if wiped := !store.Has("devices.json"); wiped != tc.wipe {
			t.Errorf("version %q: wiped = %t, want %t", tc.version, wiped, tc.wipe)
		}
		if tc.wipe {
			if got, _ := store.Get(schemaVersionKey); string(got) != "2" {
				t.Errorf("version %q: recorded %q, want 2", tc.version, got)
			}
		}
	}
}
