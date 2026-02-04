package cmd

import (
	"testing"
)

func TestAddSuccess(t *testing.T) {
	// Mock addFunc to return a fixed ID
	originalAddFunc := addFunc
	defer func() { addFunc = originalAddFunc }()
	var capturedArgs []string
	addFunc = func(item, session, window, pane, paneCreated string, noAuto bool, level string) string {
		capturedArgs = []string{item, session, window, pane, paneCreated, level}
		if noAuto {
			capturedArgs = append(capturedArgs, "true")
		} else {
			capturedArgs = append(capturedArgs, "false")
		}
		return "42"
	}

	opts := AddOptions{
		Message: "test message",
		Session: "sess1",
		Window:  "win1",
		Pane:    "pane1",
		Level:   "info",
	}
	id := Add(opts)
	if id != "42" {
		t.Errorf("Expected ID '42', got %q", id)
	}
	if capturedArgs[0] != "test message" {
		t.Errorf("Expected message 'test message', got %q", capturedArgs[0])
	}
	if capturedArgs[1] != "sess1" || capturedArgs[2] != "win1" || capturedArgs[3] != "pane1" {
		t.Errorf("Wrong context captured: %v", capturedArgs[1:4])
	}
	if capturedArgs[5] != "info" {
		t.Errorf("Expected level 'info', got %q", capturedArgs[5])
	}
	if capturedArgs[6] != "false" {
		t.Errorf("Expected noAuto false, got %q", capturedArgs[6])
	}
}

func TestAddNoAuto(t *testing.T) {
	originalAddFunc := addFunc
	defer func() { addFunc = originalAddFunc }()
	var capturedNoAuto bool
	addFunc = func(item, session, window, pane, paneCreated string, noAuto bool, level string) string {
		capturedNoAuto = noAuto
		return "99"
	}

	opts := AddOptions{
		Message: "msg",
		NoAuto:  true,
		Level:   "warning",
	}
	id := Add(opts)
	if id != "99" {
		t.Errorf("Expected ID '99', got %q", id)
	}
	if !capturedNoAuto {
		t.Error("Expected noAuto true")
	}
}

func TestAddEmptyLevelDefaults(t *testing.T) {
	originalAddFunc := addFunc
	defer func() { addFunc = originalAddFunc }()
	var capturedLevel string
	addFunc = func(item, session, window, pane, paneCreated string, noAuto bool, level string) string {
		capturedLevel = level
		return "1"
	}

	opts := AddOptions{
		Message: "msg",
	}
	Add(opts)
	// The default level is empty string, but core.AddTrayItem will default to "info".
	// That's outside the scope of this test.
	if capturedLevel != "" {
		t.Errorf("Expected empty level, got %q", capturedLevel)
	}
}

func TestAddFailure(t *testing.T) {
	originalAddFunc := addFunc
	defer func() { addFunc = originalAddFunc }()
	addFunc = func(item, session, window, pane, paneCreated string, noAuto bool, level string) string {
		return ""
	}

	opts := AddOptions{
		Message: "test",
	}
	id := Add(opts)
	if id != "" {
		t.Errorf("Expected empty ID on failure, got %q", id)
	}
}

func TestAddMessageWithNewlines(t *testing.T) {
	originalAddFunc := addFunc
	defer func() { addFunc = originalAddFunc }()
	var capturedMessage string
	addFunc = func(item, session, window, pane, paneCreated string, noAuto bool, level string) string {
		capturedMessage = item
		return "5"
	}

	message := "line1\nline2\t tab"
	opts := AddOptions{
		Message: message,
	}
	Add(opts)
	if capturedMessage != message {
		t.Errorf("Message mismatch: got %q, want %q", capturedMessage, message)
	}
}

func TestAddPaneCreated(t *testing.T) {
	originalAddFunc := addFunc
	defer func() { addFunc = originalAddFunc }()
	var capturedPaneCreated string
	addFunc = func(item, session, window, pane, paneCreated string, noAuto bool, level string) string {
		capturedPaneCreated = paneCreated
		return "7"
	}

	opts := AddOptions{
		Message:     "msg",
		PaneCreated: "1234567890",
	}
	Add(opts)
	if capturedPaneCreated != "1234567890" {
		t.Errorf("Expected pane_created '1234567890', got %q", capturedPaneCreated)
	}
}

func TestAddOptionsZeroValues(t *testing.T) {
	originalAddFunc := addFunc
	defer func() { addFunc = originalAddFunc }()
	var capturedArgs []string
	addFunc = func(item, session, window, pane, paneCreated string, noAuto bool, level string) string {
		capturedArgs = []string{item, session, window, pane, paneCreated, level}
		return "0"
	}

	opts := AddOptions{}
	Add(opts)
	// All fields should be empty strings
	for i, arg := range capturedArgs {
		if arg != "" {
			t.Errorf("Argument %d should be empty, got %q", i, arg)
		}
	}
}

func TestAddLevels(t *testing.T) {
	levels := []string{"info", "warning", "error", "critical"}
	for _, lvl := range levels {
		t.Run(lvl, func(t *testing.T) {
			originalAddFunc := addFunc
			defer func() { addFunc = originalAddFunc }()
			var capturedLevel string
			addFunc = func(item, session, window, pane, paneCreated string, noAuto bool, level string) string {
				capturedLevel = level
				return "1"
			}
			opts := AddOptions{
				Message: "msg",
				Level:   lvl,
			}
			Add(opts)
			if capturedLevel != lvl {
				t.Errorf("Expected level %q, got %q", lvl, capturedLevel)
			}
		})
	}
}
