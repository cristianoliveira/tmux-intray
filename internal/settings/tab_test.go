package settings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTabContract(t *testing.T) {
	assert.True(t, TabRecents.IsValid())
	assert.True(t, TabAll.IsValid())
	assert.False(t, Tab("invalid").IsValid())
	assert.Equal(t, TabRecents, DefaultTab())
}

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
		{name: "whitespace trimmed", raw: " ReCeNtS ", want: TabRecents},
		{name: "uppercase all", raw: "ALL", want: TabAll},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NormalizeTab(tt.raw))
		})
	}
}
