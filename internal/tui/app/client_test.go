// Package app provides TUI application adapters for command wiring.
package app

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
)

// mockSettingsLoader is a test double for SettingsLoader.
type mockSettingsLoader struct {
	settings *settings.Settings
	err      error
}

func (m *mockSettingsLoader) Load() (*settings.Settings, error) {
	return m.settings, m.err
}

// TestDefaultClient_LoadSettings_WithInjectedLoader verifies that DefaultClient
// uses the injected SettingsLoader instead of calling settings.Load directly.
func TestDefaultClient_LoadSettings_WithInjectedLoader(t *testing.T) {
	expectedSettings := &settings.Settings{
		SortBy:    "timestamp",
		SortOrder: "desc",
	}
	mockLoader := &mockSettingsLoader{
		settings: expectedSettings,
		err:      nil,
	}

	client := NewDefaultClient(nil, nil, mockLoader)

	result, err := client.LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings() error = %v, want nil", err)
	}

	if result.SortBy != expectedSettings.SortBy {
		t.Errorf("LoadSettings() SortBy = %v, want %v", result.SortBy, expectedSettings.SortBy)
	}
}

// TestDefaultClient_LoadSettings_WithLoaderError verifies that DefaultClient
// propagates errors from the injected SettingsLoader.
func TestDefaultClient_LoadSettings_WithLoaderError(t *testing.T) {
	expectedErr := errors.New("failed to load settings")
	mockLoader := &mockSettingsLoader{
		settings: nil,
		err:      expectedErr,
	}

	client := NewDefaultClient(nil, nil, mockLoader)

	_, err := client.LoadSettings()
	if err != expectedErr {
		t.Fatalf("LoadSettings() error = %v, want %v", err, expectedErr)
	}
}

// TestNewDefaultClient_BackwardCompatibility verifies that passing nil for
// settingsLoader results in a DefaultSettingsLoader being used, maintaining
// backward compatibility.
func TestNewDefaultClient_BackwardCompatibility(t *testing.T) {
	client := NewDefaultClient(nil, nil, nil)

	if client.settingsLoader == nil {
		t.Fatal("NewDefaultClient() settingsLoader is nil, want DefaultSettingsLoader")
	}

	// Verify it's a DefaultSettingsLoader by calling Load
	_, err := client.settingsLoader.Load()
	if err != nil {
		// This is expected if settings file doesn't exist, but it shouldn't panic
		// The important thing is that it doesn't panic and returns an error
	}
}

// mockTmuxClientFactory is a test double for TmuxClientFactory.
type mockTmuxClientFactory struct {
	client tmux.TmuxClient
}

func (m *mockTmuxClientFactory) NewClient() tmux.TmuxClient {
	return m.client
}

// mockProgramRunner is a test double for ProgramRunner.
type mockProgramRunner struct {
	err error
}

func (m *mockProgramRunner) Run(model tea.Model) error {
	return m.err
}

// mockModel is a simple implementation of Model interface for testing.
type mockModel struct{}

func (m *mockModel) Init() tea.Cmd {
	return nil
}

func (m *mockModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *mockModel) View() string {
	return ""
}

func (m *mockModel) SetLoadedSettings(loadedSettings *settings.Settings) {}

func (m *mockModel) FromState(settingsState settings.TUIState) error {
	return nil
}

// TestDefaultClient_CreateModel_WithInjectedFactory verifies that DefaultClient
// uses the injected TmuxClientFactory instead of creating one directly.
func TestDefaultClient_CreateModel_WithInjectedFactory(t *testing.T) {
	mockTmuxClient := new(tmux.MockClient)
	// Configure the mock to return values expected by NewModel
	mockTmuxClient.On("ListSessions").Return(map[string]string{"$1": "test-session"}, nil)
	mockTmuxClient.On("ListWindows").Return(map[string]string{"@0": "main"}, nil)
	mockTmuxClient.On("ListPanes").Return(map[string]string{"%0": "terminal"}, nil)

	mockFactory := &mockTmuxClientFactory{
		client: mockTmuxClient,
	}

	client := NewDefaultClient(mockFactory, nil, nil)

	model, err := client.CreateModel()
	if err != nil {
		t.Fatalf("CreateModel() error = %v, want nil", err)
	}

	if model == nil {
		t.Fatal("CreateModel() returned nil model")
	}
}

// TestDefaultClient_CreateModel_Success verifies that CreateModel returns
// a valid model when the factory returns a valid client.
func TestDefaultClient_CreateModel_Success(t *testing.T) {
	mockTmuxClient := new(tmux.MockClient)
	// Configure the mock to return values expected by NewModel
	mockTmuxClient.On("ListSessions").Return(map[string]string{"$1": "test-session"}, nil)
	mockTmuxClient.On("ListWindows").Return(map[string]string{"@0": "main"}, nil)
	mockTmuxClient.On("ListPanes").Return(map[string]string{"%0": "terminal"}, nil)

	mockFactory := &mockTmuxClientFactory{
		client: mockTmuxClient,
	}

	client := NewDefaultClient(mockFactory, nil, nil)

	model, err := client.CreateModel()
	assert.NoError(t, err)
	assert.NotNil(t, model)

	// Verify the model implements the Model interface
	_, ok := model.(Model)
	assert.True(t, ok, "CreateModel() should return a Model implementation")
}

// TestDefaultClient_CreateModel_NilFactory verifies that passing nil for
// TmuxClientFactory results in a DefaultTmuxClientFactory being used.
func TestDefaultClient_CreateModel_NilFactory(t *testing.T) {
	client := NewDefaultClient(nil, nil, nil)

	if client.tmuxClientFactory == nil {
		t.Fatal("NewDefaultClient() tmuxClientFactory is nil, want DefaultTmuxClientFactory")
	}

	// Verify it's a DefaultTmuxClientFactory by calling NewClient
	tmuxClient := client.tmuxClientFactory.NewClient()
	if tmuxClient == nil {
		t.Fatal("DefaultTmuxClientFactory.NewClient() returned nil")
	}
}

// TestDefaultClient_RunProgram_Success verifies that RunProgram calls the
// injected ProgramRunner and returns nil when successful.
func TestDefaultClient_RunProgram_Success(t *testing.T) {
	mockRunner := &mockProgramRunner{
		err: nil,
	}
	testModel := &mockModel{}

	client := NewDefaultClient(nil, mockRunner, nil)

	err := client.RunProgram(testModel)
	if err != nil {
		t.Fatalf("RunProgram() error = %v, want nil", err)
	}
}

// TestDefaultClient_RunProgram_WithRunnerError verifies that RunProgram
// propagates errors from the injected ProgramRunner.
func TestDefaultClient_RunProgram_WithRunnerError(t *testing.T) {
	expectedErr := errors.New("program failed")
	mockRunner := &mockProgramRunner{
		err: expectedErr,
	}
	testModel := &mockModel{}

	client := NewDefaultClient(nil, mockRunner, nil)

	err := client.RunProgram(testModel)
	if err != expectedErr {
		t.Fatalf("RunProgram() error = %v, want %v", err, expectedErr)
	}
}

// TestDefaultClient_RunProgram_NilRunner verifies that passing nil for
// ProgramRunner results in a DefaultProgramRunner being used.
func TestDefaultClient_RunProgram_NilRunner(t *testing.T) {
	client := NewDefaultClient(nil, nil, nil)

	if client.programRunner == nil {
		t.Fatal("NewDefaultClient() programRunner is nil, want DefaultProgramRunner")
	}

	// Verify it's a DefaultProgramRunner by checking its type
	_, ok := client.programRunner.(*DefaultProgramRunner)
	if !ok {
		t.Fatalf("programRunner is %T, want *DefaultProgramRunner", client.programRunner)
	}
}

// TestDefaultClient_AllDepsInjected verifies that when all dependencies are injected,
// they are used correctly and not replaced with defaults.
func TestDefaultClient_AllDepsInjected(t *testing.T) {
	mockLoader := &mockSettingsLoader{}
	mockFactory := &mockTmuxClientFactory{}
	mockRunner := &mockProgramRunner{}

	client := NewDefaultClient(mockFactory, mockRunner, mockLoader)

	// Verify all injected dependencies are preserved
	assert.Same(t, mockLoader, client.settingsLoader, "SettingsLoader should be preserved")
	assert.Same(t, mockFactory, client.tmuxClientFactory, "TmuxClientFactory should be preserved")
	assert.Same(t, mockRunner, client.programRunner, "ProgramRunner should be preserved")

	// Verify they work correctly
	expectedSettings := &settings.Settings{SortBy: "name"}
	mockLoader.settings = expectedSettings

	settings, err := client.LoadSettings()
	assert.NoError(t, err)
	assert.Equal(t, expectedSettings, settings)

	mockTmuxClient := new(tmux.MockClient)
	// Configure the mock to return values expected by NewModel
	mockTmuxClient.On("ListSessions").Return(map[string]string{"$1": "test-session"}, nil)
	mockTmuxClient.On("ListWindows").Return(map[string]string{"@0": "main"}, nil)
	mockTmuxClient.On("ListPanes").Return(map[string]string{"%0": "terminal"}, nil)
	mockFactory.client = mockTmuxClient

	model, err := client.CreateModel()
	assert.NoError(t, err)
	assert.NotNil(t, model)

	expectedErr := errors.New("test error")
	mockRunner.err = expectedErr

	testModel := &mockModel{}
	err = client.RunProgram(testModel)
	assert.Equal(t, expectedErr, err)
}

// TestDefaultClient_MixedDeps verifies that when some dependencies are injected
// and others are nil, the nil ones get default implementations.
func TestDefaultClient_MixedDeps(t *testing.T) {
	mockLoader := &mockSettingsLoader{}
	client := NewDefaultClient(nil, nil, mockLoader)

	// SettingsLoader should be the injected one
	assert.Same(t, mockLoader, client.settingsLoader)

	// TmuxClientFactory should be DefaultTmuxClientFactory
	_, ok := client.tmuxClientFactory.(*DefaultTmuxClientFactory)
	assert.True(t, ok, "TmuxClientFactory should be DefaultTmuxClientFactory")

	// ProgramRunner should be DefaultProgramRunner
	_, ok = client.programRunner.(*DefaultProgramRunner)
	assert.True(t, ok, "ProgramRunner should be DefaultProgramRunner")
}

// TestDefaultClient_ConcurrentAccess verifies that multiple goroutines can
// safely call methods on DefaultClient concurrently.
func TestDefaultClient_ConcurrentAccess(t *testing.T) {
	mockLoader := &mockSettingsLoader{
		settings: &settings.Settings{SortBy: "name"},
	}
	mockTmuxClient := new(tmux.MockClient)
	// Configure the mock to handle concurrent calls
	mockTmuxClient.On("ListSessions").Return(map[string]string{"$1": "test-session"}, nil)
	mockTmuxClient.On("ListWindows").Return(map[string]string{"@0": "main"}, nil)
	mockTmuxClient.On("ListPanes").Return(map[string]string{"%0": "terminal"}, nil)
	mockFactory := &mockTmuxClientFactory{client: mockTmuxClient}
	mockRunner := &mockProgramRunner{err: nil}

	client := NewDefaultClient(mockFactory, mockRunner, mockLoader)

	// Run multiple operations concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			client.LoadSettings()
			client.CreateModel()
			client.RunProgram(&mockModel{})
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestDefaultClient_NoOsExit verifies that no methods in DefaultClient
// can trigger os.Exit calls when given nil inputs or error conditions.
func TestDefaultClient_NoOsExit(t *testing.T) {
	// Test with all nil dependencies
	client := NewDefaultClient(nil, nil, nil)

	// None of these should panic or call os.Exit
	_, err := client.LoadSettings()
	// May return error if settings file doesn't exist, but should not panic
	_ = err

	_, err = client.CreateModel()
	// May return error or succeed, but should not panic
	_ = err

	// Test with error-returning mocks
	mockLoader := &mockSettingsLoader{err: errors.New("load error")}
	clientWithErrors := NewDefaultClient(nil, nil, mockLoader)

	_, err = clientWithErrors.LoadSettings()
	assert.Error(t, err, "LoadSettings should return error")

	mockFactory := &mockTmuxClientFactory{
		client: nil, // This might cause CreateModel to fail
	}
	clientWithNilFactory := NewDefaultClient(mockFactory, nil, nil)

	// This should not panic even if factory returns nil
	_, err = clientWithNilFactory.CreateModel()
	// May succeed or fail, but should not panic
	_ = err
}

// TestDefaultClient_LoadSettings_MultipleCalls verifies that LoadSettings
// can be called multiple times with consistent results.
func TestDefaultClient_LoadSettings_MultipleCalls(t *testing.T) {
	expectedSettings := &settings.Settings{
		SortBy:    "timestamp",
		SortOrder: "desc",
	}
	mockLoader := &mockSettingsLoader{
		settings: expectedSettings,
		err:      nil,
	}

	client := NewDefaultClient(nil, nil, mockLoader)

	// Call LoadSettings multiple times
	for i := 0; i < 5; i++ {
		result, err := client.LoadSettings()
		if err != nil {
			t.Fatalf("LoadSettings() call %d error = %v, want nil", i+1, err)
		}
		if result.SortBy != expectedSettings.SortBy {
			t.Errorf("LoadSettings() call %d SortBy = %v, want %v", i+1, result.SortBy, expectedSettings.SortBy)
		}
	}
}

// TestDefaultClient_RunProgram_WithNilModel verifies that RunProgram
// handles nil model gracefully without panicking.
func TestDefaultClient_RunProgram_WithNilModel(t *testing.T) {
	mockRunner := &mockProgramRunner{
		err: nil,
	}

	client := NewDefaultClient(nil, mockRunner, nil)

	// This should pass through to the runner and not panic
	err := client.RunProgram(nil)
	// The runner implementation may error, but DefaultClient should not panic
	_ = err
}

// TestDefaultClient_CreateModel_ReturnsModelType verifies that the model
// returned by CreateModel implements all required Model interface methods.
func TestDefaultClient_CreateModel_ReturnsModelType(t *testing.T) {
	mockTmuxClient := new(tmux.MockClient)
	// Configure the mock to return values expected by NewModel
	mockTmuxClient.On("ListSessions").Return(map[string]string{"$1": "test-session"}, nil)
	mockTmuxClient.On("ListWindows").Return(map[string]string{"@0": "main"}, nil)
	mockTmuxClient.On("ListPanes").Return(map[string]string{"%0": "terminal"}, nil)
	mockFactory := &mockTmuxClientFactory{
		client: mockTmuxClient,
	}

	client := NewDefaultClient(mockFactory, nil, nil)

	model, err := client.CreateModel()
	require.NoError(t, err)
	require.NotNil(t, model)

	// Verify it's a tea.Model
	_, ok := model.(tea.Model)
	assert.True(t, ok, "Model should implement tea.Model")

	// Verify it's our app.Model interface
	_, ok = model.(Model)
	assert.True(t, ok, "Model should implement app.Model")

	// Test SetLoadedSettings
	loadedSettings := &settings.Settings{SortBy: "name"}
	assert.NotPanics(t, func() {
		model.SetLoadedSettings(loadedSettings)
	}, "SetLoadedSettings should not panic")

	// Test FromState
	uiState := settings.TUIState{
		SortBy: "name",
	}
	err = model.FromState(uiState)
	assert.NoError(t, err, "FromState should not error")
}
