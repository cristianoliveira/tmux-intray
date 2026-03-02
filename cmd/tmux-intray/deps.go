package main

import (
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tui/app"
)

// Core dependencies
var fileStorage, _ = storage.NewFromConfig()
var coreClient = core.NewCore(nil, fileStorage)

// TUI dependencies - concrete implementations for production use
var tuiSettingsLoader = app.NewDefaultSettingsLoader()
var tuiProgramRunner = app.NewDefaultProgramRunner()
var tuiTmuxClientFactory = app.NewDefaultTmuxClientFactory()

// defaultTUIClient is the fully wired TUI client with all dependencies injected
var defaultTUIClient = app.NewDefaultClient(tuiTmuxClientFactory, tuiProgramRunner, tuiSettingsLoader)
