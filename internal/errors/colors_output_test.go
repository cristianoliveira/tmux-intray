package errors

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = oldStdout })

	fn()

	require.NoError(t, w.Close())
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	return buf.String()
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = oldStderr })

	fn()

	require.NoError(t, w.Close())
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	return buf.String()
}

func TestColorsOutputError(t *testing.T) {
	output := captureStderr(t, func() {
		(&ColorsOutput{}).Error("adapter", "error")
	})

	assert.Contains(t, output, "Error:")
	assert.Contains(t, output, "adapter error")
}

func TestColorsOutputWarning(t *testing.T) {
	output := captureStderr(t, func() {
		(&ColorsOutput{}).Warning("adapter", "warning")
	})

	assert.Contains(t, output, "Warning:")
	assert.Contains(t, output, "adapter warning")
}

func TestColorsOutputInfo(t *testing.T) {
	output := captureStdout(t, func() {
		(&ColorsOutput{}).Info("adapter", "info")
	})

	assert.Contains(t, output, "adapter info")
}

func TestColorsOutputSuccess(t *testing.T) {
	output := captureStdout(t, func() {
		(&ColorsOutput{}).Success("adapter", "success")
	})

	assert.True(t, strings.Contains(output, "adapter success"))
}

func TestNewDefaultCLIHandlerUsesColorsOutput(t *testing.T) {
	handler := NewDefaultCLIHandler()

	require.NotNil(t, handler)
	_, ok := handler.colors.(*ColorsOutput)
	assert.True(t, ok, "default CLI handler should use ColorsOutput")
}
