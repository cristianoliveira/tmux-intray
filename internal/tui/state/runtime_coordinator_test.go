package state

import (
	"errors"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeCoordinatorListSessionsCachesResults(t *testing.T) {
	mockClient := new(tmux.MockClient)
	mockClient.On("ListSessions").Return(map[string]string{"$1": "dev"}, nil).Once()

	coordinator := NewRuntimeCoordinator(mockClient)

	first, err := coordinator.ListSessions()
	require.NoError(t, err)
	second, err := coordinator.ListSessions()
	require.NoError(t, err)

	assert.Equal(t, map[string]string{"$1": "dev"}, first)
	assert.Equal(t, map[string]string{"$1": "dev"}, second)
	mockClient.AssertNumberOfCalls(t, "ListSessions", 1)
}

func TestRuntimeCoordinatorRefreshNamesPreservesCachedDataOnError(t *testing.T) {
	mockClient := new(tmux.MockClient)
	mockClient.On("ListSessions").Return(map[string]string{"$1": "one"}, nil).Once()
	mockClient.On("ListWindows").Return(map[string]string{"@1": "editor"}, nil).Once()
	mockClient.On("ListPanes").Return(map[string]string{"%1": "shell"}, nil).Once()

	mockClient.On("ListSessions").Return(map[string]string{}, errors.New("session failure")).Once()
	mockClient.On("ListWindows").Return(map[string]string{"@1": "editor-2"}, nil).Once()
	mockClient.On("ListPanes").Return(map[string]string{"%1": "shell-2"}, nil).Once()

	coordinator := NewRuntimeCoordinator(mockClient)
	require.NoError(t, coordinator.RefreshNames())
	err := coordinator.RefreshNames()
	require.Error(t, err)

	sessions, err := coordinator.ListSessions()
	require.NoError(t, err)
	windows, err := coordinator.ListWindows()
	require.NoError(t, err)
	panes, err := coordinator.ListPanes()
	require.NoError(t, err)

	assert.Equal(t, map[string]string{"$1": "one"}, sessions)
	assert.Equal(t, map[string]string{"@1": "editor-2"}, windows)
	assert.Equal(t, map[string]string{"%1": "shell-2"}, panes)
}

func TestRuntimeCoordinatorGetSessionNameCachesLookup(t *testing.T) {
	mockClient := new(tmux.MockClient)
	mockClient.On("GetSessionName", "$1").Return("session-one", nil).Once()

	coordinator := NewRuntimeCoordinator(mockClient)
	name, err := coordinator.GetSessionName("$1")
	require.NoError(t, err)
	assert.Equal(t, "session-one", name)

	name, err = coordinator.GetSessionName("$1")
	require.NoError(t, err)
	assert.Equal(t, "session-one", name)

	mockClient.AssertNumberOfCalls(t, "GetSessionName", 1)
}
