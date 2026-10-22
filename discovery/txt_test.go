package discovery

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"testing"
)

func TestTXT(t *testing.T) {
	der := []byte("example certificate bytes")
	sum := sha256.Sum256(der)
	fp := "fp=" + hex.EncodeToString(sum[:])

	tests := []struct {
		name      string
		appliance bool
		hasAdmin  bool
		want      []string
	}{
		{name: "appliance without admin", appliance: true, hasAdmin: false, want: []string{fp, "setup=1"}},
		{name: "appliance with admin", appliance: true, hasAdmin: true, want: []string{fp}},
		// Outside appliance mode the first login creates the admin, so there
		// is no adoption to announce.
		{name: "plain core without admin", appliance: false, hasAdmin: false, want: []string{fp}},
		{name: "plain core with admin", appliance: false, hasAdmin: true, want: []string{fp}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TXT(der, tt.appliance, tt.hasAdmin)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("TXT = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTXTFingerprintFormat(t *testing.T) {
	got := TXT([]byte{0x00, 0x01, 0xff}, false, true)
	if len(got) != 1 {
		t.Fatalf("TXT = %q, want a single fp entry", got)
	}
	hexPart, ok := strings.CutPrefix(got[0], "fp=")
	if !ok {
		t.Fatalf("entry %q has no fp= prefix", got[0])
	}
	if len(hexPart) != 64 || hexPart != strings.ToLower(hexPart) {
		t.Fatalf("fingerprint %q is not 64 lowercase hex characters", hexPart)
	}
	if _, err := hex.DecodeString(hexPart); err != nil {
		t.Fatalf("fingerprint %q is not hex: %v", hexPart, err)
	}
}

func TestTXTFingerprintTracksCertificate(t *testing.T) {
	a := TXT([]byte("cert a"), false, true)
	b := TXT([]byte("cert b"), false, true)
	if slices.Equal(a, b) {
		t.Fatalf("different certificates gave the same TXT %q", a)
	}
}
