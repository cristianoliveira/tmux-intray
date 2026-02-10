package state

import (
	"errors"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
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

func TestRuntimeCoordinatorEnsureTmuxRunning(t *testing.T) {
	tests := []struct {
		name     string
		running  bool
		err      error
		expected bool
	}{
		{name: "tmux running", running: true, err: nil, expected: true},
		{name: "tmux not running", running: false, err: nil, expected: false},
		{name: "tmux check returns error", running: true, err: errors.New("tmux unavailable"), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(tmux.MockClient)
			mockClient.On("HasSession").Return(tt.running, tt.err).Once()

			coordinator := NewRuntimeCoordinator(mockClient)
			assert.Equal(t, tt.expected, coordinator.EnsureTmuxRunning())
		})
	}
}

func TestRuntimeCoordinatorValidatePaneExists(t *testing.T) {
	mockClient := new(tmux.MockClient)
	mockClient.On("ValidatePaneExists", "$1", "@2", "%3").Return(true, nil).Once()
	mockClient.On("ValidatePaneExists", "$1", "@2", "%4").Return(false, errors.New("query failed")).Once()

	coordinator := NewRuntimeCoordinator(mockClient)

	exists, err := coordinator.ValidatePaneExists("$1", "@2", "%3")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = coordinator.ValidatePaneExists("$1", "@2", "%4")
	require.Error(t, err)
	assert.False(t, exists)
}

func TestRuntimeCoordinatorGetCurrentContextReturnsClientError(t *testing.T) {
	mockClient := new(tmux.MockClient)
	mockClient.On("GetCurrentContext").Return(tmux.TmuxContext{}, errors.New("no tmux context")).Once()

	coordinator := NewRuntimeCoordinator(mockClient)
	ctx, err := coordinator.GetCurrentContext()

	require.Error(t, err)
	assert.Nil(t, ctx)
}

func TestRuntimeCoordinatorGetCurrentContextBestEffortNames(t *testing.T) {
	mockClient := new(tmux.MockClient)
	mockClient.On("GetCurrentContext").Return(tmux.TmuxContext{
		SessionID: "$10",
		WindowID:  "@20",
		PaneID:    "%30",
		PanePID:   "1234",
	}, nil).Once()
	mockClient.On("GetSessionName", "$10").Return("", errors.New("session lookup failed")).Once()
	mockClient.On("ListSessions").Return(map[string]string{"$10": "dev"}, nil).Once()
	mockClient.On("ListWindows").Return(map[string]string{"@20": "editor"}, nil).Once()
	mockClient.On("ListPanes").Return(map[string]string{"%30": "shell"}, nil).Once()

	coordinator := NewRuntimeCoordinator(mockClient)
	ctx, err := coordinator.GetCurrentContext()

	require.NoError(t, err)
	assert.Equal(t, &model.TmuxContext{
		SessionID:   "$10",
		SessionName: "$10",
		WindowID:    "@20",
		WindowName:  "editor",
		PaneID:      "%30",
		PaneName:    "shell",
		PanePID:     "1234",
	}, ctx)
}

func TestRuntimeCoordinatorGetWindowNameRefreshPaths(t *testing.T) {
	t.Run("empty window id", func(t *testing.T) {
		coordinator := NewRuntimeCoordinator(new(tmux.MockClient))
		name, err := coordinator.GetWindowName("")
		require.NoError(t, err)
		assert.Equal(t, "", name)
	})

	t.Run("cache hit", func(t *testing.T) {
		coordinator := NewRuntimeCoordinator(new(tmux.MockClient))
		coordinator.SetWindowNames(map[string]string{"@1": "editor"})

		name, err := coordinator.GetWindowName("@1")
		require.NoError(t, err)
		assert.Equal(t, "editor", name)
	})

	t.Run("refresh error falls back to id", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("ListSessions").Return(map[string]string{}, errors.New("session refresh failed")).Once()
		mockClient.On("ListWindows").Return(map[string]string{"@2": "main"}, nil).Once()
		mockClient.On("ListPanes").Return(map[string]string{}, errors.New("pane refresh failed")).Once()

		coordinator := NewRuntimeCoordinator(mockClient)
		name, err := coordinator.GetWindowName("@2")

		require.Error(t, err)
		assert.Equal(t, "@2", name)
	})

	t.Run("refresh success resolves name", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("ListSessions").Return(map[string]string{}, nil).Once()
		mockClient.On("ListWindows").Return(map[string]string{"@3": "logs"}, nil).Once()
		mockClient.On("ListPanes").Return(map[string]string{}, nil).Once()

		coordinator := NewRuntimeCoordinator(mockClient)
		name, err := coordinator.GetWindowName("@3")

		require.NoError(t, err)
		assert.Equal(t, "logs", name)
	})

	t.Run("refresh success missing name returns id", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("ListSessions").Return(map[string]string{}, nil).Once()
		mockClient.On("ListWindows").Return(map[string]string{"@9": "misc"}, nil).Once()
		mockClient.On("ListPanes").Return(map[string]string{}, nil).Once()

		coordinator := NewRuntimeCoordinator(mockClient)
		name, err := coordinator.GetWindowName("@8")

		require.NoError(t, err)
		assert.Equal(t, "@8", name)
	})
}

func TestRuntimeCoordinatorGetPaneNameRefreshPaths(t *testing.T) {
	t.Run("empty pane id", func(t *testing.T) {
		coordinator := NewRuntimeCoordinator(new(tmux.MockClient))
		name, err := coordinator.GetPaneName("")
		require.NoError(t, err)
		assert.Equal(t, "", name)
	})

	t.Run("cache hit", func(t *testing.T) {
		coordinator := NewRuntimeCoordinator(new(tmux.MockClient))
		coordinator.SetPaneNames(map[string]string{"%1": "shell"})

		name, err := coordinator.GetPaneName("%1")
		require.NoError(t, err)
		assert.Equal(t, "shell", name)
	})

	t.Run("refresh error falls back to id", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("ListSessions").Return(map[string]string{}, errors.New("session refresh failed")).Once()
		mockClient.On("ListWindows").Return(map[string]string{}, errors.New("window refresh failed")).Once()
		mockClient.On("ListPanes").Return(map[string]string{"%2": "worker"}, nil).Once()

		coordinator := NewRuntimeCoordinator(mockClient)
		name, err := coordinator.GetPaneName("%2")

		require.Error(t, err)
		assert.Equal(t, "%2", name)
	})

	t.Run("refresh success resolves name", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("ListSessions").Return(map[string]string{}, nil).Once()
		mockClient.On("ListWindows").Return(map[string]string{}, nil).Once()
		mockClient.On("ListPanes").Return(map[string]string{"%3": "build"}, nil).Once()

		coordinator := NewRuntimeCoordinator(mockClient)
		name, err := coordinator.GetPaneName("%3")

		require.NoError(t, err)
		assert.Equal(t, "build", name)
	})

	t.Run("refresh success missing name returns id", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("ListSessions").Return(map[string]string{}, nil).Once()
		mockClient.On("ListWindows").Return(map[string]string{}, nil).Once()
		mockClient.On("ListPanes").Return(map[string]string{"%9": "misc"}, nil).Once()

		coordinator := NewRuntimeCoordinator(mockClient)
		name, err := coordinator.GetPaneName("%8")

		require.NoError(t, err)
		assert.Equal(t, "%8", name)
	})
}

func TestRuntimeCoordinatorResolveNamesFallbackOnError(t *testing.T) {
	t.Run("resolve session with empty input", func(t *testing.T) {
		coordinator := NewRuntimeCoordinator(new(tmux.MockClient))
		assert.Equal(t, "", coordinator.ResolveSessionName(""))
	})

	t.Run("resolve session falls back to id when lookup errors", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("GetSessionName", "$2").Return("", errors.New("lookup failed")).Once()

		coordinator := NewRuntimeCoordinator(mockClient)
		assert.Equal(t, "$2", coordinator.ResolveSessionName("$2"))
	})

	t.Run("resolve window falls back to id on refresh error", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("ListSessions").Return(map[string]string{}, errors.New("session refresh failed")).Once()
		mockClient.On("ListWindows").Return(map[string]string{}, errors.New("window refresh failed")).Once()
		mockClient.On("ListPanes").Return(map[string]string{}, errors.New("pane refresh failed")).Once()

		coordinator := NewRuntimeCoordinator(mockClient)
		assert.Equal(t, "@7", coordinator.ResolveWindowName("@7"))
	})

	t.Run("resolve pane falls back to id on refresh error", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("ListSessions").Return(map[string]string{}, errors.New("session refresh failed")).Once()
		mockClient.On("ListWindows").Return(map[string]string{}, errors.New("window refresh failed")).Once()
		mockClient.On("ListPanes").Return(map[string]string{}, errors.New("pane refresh failed")).Once()

		coordinator := NewRuntimeCoordinator(mockClient)
		assert.Equal(t, "%7", coordinator.ResolvePaneName("%7"))
	})
}

func TestRuntimeCoordinatorTmuxVisibilityAccessors(t *testing.T) {
	t.Run("get visibility success", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("GetTmuxVisibility").Return(true, nil).Once()

		coordinator := NewRuntimeCoordinator(mockClient)
		visible, err := coordinator.GetTmuxVisibility()

		require.NoError(t, err)
		assert.True(t, visible)
	})

	t.Run("get visibility error", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("GetTmuxVisibility").Return(false, errors.New("env unavailable")).Once()

		coordinator := NewRuntimeCoordinator(mockClient)
		visible, err := coordinator.GetTmuxVisibility()

		require.Error(t, err)
		assert.False(t, visible)
	})

	t.Run("set visibility success", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("SetTmuxVisibility", true).Return(nil).Once()

		coordinator := NewRuntimeCoordinator(mockClient)
		require.NoError(t, coordinator.SetTmuxVisibility(true))
	})

	t.Run("set visibility error", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		mockClient.On("SetTmuxVisibility", false).Return(errors.New("set failed")).Once()

		coordinator := NewRuntimeCoordinator(mockClient)
		require.Error(t, coordinator.SetTmuxVisibility(false))
	})
}
