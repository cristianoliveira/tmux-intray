package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStorageStubs(t *testing.T) {
	Init()

	require.Equal(t, "", AddNotification("msg", "", "", "", "", "", "info"))
	require.Equal(t, "", ListNotifications("active", "", "", "", "", "", ""))

	DismissNotification("1")
	DismissAll()
	CleanupOldNotifications(30, true)

	require.Equal(t, 0, GetActiveCount())
}
