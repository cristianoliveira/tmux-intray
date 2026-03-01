package state

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/errors"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/search"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/cristianoliveira/tmux-intray/internal/tui/controller"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
	"github.com/cristianoliveira/tmux-intray/internal/tui/service"
)

const (
	viewModeCompact       = settings.ViewModeCompact
	viewModeDetailed      = settings.ViewModeDetailed
	viewModeGrouped       = settings.ViewModeGrouped
	headerFooterLines     = 3
	defaultViewportWidth  = 80
	defaultViewportHeight = 22
	errorClearDuration    = 5 * time.Second
)

// Model represents the TUI model for bubbletea.
type Model struct {
	// Core state
	uiState           *UIState           // Extracted UI state management
	errorHandler      *errors.TUIHandler // Error handler for TUI messages
	statusMessage     string             // Current status message to display
	statusMessageType errors.MessageType // Message type for styling/prefix
	hasStatusMessage  bool               // Whether a status message is set

	// Legacy mirrors retained for backward-compatible tests.
	notifications []notification.Notification
	filtered      []notification.Notification

	// Settings fields (non-UI state)
	sortBy         string
	sortOrder      string
	unreadFirst    bool
	columns        []string
	filters        settings.Filter
	loadedSettings *settings.Settings // Track loaded settings for comparison
	settingsSvc    *settingsService
	// UI render options
	groupHeaderOptions settings.GroupHeaderOptions

	// Services - implementing BubbleTea nested model pattern
	treeService         model.TreeService
	notificationService model.NotificationService
	runtimeCoordinator  model.RuntimeCoordinator
	interactionCtrl     model.InteractionController
	// Legacy fields for backward compatibility
	client            tmux.TmuxClient
	sessionNames      map[string]string
	windowNames       map[string]string
	paneNames         map[string]string
	ensureTmuxRunning func() bool
	jumpToPane        func(sessionID, windowID, paneID string) bool
	searchProvider    search.Provider
}

// Init initializes the TUI model.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model state.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case saveSettingsSuccessMsg:
		return m.handleSaveSettingsSuccess(msg)
	case saveSettingsFailedMsg:
		return m.handleSaveSettingsFailed(msg)
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)
	case errorMsg:
		m.statusMessage = ""
		m.statusMessageType = errors.MessageTypeError
		m.hasStatusMessage = false
		return m, nil
	}
	return m, nil
}

// NewModel creates a new TUI model.
// If client is nil, a new DefaultClient is created.
func NewModel(client tmux.TmuxClient) (*Model, error) {
	if client == nil {
		client = tmux.NewDefaultClient()
	}

	// Initialize UI state
	uiState := NewUIState()

	// Initialize runtime coordinator (handles tmux integration and name resolution)
	runtimeCoordinator := service.NewRuntimeCoordinator(client)

	// Initialize tree service
	treeService := service.NewTreeService(uiState.GetGroupBy())

	// Initialize notification service with default search provider
	searchProvider := search.NewTokenProvider(
		search.WithCaseInsensitive(true),
		search.WithSessionNames(runtimeCoordinator.GetSessionNames()),
		search.WithWindowNames(runtimeCoordinator.GetWindowNames()),
		search.WithPaneNames(runtimeCoordinator.GetPaneNames()),
	)
	notificationService := service.NewNotificationService(searchProvider, runtimeCoordinator)
	interactionCtrl := controller.NewInteractionController(runtimeCoordinator)

	m := Model{
		uiState:             uiState,
		statusMessage:       "",
		statusMessageType:   errors.MessageTypeError,
		hasStatusMessage:    false,
		runtimeCoordinator:  runtimeCoordinator,
		interactionCtrl:     interactionCtrl,
		treeService:         treeService,
		notificationService: notificationService,
		settingsSvc:         newSettingsService(),
		unreadFirst:         true, // Default to true for backward compatibility
		// Legacy fields kept for backward compatibility but now using services
		client:             client,
		sessionNames:       runtimeCoordinator.GetSessionNames(),
		windowNames:        runtimeCoordinator.GetWindowNames(),
		paneNames:          runtimeCoordinator.GetPaneNames(),
		ensureTmuxRunning:  core.EnsureTmuxRunning,
		jumpToPane:         core.JumpToPane,
		groupHeaderOptions: settings.DefaultGroupHeaderOptions(),
	}

	// Initialize error handler with callback that sets error message
	m.errorHandler = errors.NewTUIHandler(func(msg errors.Message) {
		m.statusMessage = msg.Text
		m.statusMessageType = msg.Type
		m.hasStatusMessage = msg.Text != ""
		// Note: The tick command to clear the error is handled by the caller
	})

	// Set error handler on runtime coordinator if it's a DefaultRuntimeCoordinator
	// This allows jump errors to be handled by the TUI error handler instead of the CLI handler
	if coordinator, ok := runtimeCoordinator.(*service.DefaultRuntimeCoordinator); ok {
		coordinator.SetErrorHandler(m.errorHandler)
	}

	// Load initial notifications
	err := m.loadNotifications(false)
	if err != nil {
		return &Model{}, err
	}

	return &m, nil
}
