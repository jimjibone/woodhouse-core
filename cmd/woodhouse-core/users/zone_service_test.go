package users

import "testing"

func TestValidZoneIcon(t *testing.T) {
	tests := []struct {
		name    string
		icon    string
		wantErr bool
	}{
		{name: "simple", icon: "sofa"},
		{name: "hyphenated", icon: "utensils-crossed"},
		{name: "digits", icon: "gamepad-2"},
		{name: "empty", icon: "", wantErr: true},
		{name: "uppercase", icon: "Sofa", wantErr: true},
		{name: "spaces", icon: "living room", wantErr: true},
		// The point of the check: an icon key is an identifier, not a place to
		// smuggle markup or a path into whatever renders it.
		{name: "markup", icon: "<script>", wantErr: true},
		{name: "path traversal", icon: "../../etc/passwd", wantErr: true},
		{name: "too long", icon: string(make([]byte, 65)), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validZoneIcon(tt.icon)
			if tt.wantErr && err == nil {
				t.Errorf("validZoneIcon(%q): got nil, want an error", tt.icon)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validZoneIcon(%q): got %s, want nil", tt.icon, err)
			}
		})
	}
}
