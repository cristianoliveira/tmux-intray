package main

import (
	"fmt"
	"sync"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/config"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/logging"
	"github.com/cristianoliveira/tmux-intray/internal/ports"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tui/app"
	"github.com/spf13/cobra"
)

type cliCore interface {
	EnsureTmuxRunning() bool
	AddTrayItem(item, session, window, pane, paneCreated string, noAssociate bool, level string) (string, error)
	ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error)
	GetActiveCount() int
	DismissNotification(id string) error
	DismissAll() error
	MarkNotificationRead(id string) error
	MarkNotificationUnread(id string) error
	CleanupOldNotifications(daysThreshold int, dryRun bool) error
	JumpToPane(sessionID, windowID, paneID string) bool
	ValidatePaneExists(sessionID, windowID, paneID string) bool
	GetNotificationByID(id string) (string, error)
	GetCurrentTmuxContext() core.TmuxContext
	GetTmuxVisibility() string
	SetTmuxVisibility(value string) (bool, error)
	ClearTrayItems() error
	LoadSettings() (*settings.Settings, error)
	ResetSettings() (*settings.Settings, error)
}

type cliDeps struct {
	coreClient cliCore
	storage    ports.NotificationRepository
	tuiClient  tuiClient
}

type storageFactory func() (ports.NotificationRepository, error)

type coreFactory func(storage ports.NotificationRepository) (cliCore, error)

type tuiFactory func() (tuiClient, error)

type cliDepsFactories struct {
	newStorage storageFactory
	newCore    coreFactory
	newTUI     tuiFactory
}

func defaultCLIDepsFactories() cliDepsFactories {
	return cliDepsFactories{
		newStorage: func() (ports.NotificationRepository, error) {
			return storage.NewFromConfig()
		},
		newCore: func(stor ports.NotificationRepository) (cliCore, error) {
			return core.NewCoreWithDeps(nil, stor, nil), nil
		},
		newTUI: func() (tuiClient, error) {
			return app.NewDefaultClient(nil, nil, nil), nil
		},
	}
}

func buildCLIDeps() (cliDeps, error) {
	return buildCLIDepsWithFactories(defaultCLIDepsFactories())
}

func buildCLIDepsWithFactories(factories cliDepsFactories) (cliDeps, error) {
	stor, err := factories.newStorage()
	if err != nil {
		return cliDeps{}, fmt.Errorf("deps: failed to initialize storage: %w", err)
	}

	coreClient, err := factories.newCore(stor)
	if err != nil {
		return cliDeps{}, fmt.Errorf("deps: failed to initialize core: %w", err)
	}

	tuiClient, err := factories.newTUI()
	if err != nil {
		return cliDeps{}, fmt.Errorf("deps: failed to initialize tui: %w", err)
	}

	return cliDeps{
		coreClient: coreClient,
		storage:    stor,
		tuiClient:  tuiClient,
	}, nil
}

var registerCommandsOnce sync.Once

func registerCommands(root *cobra.Command, deps cliDeps) {
	registerCommandsOnce.Do(func() {
		root.AddCommand(NewAddCmd(deps.coreClient))
		root.AddCommand(NewListCmd(deps.coreClient))
		root.AddCommand(NewStatusCmd(deps.coreClient))
		root.AddCommand(NewFollowCmd(deps.coreClient))
		root.AddCommand(NewClearCmd(deps.coreClient))
		root.AddCommand(NewDismissCmd(deps.coreClient))
		root.AddCommand(NewMarkReadCmd(deps.coreClient))
		root.AddCommand(NewCleanupCmd(deps.coreClient))
		root.AddCommand(NewJumpCmd(deps.coreClient))
		root.AddCommand(NewSettingsCmd(deps.coreClient))
		root.AddCommand(NewTUICmd(deps.tuiClient))

		dismissFunc = deps.coreClient.DismissNotification
		dismissAllFunc = deps.coreClient.DismissAll
		clearAllFunc = func() error {
			return deps.coreClient.DismissAll()
		}
	})
}

func initCLI() error {
	// Load configuration first
	config.Load()

	// Check for --log-file flag
	var logFile string
	if cmd.RootCmd.PersistentFlags().Changed("log-file") {
		logFile, _ = cmd.RootCmd.PersistentFlags().GetString("log-file")
	} else {
		logFile = config.Get("log_file", "")
	}

	// Initialize structured logging if enabled
	loggingEnabled := config.GetBool("logging_enabled", false)
	logLevel := config.Get("logging_level", "info")
	maxFiles := config.GetInt("logging_max_files", 10)
	stateDir := config.Get("state_dir", "")

	cfg := &logging.LoggingConfig{
		Enabled:  loggingEnabled,
		Level:    logLevel,
		MaxFiles: maxFiles,
		LogFile:  logFile,
		StateDir: stateDir,
	}

	if err := logging.Init(cfg); err != nil {
		// Log initialization error should not prevent startup
		colors.Warning("failed to initialize logging: " + err.Error())
	}

	// Log startup information if logging is enabled
	if loggingEnabled {
		logging.LogStartup()
	}

	// Build and register CLI commands
	deps, err := buildCLIDeps()
	if err != nil {
		return err
	}
	registerCommands(cmd.RootCmd, deps)
	return nil
}
