// Package tmux provides a unified abstraction layer for tmux operations.
package tmux

import (
	"github.com/stretchr/testify/mock"
)

// MockClient is a mock implementation of TmuxClient for testing.
// It uses testify/mock to provide flexible behavior configuration and
// method call tracking for assertions.
//
// Example usage:
//
//	mockClient := new(MockClient)
//	mockClient.On("GetCurrentContext").Return(TmuxContext{
//	    SessionID: "$1",
//	    WindowID:  "@1",
//	    PaneID:    "%1",
//	    PanePID:   "1234",
//	}, nil)
//
//	ctx, err := mockClient.GetCurrentContext()
//	assert.NoError(t, err)
//	assert.Equal(t, "$1", ctx.SessionID)
//
//	// Assert that the method was called
//	mockClient.AssertCalled(t, "GetCurrentContext")
type MockClient struct {
	mock.Mock
}

// GetCurrentContext returns a mocked tmux context.
// Configure the return value using:
//
//	mock.On("GetCurrentContext").Return(TmuxContext{...}, nil)
func (m *MockClient) GetCurrentContext() (TmuxContext, error) {
	args := m.Called()
	return args.Get(0).(TmuxContext), args.Error(1)
}

// ValidatePaneExists returns a mocked boolean indicating if a pane exists.
// Configure the return value using:
//
//	mock.On("ValidatePaneExists", "$1", "@1", "%1").Return(true, nil)
func (m *MockClient) ValidatePaneExists(sessionID, windowID, paneID string) (bool, error) {
	args := m.Called(sessionID, windowID, paneID)
	return args.Bool(0), args.Error(1)
}

// JumpToPane returns a mocked boolean indicating if pane jump was successful.
// Configure the return value using:
//
//	mock.On("JumpToPane", "$1", "@1", "%1").Return(true, nil)
func (m *MockClient) JumpToPane(sessionID, windowID, paneID string) (bool, error) {
	args := m.Called(sessionID, windowID, paneID)
	return args.Bool(0), args.Error(1)
}

// SetEnvironment returns a mocked error when setting a tmux environment variable.
// Configure the return value using:
//
//	mock.On("SetEnvironment", "VAR_NAME", "value").Return(nil)
func (m *MockClient) SetEnvironment(name, value string) error {
	args := m.Called(name, value)
	return args.Error(0)
}

// GetEnvironment returns a mocked environment variable value.
// Configure the return value using:
//
//	mock.On("GetEnvironment", "VAR_NAME").Return("value", nil)
func (m *MockClient) GetEnvironment(name string) (string, error) {
	args := m.Called(name)
	return args.String(0), args.Error(1)
}

// HasSession returns a mocked boolean indicating if tmux server is running.
// Configure the return value using:
//
//	mock.On("HasSession").Return(true, nil)
func (m *MockClient) HasSession() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

// SetStatusOption returns a mocked error when setting a status option.
// Configure the return value using:
//
//	mock.On("SetStatusOption", "status-left", "value").Return(nil)
func (m *MockClient) SetStatusOption(name, value string) error {
	args := m.Called(name, value)
	return args.Error(0)
}

// ListSessions returns a mocked map of session IDs to names.
// Configure the return value using:
//
//	sessions := map[string]string{"$1": "my-session"}
//	mock.On("ListSessions").Return(sessions, nil)
func (m *MockClient) ListSessions() (map[string]string, error) {
	args := m.Called()
	return args.Get(0).(map[string]string), args.Error(1)
}

// GetSessionName returns a mocked session name for a given session ID.
// Configure the return value using:
//
//	mock.On("GetSessionName", "$1").Return("my-session", nil)
func (m *MockClient) GetSessionName(sessionID string) (string, error) {
	args := m.Called(sessionID)
	return args.String(0), args.Error(1)
}

// ListWindows returns a mocked map of window IDs to names.
// Configure the return value using:
//
//	windows := map[string]string{"@0": "main"}
//	mock.On("ListWindows").Return(windows, nil)
func (m *MockClient) ListWindows() (map[string]string, error) {
	args := m.Called()
	return args.Get(0).(map[string]string), args.Error(1)
}

// ListPanes returns a mocked map of pane IDs to names.
// Configure the return value using:
//
//	panes := map[string]string{"%0": "terminal"}
//	mock.On("ListPanes").Return(panes, nil)
func (m *MockClient) ListPanes() (map[string]string, error) {
	args := m.Called()
	return args.Get(0).(map[string]string), args.Error(1)
}

// GetTmuxVisibility returns a mocked visibility state.
// Configure the return value using:
//
//	mock.On("GetTmuxVisibility").Return(true, nil)
func (m *MockClient) GetTmuxVisibility() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

// SetTmuxVisibility returns a mocked error when setting visibility.
// Configure the return value using:
//
//	mock.On("SetTmuxVisibility", true).Return(nil)
func (m *MockClient) SetTmuxVisibility(visible bool) error {
	args := m.Called(visible)
	return args.Error(0)
}

// Run returns mocked stdout, stderr, and error for a tmux command.
// Configure the return value using:
//
//	mock.On("Run", "list-sessions", "-F", "#{session_name}").Return(
//	    "session1\nsession2", "", nil)
func (m *MockClient) Run(args ...string) (string, string, error) {
	callArgs := m.Called(args)
	return callArgs.String(0), callArgs.String(1), callArgs.Error(2)
}
