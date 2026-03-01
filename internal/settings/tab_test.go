package settings

import "testing"

func TestNormalizeTab(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want Tab
	}{
		{name: "recents is valid", raw: string(TabRecents), want: TabRecents},
		{name: "all is valid", raw: string(TabAll), want: TabAll},
		{name: "empty defaults", raw: "", want: TabRecents},
		{name: "invalid defaults", raw: "invalid", want: TabRecents},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeTab(tt.raw); got != tt.want {
				t.Fatalf("NormalizeTab(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
