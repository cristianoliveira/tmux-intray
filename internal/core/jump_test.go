package core

import (
	"errors"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJumpService_JumpToNotification(t *testing.T) {
	tests := []struct {
		name               string
		notificationLine   string
		setupMock          func(*tmux.MockClient)
		expectError        bool
		expectedSuccess    bool
		expectedJumpToPane bool
		expectedSession    string
		expectedWindow     string
		expectedPane       string
		expectedMessage    string
		expectedErrorMsg   string
	}{
		{
			name:             "successful jump to pane",
			notificationLine: "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t%1\thello\t1234567890\tinfo",
			setupMock: func(m *tmux.MockClient) {
				m.On("HasSession").Return(true, nil).Once()
				m.On("ValidatePaneExists", "$0", "%0", "%1").Return(true, nil).Once()
				m.On("JumpToPane", "$0", "%0", "%1").Return(true, nil).Once()
			},
			expectError:        false,
			expectedSuccess:    true,
			expectedJumpToPane: true,
			expectedSession:    "$0",
			expectedWindow:     "%0",
			expectedPane:       "%1",
			expectedMessage:    "Jumped to $0:%0.%1",
		},
		{
			name:             "successful jump to window (pane missing)",
			notificationLine: "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t%1\thello\t1234567890\tinfo",
			setupMock: func(m *tmux.MockClient) {
				m.On("HasSession").Return(true, nil).Once()
				m.On("ValidatePaneExists", "$0", "%0", "%1").Return(false, nil).Once()
				m.On("JumpToPane", "$0", "%0", "").Return(true, nil).Once()
			},
			expectError:        false,
			expectedSuccess:    true,
			expectedJumpToPane: false,
			expectedSession:    "$0",
			expectedWindow:     "%0",
			expectedPane:       "",
			expectedMessage:    "Jumped to $0:%0 (pane not found)",
		},
		{
			name:             "notification has no session context",
			notificationLine: "42\t2025-02-04T10:00:00Z\tactive\t\t%0\t%1\thello\t1234567890\tinfo",
			setupMock: func(m *tmux.MockClient) {
				// No tmux calls expected
			},
			expectError:        false,
			expectedSuccess:    false,
			expectedJumpToPane: false,
			expectedMessage:    "notification has no tmux session context",
		},
		{
			name:             "tmux not running",
			notificationLine: "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t%1\thello\t1234567890\tinfo",
			setupMock: func(m *tmux.MockClient) {
				m.On("HasSession").Return(false, nil).Once()
			},
			expectError:        false,
			expectedSuccess:    false,
			expectedJumpToPane: false,
			expectedMessage:    "tmux not running",
		},
		{
			name:             "tmux HasSession error",
			notificationLine: "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t%1\thello\t1234567890\tinfo",
			setupMock: func(m *tmux.MockClient) {
				m.On("HasSession").Return(false, errors.New("tmux error")).Once()
			},
			expectError:      true,
			expectedErrorMsg: "check tmux running: tmux error",
		},
		{
			name:             "ValidatePaneExists error",
			notificationLine: "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t%1\thello\t1234567890\tinfo",
			setupMock: func(m *tmux.MockClient) {
				m.On("HasSession").Return(true, nil).Once()
				m.On("ValidatePaneExists", "$0", "%0", "%1").Return(false, errors.New("validation error")).Once()
			},
			expectError:      true,
			expectedErrorMsg: "validate pane exists: validation error",
		},
		{
			name:             "JumpToPane error",
			notificationLine: "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t%1\thello\t1234567890\tinfo",
			setupMock: func(m *tmux.MockClient) {
				m.On("HasSession").Return(true, nil).Once()
				m.On("ValidatePaneExists", "$0", "%0", "%1").Return(true, nil).Once()
				m.On("JumpToPane", "$0", "%0", "%1").Return(false, errors.New("jump error")).Once()
			},
			expectError:      true,
			expectedErrorMsg: "jump to pane: jump error",
		},
		{
			name:             "JumpToPane fallback error",
			notificationLine: "42\t2025-02-04T10:00:00Z\tactive\t$0\t%0\t%1\thello\t1234567890\tinfo",
			setupMock: func(m *tmux.MockClient) {
				m.On("HasSession").Return(true, nil).Once()
				m.On("ValidatePaneExists", "$0", "%0", "%1").Return(false, nil).Once()
				m.On("JumpToPane", "$0", "%0", "").Return(false, errors.New("jump error")).Once()
			},
			expectError:      true,
			expectedErrorMsg: "jump to window: jump error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(tmux.MockClient)
			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			service := NewJumpServiceWithDeps(mockClient, nil)

			result, err := service.JumpToNotificationParsed(&notification.Notification{
				ID:          42,
				Timestamp:   "2025-02-04T10:00:00Z",
				State:       "active",
				Session:     parseField(tt.notificationLine, 3),
				Window:      parseField(tt.notificationLine, 4),
				Pane:        parseField(tt.notificationLine, 5),
				Message:     "hello",
				PaneCreated: "1234567890",
				Level:       "info",
			})

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedSuccess, result.Success)
				assert.Equal(t, tt.expectedJumpToPane, result.JumpedToPane)
				assert.Equal(t, tt.expectedSession, result.Session)
				assert.Equal(t, tt.expectedWindow, result.Window)
				assert.Equal(t, tt.expectedPane, result.Pane)
				assert.Equal(t, tt.expectedMessage, result.Message)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestJumpService_JumpToContext(t *testing.T) {
	tests := []struct {
		name               string
		sessionID          string
		windowID           string
		paneID             string
		setupMock          func(*tmux.MockClient)
		expectError        bool
		expectedSuccess    bool
		expectedJumpToPane bool
		expectedSession    string
		expectedWindow     string
		expectedPane       string
		expectedMessage    string
		expectedErrorMsg   string
	}{
		{
			name:      "successful jump to pane",
			sessionID: "$0",
			windowID:  "%0",
			paneID:    "%1",
			setupMock: func(m *tmux.MockClient) {
				m.On("HasSession").Return(true, nil).Once()
				m.On("ValidatePaneExists", "$0", "%0", "%1").Return(true, nil).Once()
				m.On("JumpToPane", "$0", "%0", "%1").Return(true, nil).Once()
			},
			expectError:        false,
			expectedSuccess:    true,
			expectedJumpToPane: true,
			expectedSession:    "$0",
			expectedWindow:     "%0",
			expectedPane:       "%1",
			expectedMessage:    "Jumped to $0:%0.%1",
		},
		{
			name:      "successful jump to window (pane missing)",
			sessionID: "$0",
			windowID:  "%0",
			paneID:    "%1",
			setupMock: func(m *tmux.MockClient) {
				m.On("HasSession").Return(true, nil).Once()
				m.On("ValidatePaneExists", "$0", "%0", "%1").Return(false, nil).Once()
				m.On("JumpToPane", "$0", "%0", "").Return(true, nil).Once()
			},
			expectError:        false,
			expectedSuccess:    true,
			expectedJumpToPane: false,
			expectedSession:    "$0",
			expectedWindow:     "%0",
			expectedPane:       "",
			expectedMessage:    "Jumped to $0:%0 (pane not found)",
		},
		{
			name:      "jump to window only (no pane provided)",
			sessionID: "$0",
			windowID:  "%0",
			paneID:    "",
			setupMock: func(m *tmux.MockClient) {
				m.On("HasSession").Return(true, nil).Once()
				m.On("JumpToPane", "$0", "%0", "").Return(true, nil).Once()
			},
			expectError:        false,
			expectedSuccess:    true,
			expectedJumpToPane: false,
			expectedSession:    "$0",
			expectedWindow:     "%0",
			expectedPane:       "",
			expectedMessage:    "Jumped to $0:%0",
		},
		{
			name:      "empty session ID",
			sessionID: "",
			windowID:  "%0",
			paneID:    "%1",
			setupMock: func(m *tmux.MockClient) {
				// No tmux calls expected
			},
			expectError:        false,
			expectedSuccess:    false,
			expectedJumpToPane: false,
			expectedMessage:    "session id cannot be empty",
		},
		{
			name:      "empty window ID",
			sessionID: "$0",
			windowID:  "",
			paneID:    "%1",
			setupMock: func(m *tmux.MockClient) {
				// No tmux calls expected
			},
			expectError:        false,
			expectedSuccess:    false,
			expectedJumpToPane: false,
			expectedMessage:    "window id cannot be empty",
		},
		{
			name:      "tmux not running",
			sessionID: "$0",
			windowID:  "%0",
			paneID:    "%1",
			setupMock: func(m *tmux.MockClient) {
				m.On("HasSession").Return(false, nil).Once()
			},
			expectError:        false,
			expectedSuccess:    false,
			expectedJumpToPane: false,
			expectedMessage:    "tmux not running",
		},
		{
			name:      "tmux HasSession error",
			sessionID: "$0",
			windowID:  "%0",
			paneID:    "%1",
			setupMock: func(m *tmux.MockClient) {
				m.On("HasSession").Return(false, errors.New("tmux error")).Once()
			},
			expectError:      true,
			expectedErrorMsg: "check tmux running: tmux error",
		},
		{
			name:      "ValidatePaneExists error",
			sessionID: "$0",
			windowID:  "%0",
			paneID:    "%1",
			setupMock: func(m *tmux.MockClient) {
				m.On("HasSession").Return(true, nil).Once()
				m.On("ValidatePaneExists", "$0", "%0", "%1").Return(false, errors.New("validation error")).Once()
			},
			expectError:      true,
			expectedErrorMsg: "validate pane exists: validation error",
		},
		{
			name:      "JumpToPane error",
			sessionID: "$0",
			windowID:  "%0",
			paneID:    "%1",
			setupMock: func(m *tmux.MockClient) {
				m.On("HasSession").Return(true, nil).Once()
				m.On("ValidatePaneExists", "$0", "%0", "%1").Return(true, nil).Once()
				m.On("JumpToPane", "$0", "%0", "%1").Return(false, errors.New("jump error")).Once()
			},
			expectError:      true,
			expectedErrorMsg: "jump to pane: jump error",
		},
		{
			name:      "JumpToPane fallback error",
			sessionID: "$0",
			windowID:  "%0",
			paneID:    "%1",
			setupMock: func(m *tmux.MockClient) {
				m.On("HasSession").Return(true, nil).Once()
				m.On("ValidatePaneExists", "$0", "%0", "%1").Return(false, nil).Once()
				m.On("JumpToPane", "$0", "%0", "").Return(false, errors.New("jump error")).Once()
			},
			expectError:      true,
			expectedErrorMsg: "jump to window: jump error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(tmux.MockClient)
			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			service := NewJumpServiceWithDeps(mockClient, nil)

			result, err := service.JumpToContext(tt.sessionID, tt.windowID, tt.paneID)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedSuccess, result.Success)
				assert.Equal(t, tt.expectedJumpToPane, result.JumpedToPane)
				assert.Equal(t, tt.expectedSession, result.Session)
				assert.Equal(t, tt.expectedWindow, result.Window)
				assert.Equal(t, tt.expectedPane, result.Pane)
				assert.Equal(t, tt.expectedMessage, result.Message)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestJumpService_NewJumpService(t *testing.T) {
	t.Run("NewJumpService creates service with default client", func(t *testing.T) {
		service := NewJumpService()
		require.NotNil(t, service)
		require.NotNil(t, service.tmuxClient)
	})

	t.Run("NewJumpServiceWithDeps creates service with custom client", func(t *testing.T) {
		mockClient := new(tmux.MockClient)
		service := NewJumpServiceWithDeps(mockClient, nil)
		require.NotNil(t, service)
		assert.Same(t, mockClient, service.tmuxClient)
	})
}

// Helper function to parse a field from a TSV line by index
func parseField(line string, index int) string {
	fields := []string{}
	current := ""
	for i, c := range line {
		if c == '\t' {
			fields = append(fields, current)
			current = ""
		} else if i == len(line)-1 {
			current += string(c)
			fields = append(fields, current)
		} else {
			current += string(c)
		}
	}
	if len(fields) > index {
		return fields[index]
	}
	return ""
}
