package stores

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFSStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewFSStore(dir)

	key := "mykey"
	value := []byte("hello world")

	if store.Has(key) {
		t.Fatalf("expected store not to have key %q before Set", key)
	}

	if err := store.Set(key, value); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if !store.Has(key) {
		t.Fatalf("expected store to have key %q after Set", key)
	}

	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, value) {
		t.Fatalf("Get returned %q, want %q", got, value)
	}

	if err := store.Del(key); err != nil {
		t.Fatalf("Del: %v", err)
	}

	if store.Has(key) {
		t.Fatalf("expected store not to have key %q after Del", key)
	}
}

func TestNewFSStoreTightensStartupPermissions(t *testing.T) {
	dir := t.TempDir()

	// Pre-seed a file with loose permissions before the store exists, and
	// loosen the directory too, to simulate an older/looser install.
	seedFile := filepath.Join(dir, "preexisting")
	if err := os.WriteFile(seedFile, []byte("secret"), 0644); err != nil {
		t.Fatalf("failed to seed file: %v", err)
	}
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatalf("failed to loosen dir permissions: %v", err)
	}

	NewFSStore(dir)

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat dir: %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0700 {
		t.Errorf("dir permissions = %o, want %o", perm, 0700)
	}

	fileInfo, err := os.Stat(seedFile)
	if err != nil {
		t.Fatalf("Stat seed file: %v", err)
	}
	if perm := fileInfo.Mode().Perm(); perm != 0600 {
		t.Errorf("seed file permissions = %o, want %o", perm, 0600)
	}
}

func TestFSStoreSetOverridesExistingLooseMode(t *testing.T) {
	dir := t.TempDir()
	store := NewFSStore(dir)

	key := "mykey"

	// Seed the file directly with a loose mode after the store has already
	// been created and tightened, to exercise the atomicfile fix (the
	// requested mode must win over any pre-existing file mode).
	path := filepath.Join(dir, key)
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatalf("failed to seed file: %v", err)
	}

	if err := store.Set(key, []byte("new")); err != nil {
		t.Fatalf("Set: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions after Set = %o, want %o", perm, 0600)
	}
}

func TestMemStoreRoundTrip(t *testing.T) {
	store := NewMemStore()

	key := "mykey"
	value := []byte("hello world")

	if store.Has(key) {
		t.Fatalf("expected store not to have key %q before Set", key)
	}

	if err := store.Set(key, value); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if !store.Has(key) {
		t.Fatalf("expected store to have key %q after Set", key)
	}

	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, value) {
		t.Fatalf("Get returned %q, want %q", got, value)
	}

	if err := store.Del(key); err != nil {
		t.Fatalf("Del: %v", err)
	}

	if store.Has(key) {
		t.Fatalf("expected store not to have key %q after Del", key)
	}
}
