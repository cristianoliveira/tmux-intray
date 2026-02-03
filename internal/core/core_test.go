package core

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCoreStubs(t *testing.T) {
	require.False(t, EnsureTmuxRunning())

	ctx := GetCurrentTmuxContext()
	require.Equal(t, TmuxContext{}, ctx)

	require.False(t, ValidatePaneExists("session", "window", "pane"))
	require.False(t, JumpToPane("session", "window", "pane"))

	require.Equal(t, "", GetTrayItems("active"))

	AddTrayItem("item", "session", "window", "pane", "created", false, "info")
	ClearTrayItems()
	SetVisibility("1")

	require.Equal(t, "", GetVisibility())
}
