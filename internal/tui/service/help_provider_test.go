package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultHelpProvider(t *testing.T) {
	provider := NewDefaultHelpProvider()
	assert.NotNil(t, provider)
}

func TestGetHelp(t *testing.T) {
	provider := NewDefaultHelpProvider()

	// Test valid commands
	help := provider.GetHelp("q")
	assert.NotEmpty(t, help)
	assert.Contains(t, help, "q")
	assert.Contains(t, help, "Quit")

	help = provider.GetHelp("group-by")
	assert.NotEmpty(t, help)
	assert.Contains(t, help, "group-by")
	assert.Contains(t, help, "grouping")
	assert.Contains(t, help, "Usage")

	help = provider.GetHelp("expand-level")
	assert.NotEmpty(t, help)
	assert.Contains(t, help, "expand-level")
	assert.Contains(t, help, "expansion")

	help = provider.GetHelp("toggle-view")
	assert.NotEmpty(t, help)
	assert.Contains(t, help, "toggle-view")
	assert.Contains(t, help, "view modes")

	// Test invalid command
	help = provider.GetHelp("nonexistent")
	assert.Empty(t, help)
}

func TestGetAllHelp(t *testing.T) {
	provider := NewDefaultHelpProvider()

	helps := provider.GetAllHelp()
	assert.Len(t, helps, 5) // q, w, group-by, expand-level, toggle-view

	// Check for each command
	commandNames := make([]string, len(helps))
	for i, help := range helps {
		commandNames[i] = help.Name
	}

	assert.Contains(t, commandNames, "q")
	assert.Contains(t, commandNames, "w")
	assert.Contains(t, commandNames, "group-by")
	assert.Contains(t, commandNames, "expand-level")
	assert.Contains(t, commandNames, "toggle-view")

	// Check that each help has the expected fields
	for _, help := range helps {
		assert.NotEmpty(t, help.Name)
		assert.NotEmpty(t, help.Description)
		assert.NotEmpty(t, help.Usage)
	}
}
