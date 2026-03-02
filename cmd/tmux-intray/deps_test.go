package main

import (
	"errors"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tui/app"
	"github.com/spf13/cobra"
)

type fakeStorage struct{}

func (f *fakeStorage) AddNotification(message, timestamp, session, window, pane, paneCreated, level string) (string, error) {
	return "", nil
}

func (f *fakeStorage) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error) {
	return "", nil
}

func (f *fakeStorage) GetNotificationByID(id string) (string, error) {
	return "", nil
}

func (f *fakeStorage) DismissNotification(id string) error {
	return nil
}

func (f *fakeStorage) DismissAll() error {
	return nil
}

func (f *fakeStorage) DismissByFilter(session, window, pane string) error {
	return nil
}

func (f *fakeStorage) MarkNotificationRead(id string) error {
	return nil
}

func (f *fakeStorage) MarkNotificationUnread(id string) error {
	return nil
}

func (f *fakeStorage) MarkNotificationReadWithTimestamp(id, timestamp string) error {
	return nil
}

func (f *fakeStorage) MarkNotificationUnreadWithTimestamp(id, timestamp string) error {
	return nil
}

func (f *fakeStorage) CleanupOldNotifications(daysThreshold int, dryRun bool) error {
	return nil
}

func (f *fakeStorage) GetActiveCount() int {
	return 0
}

type fakeCore struct{}

func (f *fakeCore) EnsureTmuxRunning() bool {
	return true
}

func (f *fakeCore) AddTrayItem(item, session, window, pane, paneCreated string, noAssociate bool, level string) (string, error) {
	return "", nil
}

func (f *fakeCore) ListNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter, olderThanCutoff, newerThanCutoff, readFilter string) (string, error) {
	return "", nil
}

func (f *fakeCore) GetActiveCount() int {
	return 0
}

func (f *fakeCore) DismissNotification(id string) error {
	return nil
}

func (f *fakeCore) DismissAll() error {
	return nil
}

func (f *fakeCore) MarkNotificationRead(id string) error {
	return nil
}

func (f *fakeCore) MarkNotificationUnread(id string) error {
	return nil
}

func (f *fakeCore) CleanupOldNotifications(daysThreshold int, dryRun bool) error {
	return nil
}

func (f *fakeCore) JumpToPane(sessionID, windowID, paneID string) bool {
	return true
}

func (f *fakeCore) ValidatePaneExists(sessionID, windowID, paneID string) bool {
	return true
}

func (f *fakeCore) GetNotificationByID(id string) (string, error) {
	return "", nil
}

func (f *fakeCore) GetCurrentTmuxContext() core.TmuxContext {
	return core.TmuxContext{}
}

func (f *fakeCore) GetTmuxVisibility() string {
	return "0"
}

func (f *fakeCore) SetTmuxVisibility(value string) (bool, error) {
	return true, nil
}

func (f *fakeCore) ClearTrayItems() error {
	return nil
}

func (f *fakeCore) LoadSettings() (*settings.Settings, error) {
	return settings.DefaultSettings(), nil
}

func (f *fakeCore) ResetSettings() (*settings.Settings, error) {
	return settings.DefaultSettings(), nil
}

type fakeTUIClient struct{}

func (f *fakeTUIClient) LoadSettings() (*settings.Settings, error) {
	return settings.DefaultSettings(), nil
}

func (f *fakeTUIClient) CreateModel() (app.Model, error) {
	return nil, errors.New("no model")
}

func (f *fakeTUIClient) RunProgram(model app.Model) error {
	return nil
}

func TestBuildCLIDepsReturnsStorageError(t *testing.T) {
	originalNewStorage := newStorageFromConfig
	defer func() { newStorageFromConfig = originalNewStorage }()

	newStorageFromConfig = func() (storage.Storage, error) {
		return nil, errors.New("storage unavailable")
	}

	_, err := buildCLIDeps()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() == "" {
		t.Fatal("expected error message, got empty string")
	}
}

func TestBuildCLIDepsSuccess(t *testing.T) {
	originalNewStorage := newStorageFromConfig
	defer func() { newStorageFromConfig = originalNewStorage }()

	stubStorage := &fakeStorage{}
	newStorageFromConfig = func() (storage.Storage, error) {
		return stubStorage, nil
	}

	deps, err := buildCLIDeps()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deps.coreClient == nil {
		t.Fatal("expected coreClient to be set")
	}
	if deps.storage != stubStorage {
		t.Fatal("expected storage to match stub")
	}
	if deps.tuiClient == nil {
		t.Fatal("expected tuiClient to be set")
	}
}

func TestRegisterCommandsAddsCommands(t *testing.T) {
	originalDismissFunc := dismissFunc
	originalDismissAllFunc := dismissAllFunc
	originalClearAllFunc := clearAllFunc
	defer func() {
		dismissFunc = originalDismissFunc
		dismissAllFunc = originalDismissAllFunc
		clearAllFunc = originalClearAllFunc
	}()

	root := &cobra.Command{Use: "root"}
	deps := cliDeps{
		coreClient: &fakeCore{},
		storage:    &fakeStorage{},
		tuiClient:  &fakeTUIClient{},
	}

	registerCommands(root, deps)

	commandNames := map[string]bool{}
	for _, cmd := range root.Commands() {
		commandNames[cmd.Name()] = true
	}

	expected := []string{"add", "list", "status", "follow", "clear", "dismiss", "mark-read", "cleanup", "jump", "settings", "tui"}
	for _, name := range expected {
		if !commandNames[name] {
			t.Fatalf("expected command %q to be registered", name)
		}
	}
}
