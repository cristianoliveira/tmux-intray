package colors

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestColorFunctionsNoPanic(t *testing.T) {
	require.NotPanics(t, func() {
		Error("problem")
		Success("ok")
		Warning("warn")
		Info("info")
		LogInfo("log")
		Debug("debug")
	})
}
