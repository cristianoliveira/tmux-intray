package state

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/search"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/controller"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
	"github.com/cristianoliveira/tmux-intray/internal/tui/service"
)

// SetLoadedSettings stores the loaded settings reference for later comparison.
func (m *Model) SetLoadedSettings(loaded *settings.Settings) {
	m.ensureSettingsService().setLoadedSettings(loaded)
	m.loadedSettings = loaded
	if loaded != nil {
		m.unreadFirst = loaded.UnreadFirst
		m.groupHeaderOptions = loaded.GroupHeader.Clone()
		// Pass settings to notification service for configurable sorting
		if notifSvc, ok := m.notificationService.(*service.DefaultNotificationService); ok {
			notifSvc.SetSettings(loaded)
		}
	} else {
		m.unreadFirst = true
		m.groupHeaderOptions = settings.DefaultGroupHeaderOptions()
	}
}

// ToState converts the Model to a TUIState DTO for settings persistence.
// Only persists user-configurable settings (columns, sort, filters, view mode).
func (m *Model) ToState() settings.TUIState {
	return m.ensureSettingsService().toState(m.uiState, m.columns, m.sortBy, m.sortOrder, m.unreadFirst, m.filters)
}

// FromState applies settings from TUIState to the Model.
// Supports partial updates - only updates non-empty fields.
// Returns an error if the settings are invalid.
func (m *Model) FromState(state settings.TUIState) error {
	if err := m.ensureSettingsService().fromState(state, m.uiState, &m.columns, &m.sortBy, &m.sortOrder, &m.unreadFirst, &m.filters); err != nil {
		return err
	}

	// If the persisted view mode is search, start with the search input active.
	if m.uiState.GetViewMode() == model.ViewModeSearch {
		m.uiState.SetSearchMode(true)
	}

	m.applySearchFilter()
	m.resetCursor()
	return nil
}

func (m *Model) ensureTreeService() model.TreeService {
	if m.treeService != nil {
		return m.treeService
	}

	groupBy := model.GroupByNone
	if m.uiState != nil {
		groupBy = m.uiState.GetGroupBy()
	}

	m.treeService = service.NewTreeService(groupBy)
	return m.treeService
}

func (m *Model) ensureSettingsService() *settingsService {
	if m.settingsSvc == nil {
		m.settingsSvc = newSettingsService()
		if m.loadedSettings != nil {
			m.settingsSvc.setLoadedSettings(m.loadedSettings)
		}
	}

	return m.settingsSvc
}

func (m *Model) ensureNotificationService() model.NotificationService {
	if m.notificationService == nil {
		// Get the search provider from runtime coordinator
		searchProvider := search.NewTokenProvider(
			search.WithCaseInsensitive(true),
		)
		if m.runtimeCoordinator != nil {
			searchProvider = search.NewTokenProvider(
				search.WithCaseInsensitive(true),
				search.WithSessionNames(m.runtimeCoordinator.GetSessionNames()),
				search.WithWindowNames(m.runtimeCoordinator.GetWindowNames()),
				search.WithPaneNames(m.runtimeCoordinator.GetPaneNames()),
			)
		}
		m.notificationService = service.NewNotificationService(searchProvider, m.runtimeCoordinator)
	}
	return m.notificationService
}

func (m *Model) ensureInteractionController() model.InteractionController {
	if m.interactionCtrl == nil {
		m.interactionCtrl = controller.NewInteractionController(m.runtimeCoordinator)
	}

	if syncer, ok := m.interactionCtrl.(interface {
		SetRuntimeCoordinator(model.RuntimeCoordinator)
	}); ok {
		syncer.SetRuntimeCoordinator(m.runtimeCoordinator)
	}

	return m.interactionCtrl
}

// applySearchFilter filters notifications based on the search query.
// This function only updates the filtered notifications; cursor management
// should be handled separately by resetCursor() or restoreCursor().
func (m *Model) applySearchFilter() {
	treeService := m.ensureTreeService()
	notificationService := m.ensureNotificationService()
	treeService.InvalidateCache()
	if len(notificationService.GetNotifications()) == 0 && len(m.notifications) > 0 {
		notificationService.SetNotifications(m.notifications)
	}

	notificationService.ApplyFiltersAndSearch(
		m.uiState.GetSearchQuery(),
		m.filters.State,
		m.filters.Level,
		m.filters.Session,
		m.filters.Window,
		m.filters.Pane,
		m.filters.Read,
		m.sortBy,
		m.sortOrder,
	)
	if m.isGroupedView() {
		_ = m.treeService.RebuildTreeForFilter(
			m.filteredNotifications(),
			string(m.uiState.GetGroupBy()),
			m.uiState.GetExpansionState(),
		)
	} else {
		treeService.ClearTree()
	}
	m.syncNotificationMirrors()
	m.updateViewportContent()
}

func (m *Model) allNotifications() []notification.Notification {
	if m.notificationService == nil {
		return nil
	}
	return m.ensureNotificationService().GetNotifications()
}

func (m *Model) filteredNotifications() []notification.Notification {
	return m.ensureNotificationService().GetFilteredNotifications()
}

func (m *Model) syncNotificationMirrors() {
	m.notifications = m.allNotifications()
	m.filtered = m.filteredNotifications()
}

// ApplySearchFilter is the public version of applySearchFilter.
func (m *Model) ApplySearchFilter() {
	m.applySearchFilter()
}

// loadNotifications loads notifications from storage.
// If preserveCursor is true, attempts to maintain the current cursor position.
func (m *Model) loadNotifications(preserveCursor bool) error {
	var savedCursorPos int
	var savedNodeID string

	if preserveCursor {
		// Save current cursor state
		savedCursorPos = m.uiState.GetCursor()
		cursor := m.uiState.GetCursor()
		visibleNodes := m.treeService.GetVisibleNodes()
		if m.isGroupedView() && cursor < len(visibleNodes) {
			savedNodeID = m.getNodeIdentifier(visibleNodes[cursor])
		} else if !m.isGroupedView() && cursor < len(m.filtered) {
			savedNodeID = fmt.Sprintf("notif:%d", m.filtered[cursor].ID)
		}
	}

	notifications, err := m.ensureInteractionController().LoadActiveNotifications()
	if err != nil {
		return fmt.Errorf("failed to load notifications: %w", err)
	}
	if len(notifications) == 0 {
		m.ensureNotificationService().SetNotifications([]notification.Notification{})
		m.syncNotificationMirrors()
		m.treeService.ClearTree()
		if preserveCursor {
			m.adjustCursorBounds()
		} else {
			m.resetCursor()
		}
		m.updateViewportContent()
		return nil
	}

	m.ensureNotificationService().SetNotifications(notifications)
	m.applySearchFilter()

	if preserveCursor {
		if savedNodeID != "" {
			// Try to restore cursor to the same notification
			m.restoreCursor(savedNodeID)
		} else {
			// If we couldn't save the node ID, just adjust to bounds
			m.uiState.SetCursor(savedCursorPos)
			m.adjustCursorBounds()
		}
	} else {
		m.resetCursor()
	}

	return nil
}

// resetCursor resets the cursor to the first item.
func (m *Model) resetCursor() {
	m.uiState.ResetCursor()
}

// ResetCursor is the public version of resetCursor.
func (m *Model) ResetCursor() {
	m.resetCursor()
}
