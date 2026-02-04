package hooks

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitAndRunNoPanic(t *testing.T) {
	require.NotPanics(t, func() {
		Init()
		Run("pre-add", "FOO=bar")
	})
}
