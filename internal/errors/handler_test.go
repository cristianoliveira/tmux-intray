package errors

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockColorOutput is a mock implementation of ColorOutput for testing.
type mockColorOutput struct {
	mu          sync.Mutex
	errorCalled bool
	errorMsg    string

	warningCalled bool
	warningMsg    string

	infoCalled bool
	infoMsg    string

	successCalled bool
	successMsg    string
}

func (m *mockColorOutput) Error(msgs ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCalled = true
	if len(msgs) > 0 {
		m.errorMsg = msgs[0]
	}
}

func (m *mockColorOutput) Warning(msgs ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.warningCalled = true
	if len(msgs) > 0 {
		m.warningMsg = msgs[0]
	}
}

func (m *mockColorOutput) Info(msgs ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infoCalled = true
	if len(msgs) > 0 {
		m.infoMsg = msgs[0]
	}
}

func (m *mockColorOutput) Success(msgs ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.successCalled = true
	if len(msgs) > 0 {
		m.successMsg = msgs[0]
	}
}

func (m *mockColorOutput) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCalled = false
	m.errorMsg = ""
	m.warningCalled = false
	m.warningMsg = ""
	m.infoCalled = false
	m.infoMsg = ""
	m.successCalled = false
	m.successMsg = ""
}

// CLIHandler Tests

func TestCLIHandlerError(t *testing.T) {
	mock := &mockColorOutput{}
	handler := NewCLIHandler(mock)

	handler.Error("test error")

	assert.True(t, mock.errorCalled, "Error() should have been called")
	assert.Equal(t, "test error", mock.errorMsg, "Error() should have been called with correct message")
}

func TestNewDefaultCLIHandler(t *testing.T) {
	handler := NewDefaultCLIHandler()
	require.NotNil(t, handler)
}

func TestCLIHandlerWarning(t *testing.T) {
	mock := &mockColorOutput{}
	handler := NewCLIHandler(mock)

	handler.Warning("test warning")

	assert.True(t, mock.warningCalled, "Warning() should have been called")
	assert.Equal(t, "test warning", mock.warningMsg, "Warning() should have been called with correct message")
}

func TestCLIHandlerInfo(t *testing.T) {
	mock := &mockColorOutput{}
	handler := NewCLIHandler(mock)

	handler.Info("test info")

	assert.True(t, mock.infoCalled, "Info() should have been called")
	assert.Equal(t, "test info", mock.infoMsg, "Info() should have been called with correct message")
}

func TestCLIHandlerSuccess(t *testing.T) {
	mock := &mockColorOutput{}
	handler := NewCLIHandler(mock)

	handler.Success("test success")

	assert.True(t, mock.successCalled, "Success() should have been called")
	assert.Equal(t, "test success", mock.successMsg, "Success() should have been called with correct message")
}

func TestCLIHandlerRecursiveErrorHandling(t *testing.T) {
	// Create a mock that will trigger recursive error handling
	// The CLIHandler has an inHandling flag to prevent recursion
	mock := &mockColorOutput{}
	handler := NewCLIHandler(mock)

	// First error sets inHandling flag to true
	handler.Error("first error")
	require.True(t, mock.errorCalled, "First error should be handled")
	require.Equal(t, "first error", mock.errorMsg)

	mock.reset()

	// Second error while inHandling should skip the locking mechanism
	// and still call colors.Error()
	handler.Error("second error during handling")
	assert.True(t, mock.errorCalled, "Second error should still be handled")
	assert.Equal(t, "second error during handling", mock.errorMsg)

	// After the first error completes, inHandling should be reset
	mock.reset()

	// Third error should work normally
	handler.Error("third error")
	assert.True(t, mock.errorCalled, "Third error should be handled")
	assert.Equal(t, "third error", mock.errorMsg)
}

func TestCLIHandlerErrorWhenAlreadyHandling(t *testing.T) {
	mock := &mockColorOutput{}
	handler := NewCLIHandler(mock)

	handler.inHandling = true
	handler.Error("error while already handling")

	assert.True(t, mock.errorCalled, "Error() should be called even when already handling")
	assert.Equal(t, "error while already handling", mock.errorMsg)
	assert.True(t, handler.inHandling, "inHandling should stay true on fast path")
}

// TUIHandler Tests

func TestTUIHandlerError(t *testing.T) {
	callbackCalled := false
	var callbackMsg Message

	handler := NewTUIHandler(func(msg Message) {
		callbackCalled = true
		callbackMsg = msg
	})

	handler.Error("error message")

	require.True(t, callbackCalled, "Callback should have been called")
	require.Equal(t, "error message", callbackMsg.Text, "Callback should have correct message text")
	require.Equal(t, MessageTypeError, callbackMsg.Type, "Callback should have correct message type")

	// Verify message is stored
	latest, ok := handler.GetLatest()
	require.True(t, ok, "GetLatest should return true when messages exist")
	assert.Equal(t, "error message", latest.Text)
	assert.Equal(t, MessageTypeError, latest.Type)
	assert.False(t, latest.Timestamp.IsZero(), "Timestamp should be set")
}

func TestTUIHandlerWarning(t *testing.T) {
	callbackCalled := false
	var callbackMsg Message

	handler := NewTUIHandler(func(msg Message) {
		callbackCalled = true
		callbackMsg = msg
	})

	handler.Warning("warning message")

	require.True(t, callbackCalled, "Callback should have been called")
	require.Equal(t, "warning message", callbackMsg.Text, "Callback should have correct message text")
	require.Equal(t, MessageTypeWarning, callbackMsg.Type, "Callback should have correct message type")

	// Verify message is stored
	latest, ok := handler.GetLatest()
	require.True(t, ok, "GetLatest should return true when messages exist")
	assert.Equal(t, "warning message", latest.Text)
	assert.Equal(t, MessageTypeWarning, latest.Type)
}

func TestTUIHandlerInfo(t *testing.T) {
	callbackCalled := false
	var callbackMsg Message

	handler := NewTUIHandler(func(msg Message) {
		callbackCalled = true
		callbackMsg = msg
	})

	handler.Info("info message")

	require.True(t, callbackCalled, "Callback should have been called")
	require.Equal(t, "info message", callbackMsg.Text, "Callback should have correct message text")
	require.Equal(t, MessageTypeInfo, callbackMsg.Type, "Callback should have correct message type")

	// Verify message is stored
	latest, ok := handler.GetLatest()
	require.True(t, ok, "GetLatest should return true when messages exist")
	assert.Equal(t, "info message", latest.Text)
	assert.Equal(t, MessageTypeInfo, latest.Type)
}

func TestTUIHandlerSuccess(t *testing.T) {
	callbackCalled := false
	var callbackMsg Message

	handler := NewTUIHandler(func(msg Message) {
		callbackCalled = true
		callbackMsg = msg
	})

	handler.Success("success message")

	require.True(t, callbackCalled, "Callback should have been called")
	require.Equal(t, "success message", callbackMsg.Text, "Callback should have correct message text")
	require.Equal(t, MessageTypeSuccess, callbackMsg.Type, "Callback should have correct message type")

	// Verify message is stored
	latest, ok := handler.GetLatest()
	require.True(t, ok, "GetLatest should return true when messages exist")
	assert.Equal(t, "success message", latest.Text)
	assert.Equal(t, MessageTypeSuccess, latest.Type)
}

func TestTUIHandlerGetLatest(t *testing.T) {
	handler := NewTUIHandler(nil)

	// GetLatest on empty handler should return false
	_, ok := handler.GetLatest()
	assert.False(t, ok, "GetLatest should return false when no messages exist")

	// Add some messages
	handler.Info("first message")
	handler.Error("second message")
	handler.Warning("third message")

	// GetLatest should return the most recent message
	latest, ok := handler.GetLatest()
	require.True(t, ok, "GetLatest should return true when messages exist")
	assert.Equal(t, "third message", latest.Text)
	assert.Equal(t, MessageTypeWarning, latest.Type)
}

func TestTUIHandlerGetAll(t *testing.T) {
	handler := NewTUIHandler(nil)

	// GetAll on empty handler should return empty slice
	all := handler.GetAll()
	assert.Empty(t, all, "GetAll should return empty slice when no messages exist")

	// Add multiple messages
	handler.Error("error 1")
	handler.Warning("warning 2")
	handler.Info("info 3")
	handler.Success("success 4")

	// GetAll should return all messages in order
	all = handler.GetAll()
	require.Len(t, all, 4, "GetAll should return 4 messages")

	assert.Equal(t, "error 1", all[0].Text)
	assert.Equal(t, MessageTypeError, all[0].Type)

	assert.Equal(t, "warning 2", all[1].Text)
	assert.Equal(t, MessageTypeWarning, all[1].Type)

	assert.Equal(t, "info 3", all[2].Text)
	assert.Equal(t, MessageTypeInfo, all[2].Type)

	assert.Equal(t, "success 4", all[3].Text)
	assert.Equal(t, MessageTypeSuccess, all[3].Type)

	// Verify that modifying returned slice doesn't affect internal state
	all[0].Text = "modified text"
	allModified := handler.GetAll()
	assert.Equal(t, "error 1", allModified[0].Text, "Modifying returned slice should not affect internal state")
}

func TestTUIHandlerClear(t *testing.T) {
	handler := NewTUIHandler(nil)

	// Add some messages
	handler.Error("error 1")
	handler.Warning("warning 2")
	handler.Info("info 3")

	// Verify messages exist
	all := handler.GetAll()
	assert.Len(t, all, 3, "Should have 3 messages before clear")

	// Clear messages
	handler.Clear()

	// Verify all messages are cleared
	all = handler.GetAll()
	assert.Empty(t, all, "GetAll should return empty slice after clear")

	// GetLatest should also return false
	_, ok := handler.GetLatest()
	assert.False(t, ok, "GetLatest should return false after clear")
}

func TestTUIHandlerCallback(t *testing.T) {
	callbackCount := 0
	var callbackMessages []Message

	handler := NewTUIHandler(func(msg Message) {
		callbackCount++
		callbackMessages = append(callbackMessages, msg)
	})

	// Verify callback is called for each message type
	handler.Error("error message")
	handler.Warning("warning message")
	handler.Info("info message")
	handler.Success("success message")

	require.Equal(t, 4, callbackCount, "Callback should be called 4 times")
	require.Len(t, callbackMessages, 4, "Should have 4 callback messages")

	assert.Equal(t, "error message", callbackMessages[0].Text)
	assert.Equal(t, MessageTypeError, callbackMessages[0].Type)

	assert.Equal(t, "warning message", callbackMessages[1].Text)
	assert.Equal(t, MessageTypeWarning, callbackMessages[1].Type)

	assert.Equal(t, "info message", callbackMessages[2].Text)
	assert.Equal(t, MessageTypeInfo, callbackMessages[2].Type)

	assert.Equal(t, "success message", callbackMessages[3].Text)
	assert.Equal(t, MessageTypeSuccess, callbackMessages[3].Type)
}

func TestTUIHandlerNilCallback(t *testing.T) {
	// TUIHandler should work correctly with nil callback
	handler := NewTUIHandler(nil)

	// This should not panic
	handler.Error("error message")
	handler.Warning("warning message")
	handler.Info("info message")
	handler.Success("success message")

	// Verify messages are stored
	all := handler.GetAll()
	require.Len(t, all, 4, "Messages should be stored even with nil callback")
}

func TestTUIHandlerConcurrentAccess(t *testing.T) {
	handler := NewTUIHandler(nil)

	// Test concurrent writes
	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				handler.Info("message from goroutine")
			}
		}(i)
	}

	wg.Wait()

	// Verify all messages are stored
	all := handler.GetAll()
	expectedCount := numGoroutines * messagesPerGoroutine
	assert.Equal(t, expectedCount, len(all), "Should have stored all messages from concurrent access")

	// Test concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = handler.GetAll()
			_, _ = handler.GetLatest()
		}()
	}

	wg.Wait()
}
