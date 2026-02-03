package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadAndGet(t *testing.T) {
	Load()

	got := Get("missing", "default")
	require.Equal(t, "default", got)
}
