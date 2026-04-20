package main

import (
	"errors"
	"strings"
	"testing"

	appcore "github.com/cristianoliveira/tmux-intray/internal/app"
	"github.com/cristianoliveira/tmux-intray/internal/core"
	"github.com/cristianoliveira/tmux-intray/internal/ports"
	"github.com/cristianoliveira/tmux-intray/internal/search"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
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

func testFactories() cliDepsFactories {
	return cliDepsFactories{
		newStorage: func() (ports.NotificationRepository, error) {
			return &fakeStorage{}, nil
		},
		newCore: func(stor ports.NotificationRepository) (cliCore, error) {
			return &fakeCore{}, nil
		},
		newTUI: func() (tuiClient, error) {
			return &fakeTUIClient{}, nil
		},
	}
}

func TestBuildCLIDepsWithFactoriesReturnsStorageError(t *testing.T) {
	factories := testFactories()
	factories.newStorage = func() (ports.NotificationRepository, error) {
		return nil, errors.New("storage unavailable")
	}

	_, err := buildCLIDepsWithFactories(factories)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to initialize storage") {
		t.Fatalf("expected storage initialization error, got %q", err.Error())
	}
}

func TestBuildCLIDepsWithFactoriesReturnsCoreError(t *testing.T) {
	factories := testFactories()
	factories.newCore = func(stor ports.NotificationRepository) (cliCore, error) {
		return nil, errors.New("core unavailable")
	}

	_, err := buildCLIDepsWithFactories(factories)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to initialize core") {
		t.Fatalf("expected core initialization error, got %q", err.Error())
	}
}

func TestBuildCLIDepsWithFactoriesReturnsTUIError(t *testing.T) {
	factories := testFactories()
	factories.newTUI = func() (tuiClient, error) {
		return nil, errors.New("tui unavailable")
	}

	_, err := buildCLIDepsWithFactories(factories)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to initialize tui") {
		t.Fatalf("expected tui initialization error, got %q", err.Error())
	}
}

func TestBuildCLIDepsWithFactoriesSuccess(t *testing.T) {
	stubStorage := &fakeStorage{}
	stubCore := &fakeCore{}
	stubTUI := &fakeTUIClient{}
	stubSearchProviderFactory := func(regex bool) search.Provider {
		return search.NewSubstringProvider()
	}
	stubStatusPresetLookup := func(name string) (string, bool) {
		return "{{unread-count}}", true
	}

	factories := testFactories()
	factories.newStorage = func() (ports.NotificationRepository, error) {
		return stubStorage, nil
	}
	factories.newCore = func(stor ports.NotificationRepository) (cliCore, error) {
		if stor != stubStorage {
			t.Fatalf("expected storage to be passed to core constructor")
		}
		return stubCore, nil
	}
	factories.newTUI = func() (tuiClient, error) {
		return stubTUI, nil
	}
	factories.newListSearchProvider = func() listSearchProviderFactory {
		return stubSearchProviderFactory
	}
	factories.newStatusPresetLookup = func() appcore.StatusPresetLookup {
		return stubStatusPresetLookup
	}

	deps, err := buildCLIDepsWithFactories(factories)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deps.coreClient != stubCore {
		t.Fatal("expected coreClient to match stub")
	}
	if deps.storage != stubStorage {
		t.Fatal("expected storage to match stub")
	}
	if deps.tuiClient != stubTUI {
		t.Fatal("expected tuiClient to match stub")
	}
	if deps.listSearchProviderFactory == nil {
		t.Fatal("expected listSearchProviderFactory to be set")
	}
	if deps.statusPresetLookup == nil {
		t.Fatal("expected statusPresetLookup to be set")
	}
}

func TestDefaultCLIDepsFactoriesWireRuntimeConstructors(t *testing.T) {
	factories := defaultCLIDepsFactories()

	if factories.newStorage == nil {
		t.Fatal("expected storage factory to be set")
	}
	if factories.newCore == nil {
		t.Fatal("expected core factory to be set")
	}
	if factories.newTUI == nil {
		t.Fatal("expected tui factory to be set")
	}
	if factories.newListSearchProvider == nil {
		t.Fatal("expected list search provider factory to be set")
	}
	if factories.newStatusPresetLookup == nil {
		t.Fatal("expected status preset lookup to be set")
	}

	coreClient, err := factories.newCore(&fakeStorage{})
	if err != nil {
		t.Fatalf("unexpected error creating core client: %v", err)
	}
	if coreClient == nil {
		t.Fatal("expected core client to be created")
	}

	tuiClient, err := factories.newTUI()
	if err != nil {
		t.Fatalf("unexpected error creating tui client: %v", err)
	}
	if tuiClient == nil {
		t.Fatal("expected tui client to be created")
	}

	listSearchProviderFactory := factories.newListSearchProvider()
	if listSearchProviderFactory == nil {
		t.Fatal("expected list search provider factory to be created")
	}

	statusPresetLookup := factories.newStatusPresetLookup()
	if statusPresetLookup == nil {
		t.Fatal("expected status preset lookup to be created")
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
		coreClient:                &fakeCore{},
		storage:                   &fakeStorage{},
		tuiClient:                 &fakeTUIClient{},
		listSearchProviderFactory: func(regex bool) search.Provider { return search.NewSubstringProvider() },
		statusPresetLookup:        func(name string) (string, bool) { return "{{unread-count}}", true },
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
