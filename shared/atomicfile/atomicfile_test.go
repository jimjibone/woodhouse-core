package atomicfile

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileFreshEndsAtRequestedMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fresh")

	if err := WriteFile(path, 0600, bytes.NewReader([]byte("hello"))); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("permissions = %o, want %o", perm, 0600)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("content = %q, want %q", got, "hello")
	}
}

func TestWriteFileOverExistingIgnoresPreviousMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing")

	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatalf("seed WriteFile: %v", err)
	}

	if err := WriteFile(path, 0600, bytes.NewReader([]byte("new"))); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("permissions = %o, want %o (requested mode should be authoritative)", perm, 0600)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "new" {
		t.Errorf("content = %q, want %q", got, "new")
	}
}
