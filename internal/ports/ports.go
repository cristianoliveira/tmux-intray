// Package ports defines application boundary interfaces used by core services.
package ports

// NotificationRepository defines the storage operations used by core services.
type NotificationRepository interface {
	AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error)
	ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error)
	GetNotificationByID(id string) (string, error)
	DismissNotification(id string) error
	DismissAll() error
	MarkNotificationRead(id string) error
	MarkNotificationUnread(id string) error
	CleanupOldNotifications(daysThreshold int, dryRun bool) error
	GetActiveCount() int
}

// NotificationLookup defines read-only notification lookup operations.
type NotificationLookup interface {
	GetNotificationByID(id string) (string, error)
}

// TmuxContext captures current tmux identifiers.
type TmuxContext struct {
	SessionID string
	WindowID  string
	PaneID    string
	PanePID   string
}

// TmuxClient defines the tmux operations needed by core services.
type TmuxClient interface {
	GetCurrentContext() (TmuxContext, error)
	ValidatePaneExists(sessionID, windowID, paneID string) (bool, error)
	JumpToPane(sessionID, windowID, paneID string) (bool, error)
	SetEnvironment(name, value string) error
	GetEnvironment(name string) (string, error)
	HasSession() (bool, error)
	Run(args ...string) (string, string, error)
}

// SettingsStore defines settings operations consumed by core services.
type SettingsStore interface {
	LoadSettings() (any, error)
	ResetSettings() (any, error)
}

// ConfigProvider defines config reads used by status formatting clients.
type ConfigProvider interface {
	GetConfigBool(key string, defaultValue bool) bool
	GetConfigString(key, defaultValue string) string
}

// StatusPublisher defines minimal tmux status publishing capability.
type StatusPublisher interface {
	HasSession() (bool, error)
	SetStatusOption(name, value string) error
}
