package state

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupStorage(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	t.Setenv("TMUX_INTRAY_STATE_DIR", tmpDir)
	storage.Reset()
	require.NoError(t, storage.Init())

	return tmpDir
}

func setupConfig(t *testing.T, dir string) {
	t.Helper()

	t.Setenv("TMUX_INTRAY_CONFIG_DIR", dir)
}

func stubSessionFetchers(t *testing.T) *tmux.MockClient {
	t.Helper()

	mockClient := new(tmux.MockClient)
	// Mock ListSessions to return empty map
	mockClient.On("ListSessions").Return(map[string]string{}, nil)

	return mockClient
}

func TestNewModelInitialState(t *testing.T) {
	setupStorage(t)
	mockClient := stubSessionFetchers(t)

	model, err := NewModel(mockClient)

	require.NoError(t, err)
	assert.Equal(t, 0, model.width)
	assert.Equal(t, 0, model.height)
	assert.Equal(t, 0, model.cursor)
	assert.False(t, model.searchMode)
	assert.Equal(t, "", model.searchQuery)
	assert.Empty(t, model.notifications)
	assert.Empty(t, model.filtered)
	assert.NotNil(t, model.expansionState)
	assert.Empty(t, model.expansionState)
	assert.Nil(t, model.treeRoot)
	assert.Empty(t, model.visibleNodes)
}

func TestModelGroupedModeBuildsVisibleNodes(t *testing.T) {
	model := &Model{
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
			{ID: 2, Session: "$2", Window: "@1", Pane: "%2", Message: "Two"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
	}

	model.applySearchFilter()

	require.NotNil(t, model.treeRoot)
	require.Len(t, model.visibleNodes, 2)
	assert.Equal(t, NodeKindNotification, model.visibleNodes[0].Kind)
	assert.Equal(t, NodeKindNotification, model.visibleNodes[1].Kind)
}

func TestModelSwitchesViewModes(t *testing.T) {
	model := &Model{
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "One"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
	}

	model.applySearchFilter()
	require.NotNil(t, model.treeRoot)
	require.NotEmpty(t, model.visibleNodes)

	model.viewMode = "flat"
	model.applySearchFilter()
	assert.Nil(t, model.treeRoot)
	assert.Empty(t, model.visibleNodes)
}

func TestModelSelectedNotificationGroupedView(t *testing.T) {
	model := &Model{
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "b", Window: "@1", Pane: "%1", Message: "B"},
			{ID: 2, Session: "a", Window: "@1", Pane: "%1", Message: "A"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
	}

	model.applySearchFilter()
	model.cursor = 0

	selected, ok := model.selectedNotification()

	require.True(t, ok)
	assert.Equal(t, "a", selected.Session)
}

func TestModelInitReturnsNil(t *testing.T) {
	model := &Model{}

	cmd := model.Init()

	assert.Nil(t, cmd)
}

func TestModelUpdateHandlesNavigation(t *testing.T) {
	stubSessionFetchers(t)

	model := &Model{
		notifications: []notification.Notification{
			{ID: 1, Message: "First"},
			{ID: 2, Message: "Second"},
			{ID: 3, Message: "Third"},
		},
		filtered: []notification.Notification{
			{ID: 1, Message: "First"},
			{ID: 2, Message: "Second"},
			{ID: 3, Message: "Third"},
		},
		cursor:   0,
		width:    80,
		viewport: viewport.New(80, 22),
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updated, _ := model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 0, model.cursor)

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 1, model.cursor)

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 2, model.cursor)

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 2, model.cursor)

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 1, model.cursor)
}

func TestModelUpdateHandlesSearch(t *testing.T) {
	stubSessionFetchers(t)

	model := &Model{
		notifications: []notification.Notification{
			{ID: 1, Message: "Error: file not found"},
			{ID: 2, Message: "Warning: low memory"},
			{ID: 3, Message: "Error: connection failed"},
		},
		filtered: []notification.Notification{
			{ID: 1, Message: "Error: file not found"},
			{ID: 2, Message: "Warning: low memory"},
			{ID: 3, Message: "Error: connection failed"},
		},
		width:    80,
		viewport: viewport.New(80, 22),
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updated, _ := model.Update(msg)
	model = updated.(*Model)

	assert.True(t, model.searchMode)
	assert.Equal(t, "", model.searchQuery)
	assert.Equal(t, 0, model.cursor)
	assert.Len(t, model.filtered, 3)

	model.searchQuery = "error"
	model.applySearchFilter()

	require.Len(t, model.filtered, 2)
	assert.True(t, strings.Contains(model.filtered[0].Message, "Error"))

	model.searchQuery = "not found"
	model.applySearchFilter()

	require.Len(t, model.filtered, 1)
	assert.True(t, strings.Contains(strings.ToLower(model.filtered[0].Message), "not found"))

	model.searchQuery = ""
	model.applySearchFilter()

	assert.Len(t, model.filtered, 3)
}

func TestModelUpdateHandlesQuit(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(msg)
	assert.NotNil(t, cmd)

	msg = tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd = model.Update(msg)
	assert.NotNil(t, cmd)

	msg = tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd = model.Update(msg)
	assert.NotNil(t, cmd)
}

func TestModelUpdateHandlesSearchEscape(t *testing.T) {
	model := &Model{searchMode: true, searchQuery: "test"}

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, cmd := model.Update(msg)
	model = updated.(*Model)

	assert.Nil(t, cmd)
	assert.False(t, model.searchMode)
	assert.Equal(t, "", model.searchQuery)
}

func TestModelUpdateHandlesCommandMode(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}}
	updated, _ := model.Update(msg)
	model = updated.(*Model)

	assert.True(t, model.commandMode)
	assert.Equal(t, "", model.commandQuery)

	model.commandMode = true
	model.commandQuery = "test"

	msg = tea.KeyMsg{Type: tea.KeyEsc}
	updated, cmd := model.Update(msg)
	model = updated.(*Model)

	assert.Nil(t, cmd)
	assert.False(t, model.commandMode)
	assert.Equal(t, "", model.commandQuery)

	model.commandMode = true
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, "q", model.commandQuery)

	msg = tea.KeyMsg{Type: tea.KeyBackspace}
	updated, _ = model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, "", model.commandQuery)

	model.commandMode = true
	model.commandQuery = "q"
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd = model.Update(msg)

	assert.NotNil(t, cmd)
}

func TestModelUpdateHandlesWindowSize(t *testing.T) {
	model := &Model{}

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	updated, _ := model.Update(msg)
	model = updated.(*Model)

	assert.Equal(t, 100, model.width)
	assert.Equal(t, 30, model.height)
	assert.Equal(t, 28, model.viewport.Height)
}

func TestModelViewRendersContent(t *testing.T) {
	stubSessionFetchers(t)

	model := &Model{
		notifications: []notification.Notification{
			{ID: 1, Message: "Test notification", Timestamp: "2024-01-01T12:00:00Z", Level: "info", State: "active"},
		},
		filtered: []notification.Notification{
			{ID: 1, Message: "Test notification", Timestamp: "2024-01-01T12:00:00Z", Level: "info", State: "active"},
		},
		cursor:   0,
		width:    80,
		height:   24,
		viewport: viewport.New(80, 22),
	}
	model.updateViewportContent()

	view := model.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "TYPE")
	assert.Contains(t, view, "STATUS")
	assert.Contains(t, view, "SESSION")
	assert.Contains(t, view, "MESSAGE")
	assert.Contains(t, view, "PANE")
	assert.Contains(t, view, "AGE")
	assert.Contains(t, view, "Test notification")
	assert.Contains(t, view, "j/k: move")
	assert.Contains(t, view, "q: quit")
}

func TestModelViewWithNoNotifications(t *testing.T) {
	model := &Model{
		notifications: []notification.Notification{},
		filtered:      []notification.Notification{},
		width:         80,
		height:        24,
		viewport:      viewport.New(80, 22),
	}
	model.updateViewportContent()

	view := model.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "No notifications found")
}

func TestHandleDismiss(t *testing.T) {
	setupStorage(t)
	mockClient := stubSessionFetchers(t)

	id, err := storage.AddNotification("Test message", "2024-01-01T12:00:00Z", "", "", "", "1234", "info")
	require.NoError(t, err)
	require.NotEmpty(t, id)

	model, err := NewModel(mockClient)
	require.NoError(t, err)
	require.Len(t, model.filtered, 1)

	cmd := model.handleDismiss()

	assert.Nil(t, cmd)

	model, err = NewModel(mockClient)
	require.NoError(t, err)
	assert.Empty(t, model.filtered)
}

func TestHandleDismissGroupedViewUsesVisibleNodes(t *testing.T) {
	setupStorage(t)
	mockClient := stubSessionFetchers(t)

	_, err := storage.AddNotification("B msg", "2024-02-02T12:00:00Z", "b", "@1", "%1", "", "info")
	require.NoError(t, err)
	_, err = storage.AddNotification("A msg", "2024-01-01T12:00:00Z", "a", "@1", "%1", "", "info")
	require.NoError(t, err)

	model, err := NewModel(mockClient)
	require.NoError(t, err)
	model.viewMode = viewModeGrouped
	model.applySearchFilter()
	model.cursor = 0

	cmd := model.handleDismiss()

	assert.Nil(t, cmd)

	lines, err := storage.ListNotifications("active", "", "", "", "", "", "")
	require.NoError(t, err)

	remainingSessions := []string{}
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		notif, err := notification.ParseNotification(line)
		require.NoError(t, err)
		remainingSessions = append(remainingSessions, notif.Session)
	}

	require.Len(t, remainingSessions, 1)
	assert.Equal(t, "b", remainingSessions[0])
}

func TestHandleDismissWithEmptyList(t *testing.T) {
	model := &Model{
		notifications: []notification.Notification{},
		filtered:      []notification.Notification{},
		cursor:        0,
	}

	cmd := model.handleDismiss()

	assert.Nil(t, cmd)
}

func TestHandleJumpWithMissingContext(t *testing.T) {
	model := &Model{
		notifications: []notification.Notification{
			{ID: 1, Message: "Test"},
		},
		filtered: []notification.Notification{
			{ID: 1, Message: "Test"},
		},
		cursor: 0,
	}

	cmd := model.handleJump()
	assert.Nil(t, cmd)

	model.filtered[0].Session = "$1"
	cmd = model.handleJump()
	assert.Nil(t, cmd)

	model.filtered[0].Window = "@2"
	model.filtered[0].Pane = ""
	cmd = model.handleJump()
	assert.Nil(t, cmd)
}

func TestHandleJumpGroupedViewUsesVisibleNodes(t *testing.T) {
	model := &Model{
		viewMode: viewModeGrouped,
		notifications: []notification.Notification{
			{ID: 1, Session: "b", Window: "@1", Pane: "%1", Message: "B"},
			{ID: 2, Session: "a", Window: "", Pane: "%1", Message: "A"},
		},
		viewport: viewport.New(80, 22),
		width:    80,
		ensureTmuxRunning: func() bool {
			t.Fatal("ensureTmuxRunning should not be called")
			return true
		},
		jumpToPane: func(sessionID, windowID, paneID string) bool {
			t.Fatal("jumpToPane should not be called")
			return true
		},
	}

	model.applySearchFilter()
	model.cursor = 0

	cmd := model.handleJump()

	assert.Nil(t, cmd)
}

func TestHandleJumpWithEmptyList(t *testing.T) {
	model := &Model{
		notifications: []notification.Notification{},
		filtered:      []notification.Notification{},
		cursor:        0,
	}

	cmd := model.handleJump()

	assert.Nil(t, cmd)
}

func TestModelUpdateHandlesDismissKey(t *testing.T) {
	model := &Model{
		notifications: []notification.Notification{},
		filtered:      []notification.Notification{},
		cursor:        0,
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	updated, _ := model.Update(msg)

	assert.NotNil(t, updated.(*Model))
}

func TestModelUpdateHandlesEnterKey(t *testing.T) {
	model := &Model{
		notifications: []notification.Notification{},
		filtered:      []notification.Notification{},
		cursor:        0,
	}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, _ := model.Update(msg)

	assert.NotNil(t, updated.(*Model))
}

func TestGetSessionNameCachesFetcher(t *testing.T) {
	model := &Model{
		sessionNames: map[string]string{
			"$1": "$1-name",
		},
	}

	name := model.getSessionName("$1")
	assert.Equal(t, "$1-name", name)

	// Call again - should return cached value
	name = model.getSessionName("$1")
	assert.Equal(t, "$1-name", name)
}

func TestToState(t *testing.T) {
	tests := []struct {
		name  string
		model *Model
		want  settings.TUIState
	}{
		{
			name:  "empty model",
			model: &Model{},
			want: settings.TUIState{
				DefaultExpandLevelSet: true,
			},
		},
		{
			name: "model with settings",
			model: &Model{
				sortBy:    settings.SortByLevel,
				sortOrder: settings.SortOrderAsc,
				columns:   []string{settings.ColumnID, settings.ColumnMessage, settings.ColumnLevel},
				filters: settings.Filter{
					Level:   settings.LevelFilterWarning,
					State:   settings.StateFilterActive,
					Session: "my-session",
					Window:  "@1",
					Pane:    "%1",
				},
				viewMode:           settings.ViewModeDetailed,
				groupBy:            settings.GroupBySession,
				defaultExpandLevel: 2,
				expansionState: map[string]bool{
					"session:$1": true,
				},
			},
			want: settings.TUIState{
				SortBy:    settings.SortByLevel,
				SortOrder: settings.SortOrderAsc,
				Columns:   []string{settings.ColumnID, settings.ColumnMessage, settings.ColumnLevel},
				Filters: settings.Filter{
					Level:   settings.LevelFilterWarning,
					State:   settings.StateFilterActive,
					Session: "my-session",
					Window:  "@1",
					Pane:    "%1",
				},
				ViewMode:              settings.ViewModeDetailed,
				GroupBy:               settings.GroupBySession,
				DefaultExpandLevel:    2,
				DefaultExpandLevelSet: true,
				ExpansionState: map[string]bool{
					"session:$1": true,
				},
			},
		},
		{
			name: "model with partial settings",
			model: &Model{
				sortBy:   settings.SortByTimestamp,
				viewMode: settings.ViewModeCompact,
				groupBy:  settings.GroupByNone,
			},
			want: settings.TUIState{
				SortBy:                settings.SortByTimestamp,
				ViewMode:              settings.ViewModeCompact,
				GroupBy:               settings.GroupByNone,
				DefaultExpandLevelSet: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.model.ToState()

			assert.Equal(t, tt.want.SortBy, got.SortBy)
			assert.Equal(t, tt.want.SortOrder, got.SortOrder)
			assert.Equal(t, tt.want.Columns, got.Columns)
			assert.Equal(t, tt.want.Filters, got.Filters)
			assert.Equal(t, tt.want.ViewMode, got.ViewMode)
			assert.Equal(t, tt.want.GroupBy, got.GroupBy)
			assert.Equal(t, tt.want.DefaultExpandLevel, got.DefaultExpandLevel)
			assert.Equal(t, tt.want.DefaultExpandLevelSet, got.DefaultExpandLevelSet)
			assert.Equal(t, tt.want.ExpansionState, got.ExpansionState)
		})
	}
}

func TestFromState(t *testing.T) {
	tests := []struct {
		name     string
		model    *Model
		state    settings.TUIState
		wantErr  bool
		verifyFn func(*testing.T, *Model)
	}{
		{
			name:    "empty state - no changes",
			model:   &Model{},
			state:   settings.TUIState{},
			wantErr: false,
			verifyFn: func(t *testing.T, m *Model) {
				assert.Equal(t, "", m.sortBy)
				assert.Equal(t, "", m.sortOrder)
				assert.Empty(t, m.columns)
				assert.Equal(t, "", m.viewMode)
				assert.Equal(t, "", m.groupBy)
				assert.Equal(t, 0, m.defaultExpandLevel)
				assert.Nil(t, m.expansionState)
				assert.Equal(t, settings.Filter{}, m.filters)
			},
		},
		{
			name:  "full state - all fields set",
			model: &Model{},
			state: settings.TUIState{
				SortBy:    settings.SortByLevel,
				SortOrder: settings.SortOrderAsc,
				Columns:   []string{settings.ColumnID, settings.ColumnMessage, settings.ColumnLevel},
				Filters: settings.Filter{
					Level:   settings.LevelFilterWarning,
					State:   settings.StateFilterActive,
					Session: "my-session",
					Window:  "@1",
					Pane:    "%1",
				},
				ViewMode:              settings.ViewModeDetailed,
				GroupBy:               settings.GroupByWindow,
				DefaultExpandLevel:    2,
				DefaultExpandLevelSet: true,
				ExpansionState: map[string]bool{
					"window:@1": true,
				},
			},
			wantErr: false,
			verifyFn: func(t *testing.T, m *Model) {
				assert.Equal(t, settings.SortByLevel, m.sortBy)
				assert.Equal(t, settings.SortOrderAsc, m.sortOrder)
				assert.Equal(t, []string{settings.ColumnID, settings.ColumnMessage, settings.ColumnLevel}, m.columns)
				assert.Equal(t, settings.ViewModeDetailed, m.viewMode)
				assert.Equal(t, settings.GroupByWindow, m.groupBy)
				assert.Equal(t, 2, m.defaultExpandLevel)
				assert.Equal(t, map[string]bool{"window:@1": true}, m.expansionState)
				assert.Equal(t, settings.LevelFilterWarning, m.filters.Level)
				assert.Equal(t, settings.StateFilterActive, m.filters.State)
				assert.Equal(t, "my-session", m.filters.Session)
				assert.Equal(t, "@1", m.filters.Window)
				assert.Equal(t, "%1", m.filters.Pane)
			},
		},
		{
			name: "partial state - only some fields set",
			model: &Model{
				sortBy:    settings.SortByTimestamp,
				sortOrder: settings.SortOrderDesc,
				columns:   []string{settings.ColumnID},
				filters: settings.Filter{
					Level: settings.LevelFilterError,
				},
				viewMode:           settings.ViewModeCompact,
				groupBy:            settings.GroupBySession,
				defaultExpandLevel: 3,
			},
			state: settings.TUIState{
				SortBy:                settings.SortByLevel,
				Columns:               []string{settings.ColumnID, settings.ColumnMessage},
				DefaultExpandLevel:    0,
				DefaultExpandLevelSet: true,
			},
			wantErr: false,
			verifyFn: func(t *testing.T, m *Model) {
				assert.Equal(t, settings.SortByLevel, m.sortBy)
				assert.Equal(t, settings.SortOrderDesc, m.sortOrder)
				assert.Equal(t, []string{settings.ColumnID, settings.ColumnMessage}, m.columns)
				assert.Equal(t, settings.LevelFilterError, m.filters.Level)
				assert.Equal(t, settings.ViewModeCompact, m.viewMode)
				assert.Equal(t, settings.GroupBySession, m.groupBy)
				assert.Equal(t, 0, m.defaultExpandLevel)
			},
		},
		{
			name: "partial filters - only some filter fields set",
			model: &Model{
				filters: settings.Filter{
					Level:   settings.LevelFilterError,
					State:   settings.StateFilterActive,
					Session: "old-session",
				},
				groupBy:            settings.GroupByPane,
				defaultExpandLevel: 2,
				expansionState: map[string]bool{
					"pane:%1": true,
				},
			},
			state: settings.TUIState{
				Filters: settings.Filter{
					Level:   settings.LevelFilterWarning,
					Session: "new-session",
				},
				ExpansionState: map[string]bool{},
			},
			wantErr: false,
			verifyFn: func(t *testing.T, m *Model) {
				assert.Equal(t, settings.LevelFilterWarning, m.filters.Level)
				assert.Equal(t, settings.StateFilterActive, m.filters.State)
				assert.Equal(t, "new-session", m.filters.Session)
				assert.Empty(t, m.filters.Window)
				assert.Empty(t, m.filters.Pane)
				assert.Equal(t, settings.GroupByPane, m.groupBy)
				assert.Equal(t, 2, m.defaultExpandLevel)
				assert.Equal(t, map[string]bool{}, m.expansionState)
			},
		},
		{
			name:    "invalid groupBy",
			model:   &Model{},
			state:   settings.TUIState{GroupBy: "invalid"},
			wantErr: true,
		},
		{
			name:    "invalid defaultExpandLevel",
			model:   &Model{},
			state:   settings.TUIState{DefaultExpandLevel: 4, DefaultExpandLevelSet: true},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.model.FromState(tt.state)

			if (err != nil) != tt.wantErr {
				t.Fatalf("FromState() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.verifyFn != nil {
				tt.verifyFn(t, tt.model)
			}
		})
	}
}

func TestRoundTripSettings(t *testing.T) {
	tests := []struct {
		name  string
		model *Model
	}{
		{
			name:  "empty model",
			model: &Model{},
		},
		{
			name: "model with all settings",
			model: &Model{
				sortBy:    settings.SortByLevel,
				sortOrder: settings.SortOrderAsc,
				columns:   []string{settings.ColumnID, settings.ColumnMessage, settings.ColumnLevel},
				filters: settings.Filter{
					Level:   settings.LevelFilterWarning,
					State:   settings.StateFilterActive,
					Session: "my-session",
					Window:  "@1",
					Pane:    "%1",
				},
				viewMode:           settings.ViewModeDetailed,
				groupBy:            settings.GroupByWindow,
				defaultExpandLevel: 2,
				expansionState: map[string]bool{
					"window:@1": true,
				},
			},
		},
		{
			name: "model with partial settings",
			model: &Model{
				sortBy:   settings.SortByTimestamp,
				viewMode: settings.ViewModeCompact,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.model.ToState()

			newModel := &Model{}
			err := newModel.FromState(state)
			require.NoError(t, err)

			assert.Equal(t, tt.model.sortBy, newModel.sortBy)
			assert.Equal(t, tt.model.sortOrder, newModel.sortOrder)
			assert.Equal(t, tt.model.columns, newModel.columns)
			assert.Equal(t, tt.model.filters, newModel.filters)
			assert.Equal(t, tt.model.viewMode, newModel.viewMode)
			assert.Equal(t, tt.model.groupBy, newModel.groupBy)
			assert.Equal(t, tt.model.defaultExpandLevel, newModel.defaultExpandLevel)
			assert.Equal(t, tt.model.expansionState, newModel.expansionState)
		})
	}
}

func TestSaveSettings(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{
		sortBy:             settings.SortByLevel,
		sortOrder:          settings.SortOrderAsc,
		columns:            []string{settings.ColumnID, settings.ColumnMessage},
		viewMode:           settings.ViewModeDetailed,
		groupBy:            settings.GroupBySession,
		defaultExpandLevel: 2,
		expansionState: map[string]bool{
			"session:$1": true,
		},
	}

	err := model.saveSettings()
	require.NoError(t, err)

	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, loaded.SortBy)
	assert.Equal(t, settings.SortOrderAsc, loaded.SortOrder)
	assert.Equal(t, []string{settings.ColumnID, settings.ColumnMessage}, loaded.Columns)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)
	assert.Equal(t, settings.GroupBySession, loaded.GroupBy)
	assert.Equal(t, 2, loaded.DefaultExpandLevel)
	assert.Equal(t, map[string]bool{"session:$1": true}, loaded.ExpansionState)
}

func TestModelSaveOnQuit(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{
		sortBy:    settings.SortByLevel,
		sortOrder: settings.SortOrderAsc,
		viewMode:  settings.ViewModeDetailed,
		groupBy:   settings.GroupBySession,
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(msg)

	assert.NotNil(t, cmd)

	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, loaded.SortBy)
	assert.Equal(t, settings.SortOrderAsc, loaded.SortOrder)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)
	assert.Equal(t, settings.GroupBySession, loaded.GroupBy)
}

func TestTUISaveOnExit(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{
		sortBy:    settings.SortByLevel,
		sortOrder: settings.SortOrderAsc,
		columns:   []string{settings.ColumnID, settings.ColumnMessage},
		viewMode:  settings.ViewModeDetailed,
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(msg)
	assert.NotNil(t, cmd)

	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, loaded.SortBy)
	assert.Equal(t, settings.SortOrderAsc, loaded.SortOrder)
	assert.Equal(t, []string{settings.ColumnID, settings.ColumnMessage}, loaded.Columns)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)
}

func TestModelSaveOnCtrlC(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{
		sortBy:   settings.SortByTimestamp,
		viewMode: settings.ViewModeDetailed,
		groupBy:  settings.GroupByWindow,
	}

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := model.Update(msg)

	assert.NotNil(t, cmd)

	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByTimestamp, loaded.SortBy)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)
	assert.Equal(t, settings.GroupByWindow, loaded.GroupBy)
}

func TestTUILoadOnStart(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	original := &settings.Settings{
		Columns:   []string{settings.ColumnID, settings.ColumnMessage, settings.ColumnLevel},
		SortBy:    settings.SortByLevel,
		SortOrder: settings.SortOrderAsc,
		Filters: settings.Filter{
			Level:   settings.LevelFilterWarning,
			State:   settings.StateFilterActive,
			Session: "session-1",
		},
		ViewMode: settings.ViewModeDetailed,
	}

	require.NoError(t, settings.Save(original))

	loaded, err := settings.Load()
	require.NoError(t, err)

	model := &Model{}
	model.SetLoadedSettings(loaded)
	require.NoError(t, model.FromState(settings.FromSettings(loaded)))

	assert.Equal(t, original.Columns, model.columns)
	assert.Equal(t, original.SortBy, model.sortBy)
	assert.Equal(t, original.SortOrder, model.sortOrder)
	assert.Equal(t, original.Filters, model.filters)
	assert.Equal(t, original.ViewMode, model.viewMode)
}

func TestModelSaveOnCommandQ(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{
		sortBy:   settings.SortByLevel,
		viewMode: settings.ViewModeDetailed,
		groupBy:  settings.GroupByPane,
	}

	model.commandMode = true
	model.commandQuery = "q"
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := model.Update(msg)

	assert.NotNil(t, cmd)

	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, loaded.SortBy)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)
	assert.Equal(t, settings.GroupByPane, loaded.GroupBy)
}

func TestModelSaveCommandW(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	model := &Model{
		sortBy:   settings.SortByLevel,
		viewMode: settings.ViewModeDetailed,
		groupBy:  settings.GroupBySession,
	}

	model.commandMode = true
	model.commandQuery = "w"
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := model.Update(msg)

	require.NotNil(t, cmd)
	_ = cmd()

	model = updated.(*Model)
	assert.False(t, model.commandMode)
	assert.Equal(t, "", model.commandQuery)

	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, loaded.SortBy)
	assert.Equal(t, settings.ViewModeDetailed, loaded.ViewMode)
	assert.Equal(t, settings.GroupBySession, loaded.GroupBy)
}

func TestModelMissingSettingsFile(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	setupStorage(t)
	mockClient := stubSessionFetchers(t)

	settingsPath := tmpDir + "/settings.json"
	_, err := os.Stat(settingsPath)
	assert.True(t, os.IsNotExist(err))

	model, err := NewModel(mockClient)
	require.NoError(t, err)
	assert.NotNil(t, model)
}

func TestModelCorruptedSettingsFile(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	setupStorage(t)
	stubSessionFetchers(t)

	settingsPath := tmpDir + "/settings.json"
	err := os.WriteFile(settingsPath, []byte("invalid json {{{"), 0644)
	require.NoError(t, err)

	loaded, err := settings.Load()
	require.NoError(t, err)
	assert.NotNil(t, loaded)
	assert.NotEmpty(t, loaded.SortBy)
}

func TestModelSettingsLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	setupConfig(t, tmpDir)

	setupStorage(t)
	stubSessionFetchers(t)

	loaded, err := settings.Load()
	require.NoError(t, err)
	require.NotNil(t, loaded)

	model := &Model{}
	state := settings.FromSettings(loaded)
	err = model.FromState(state)
	require.NoError(t, err)

	model.sortBy = settings.SortByLevel
	model.sortOrder = settings.SortOrderAsc
	model.viewMode = settings.ViewModeDetailed
	model.groupBy = settings.GroupByWindow
	model.defaultExpandLevel = 2
	model.expansionState = map[string]bool{"window:@1": true}

	err = model.saveSettings()
	require.NoError(t, err)

	reloaded, err := settings.Load()
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, reloaded.SortBy)
	assert.Equal(t, settings.SortOrderAsc, reloaded.SortOrder)
	assert.Equal(t, settings.ViewModeDetailed, reloaded.ViewMode)
	assert.Equal(t, settings.GroupByWindow, reloaded.GroupBy)
	assert.Equal(t, 2, reloaded.DefaultExpandLevel)
	assert.Equal(t, map[string]bool{"window:@1": true}, reloaded.ExpansionState)

	newModel := &Model{}
	newState := settings.FromSettings(reloaded)
	err = newModel.FromState(newState)
	require.NoError(t, err)
	assert.Equal(t, settings.SortByLevel, newModel.sortBy)
	assert.Equal(t, settings.SortOrderAsc, newModel.sortOrder)
	assert.Equal(t, settings.ViewModeDetailed, newModel.viewMode)
	assert.Equal(t, settings.GroupByWindow, newModel.groupBy)
	assert.Equal(t, 2, newModel.defaultExpandLevel)
	assert.Equal(t, map[string]bool{"window:@1": true}, newModel.expansionState)
}
