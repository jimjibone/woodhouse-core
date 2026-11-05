package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jimjibone/woodhouse-api/schema"
)

// TestGeneratedFileUpToDate fails when the checked-in webui constants differ
// from what the generator produces for the embedded schema.
func TestGeneratedFileUpToDate(t *testing.T) {
	s := schema.MustLoad()
	want, err := generate(s)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	path := filepath.Join("..", "..", "webui", "src", "lib", "schema", "zz_services.ts")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run task webui:schema)", path, err)
	}

	wantVersion := fmt.Sprintf("export const SCHEMA_VERSION = %d;\n", s.Version)
	if !strings.Contains(string(got), wantVersion) {
		t.Errorf("%s: schema version out of date, want %q: run task webui:schema", path, strings.TrimSpace(wantVersion))
	}
	if string(got) != string(want) {
		t.Errorf("%s is out of date: run task webui:schema", path)
	}
}

func TestGenerateASCII(t *testing.T) {
	src, err := generate(schema.MustLoad())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for i, c := range src {
		if c > 0x7e || (c < 0x20 && c != '\n' && c != '\t') {
			t.Fatalf("non-ASCII or control byte 0x%02x at offset %d", c, i)
		}
	}
}
