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
	assert.Equal(t, TabRecents, NormalizeTab(""))
	assert.Equal(t, TabRecents, NormalizeTab("invalid"))
	assert.Equal(t, TabRecents, NormalizeTab(" ReCeNtS "))
	assert.Equal(t, TabAll, NormalizeTab("ALL"))
}
