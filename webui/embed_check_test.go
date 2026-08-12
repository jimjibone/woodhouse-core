package webui

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// Every file the build produced must actually be in the embedded FS. Go's
// default embed walk drops "_"/"." prefixed names, which Vite does generate.
func TestEmbedIncludesEveryBuiltFile(t *testing.T) {
	var missing []string
	err := filepath.WalkDir("build", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if _, statErr := fs.Stat(Content, filepath.ToSlash(path)); statErr != nil {
			missing = append(missing, path)
		}
		return nil
	})
	if os.IsNotExist(err) {
		t.Skip("no build directory")
	}
	if err != nil {
		t.Fatalf("walk: %s", err)
	}
	for _, m := range missing {
		t.Errorf("built file not embedded: %s", m)
	}
}
