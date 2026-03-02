package main

import (
	"fmt"
	"sync"

	"github.com/cristianoliveira/tmux-intray/cmd"
	"github.com/cristianoliveira/tmux-intray/internal/core"
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

var newStorageFromConfig = storage.NewFromConfig

func buildCLIDeps() (cliDeps, error) {
	stor, err := newStorageFromConfig()
	if err != nil {
		return cliDeps{}, fmt.Errorf("deps: failed to initialize storage: %w", err)
	}
	coreClient := core.NewCoreWithDeps(nil, stor, nil)
	return cliDeps{
		coreClient: coreClient,
		storage:    stor,
		tuiClient:  app.NewDefaultClient(nil),
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
	deps, err := buildCLIDeps()
	if err != nil {
		return err
	}
	registerCommands(cmd.RootCmd, deps)
	return nil
}
