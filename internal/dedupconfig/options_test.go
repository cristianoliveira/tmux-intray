package dedupconfig

import (
	"testing"
	"time"

	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/dedup"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	config.Load()
	opts := Load()
	require.Equal(t, dedup.CriteriaMessage, opts.Criteria)
	require.Equal(t, time.Duration(0), opts.Window)
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("TMUX_INTRAY_DEDUP__CRITERIA", "message_source")
	t.Setenv("TMUX_INTRAY_DEDUP__WINDOW", "75s")
	config.Load()
	opts := Load()
	require.Equal(t, dedup.CriteriaMessageSource, opts.Criteria)
	require.Equal(t, 75*time.Second, opts.Window)
}
