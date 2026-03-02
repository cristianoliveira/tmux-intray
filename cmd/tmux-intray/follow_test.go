package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

type fakeFollowClient struct {
	calls []struct {
		state      string
		level      string
		session    string
		window     string
		pane       string
		olderThan  string
		newerThan  string
		readFilter string
	}
	result string
	err    error
}

func (f *fakeFollowClient) ListNotifications(state, level, session, window, pane, olderThan, newerThan, readFilter string) (string, error) {
	f.calls = append(f.calls, struct {
		state      string
		level      string
		session    string
		window     string
		pane       string
		olderThan  string
		newerThan  string
		readFilter string
	}{
		state: state, level: level, session: session, window: window,
		pane: pane, olderThan: olderThan, newerThan: newerThan, readFilter: readFilter,
	})
	return f.result, f.err
}

func TestNewFollowCmdPanicsWhenClientIsNil(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic, got nil")
		}

		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected panic message as string, got %T", r)
		}
		if !strings.Contains(msg, "client dependency cannot be nil") {
			t.Fatalf("expected panic message to mention nil dependency, got %q", msg)
		}
	}()

	NewFollowCmd(nil)
}

func TestFollowCmdWithClient(t *testing.T) {
	// Create tick channel we can control
	tickChan := make(chan time.Time)
	defer close(tickChan)

	// Create fake client
	client := &fakeFollowClient{
		result: "1\t2025-01-01T12:00:00Z\tactive\t\t\tpane1\thello\t\tinfo",
	}

	// Capture output
	var buf bytes.Buffer
	opts := FollowOptions{
		Client:   client,
		State:    "active",
		TickChan: tickChan,
		Output:   &buf,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run follow in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- Follow(ctx, opts)
	}()

	// Trigger first tick
	tickChan <- time.Now()
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Wait for follow to exit
	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("Follow returned error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Follow did not exit after cancellation")
	}

	// Verify client was called
	if len(client.calls) != 1 {
		t.Fatalf("expected 1 call to ListNotifications, got %d", len(client.calls))
	}
	if client.calls[0].state != "active" {
		t.Errorf("expected state 'active', got %q", client.calls[0].state)
	}

	// Check output
	if !strings.Contains(buf.String(), "hello") {
		t.Error("Expected output to contain 'hello'")
	}
}

func TestFollowNewNotifications(t *testing.T) {
	// Mock listFunc to return different lines on each call
	calls := 0
	originalListFunc := listFunc
	defer func() { listFunc = originalListFunc }()
	listFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) (string, error) {
		calls++
		switch calls {
		case 1:
			return "1\t2025-01-01T12:00:00Z\tactive\t\t\tpane1\thello\t\tinfo", nil
		case 2:
			return "1\t2025-01-01T12:00:00Z\tactive\t\t\tpane1\thello\t\tinfo\n2\t2025-01-01T12:00:01Z\tactive\t\t\tpane2\tworld\t\twarning", nil
		default:
			return "", nil
		}
	}

	// Create tick channel we can control
	tickChan := make(chan time.Time)
	defer close(tickChan)

	// Capture output
	var buf bytes.Buffer
	opts := FollowOptions{
		State:    "active",
		TickChan: tickChan,
		Output:   &buf,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run follow in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- Follow(ctx, opts)
	}()

	// Trigger first tick
	tickChan <- time.Now()
	// Wait a bit for processing
	time.Sleep(10 * time.Millisecond)
	// Trigger second tick
	tickChan <- time.Now()
	// Wait a bit
	time.Sleep(10 * time.Millisecond)
	// Cancel context to stop follow
	cancel()

	// Wait for follow to exit
	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("Follow returned error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Follow did not exit after cancellation")
	}

	output := buf.String()
	// Check that both notifications appear
	if !strings.Contains(output, "hello") {
		t.Error("Expected output to contain 'hello'")
	}
	if !strings.Contains(output, "world") {
		t.Error("Expected output to contain 'world'")
	}
	// Ensure each notification printed only once
	if strings.Count(output, "hello") != 1 {
		t.Errorf("Expected 'hello' to appear exactly once, got %d", strings.Count(output, "hello"))
	}
	if strings.Count(output, "world") != 1 {
		t.Errorf("Expected 'world' to appear exactly once, got %d", strings.Count(output, "world"))
	}
}

func TestFollowFilters(t *testing.T) {
	originalListFunc := listFunc
	defer func() { listFunc = originalListFunc }()
	var capturedState, capturedLevel, capturedPane string
	listFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) (string, error) {
		capturedState = state
		capturedLevel = level
		capturedPane = pane
		return "", nil
	}

	tickChan := make(chan time.Time)
	defer close(tickChan)
	var buf bytes.Buffer
	opts := FollowOptions{
		State:    "dismissed",
		Level:    "error",
		Pane:     "%123",
		TickChan: tickChan,
		Output:   &buf,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- Follow(ctx, opts)
	}()

	tickChan <- time.Now()
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("Follow returned error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Follow did not exit after cancellation")
	}

	if capturedState != "dismissed" {
		t.Errorf("Expected state filter 'dismissed', got %q", capturedState)
	}
	if capturedLevel != "error" {
		t.Errorf("Expected level filter 'error', got %q", capturedLevel)
	}
	if capturedPane != "%123" {
		t.Errorf("Expected pane filter '%%123', got %q", capturedPane)
	}
}

func TestFollowEmptyLines(t *testing.T) {
	originalListFunc := listFunc
	defer func() { listFunc = originalListFunc }()
	calls := 0
	listFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) (string, error) {
		calls++
		// Return empty string on first call, then a notification on second
		if calls == 1 {
			return "", nil
		}
		return "1\t2025-01-01T12:00:00Z\tactive\t\t\t\tmessage\t\tinfo", nil
	}

	tickChan := make(chan time.Time)
	defer close(tickChan)
	var buf bytes.Buffer
	opts := FollowOptions{
		TickChan: tickChan,
		Output:   &buf,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- Follow(ctx, opts)
	}()

	// First tick - empty lines
	tickChan <- time.Now()
	time.Sleep(10 * time.Millisecond)
	// Second tick - with notification
	tickChan <- time.Now()
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("Follow returned error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Follow did not exit after cancellation")
	}

	if !strings.Contains(buf.String(), "message") {
		t.Error("Expected notification 'message' to appear")
	}
}

func TestFollowDuplicateNotification(t *testing.T) {
	originalListFunc := listFunc
	defer func() { listFunc = originalListFunc }()
	listFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) (string, error) {
		// Return same notification twice
		return "1\t2025-01-01T12:00:00Z\tactive\t\t\t\tduplicate\t\tinfo", nil
	}

	tickChan := make(chan time.Time)
	defer close(tickChan)
	var buf bytes.Buffer
	opts := FollowOptions{
		TickChan: tickChan,
		Output:   &buf,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- Follow(ctx, opts)
	}()

	// Two ticks
	tickChan <- time.Now()
	time.Sleep(10 * time.Millisecond)
	tickChan <- time.Now()
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("Follow returned error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Follow did not exit after cancellation")
	}

	// Should appear only once
	count := strings.Count(buf.String(), "duplicate")
	if count != 1 {
		t.Errorf("Expected duplicate notification to appear once, got %d", count)
	}
}

func TestFollowColorOutput(t *testing.T) {
	originalListFunc := listFunc
	defer func() { listFunc = originalListFunc }()
	listFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) (string, error) {
		return "1\t2025-01-01T12:00:00Z\tactive\t\t\t\ttest\t\terror", nil
	}

	tickChan := make(chan time.Time)
	defer close(tickChan)
	var buf bytes.Buffer
	opts := FollowOptions{
		TickChan: tickChan,
		Output:   &buf,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- Follow(ctx, opts)
	}()

	tickChan <- time.Now()
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("Follow returned error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Follow did not exit after cancellation")
	}

	output := buf.String()
	// Check that red color code appears
	if !strings.Contains(output, "\033[0;31m") {
		t.Error("Expected red color code for error level")
	}
	if !strings.Contains(output, "\033[0m") {
		t.Error("Expected reset color code")
	}
}

func TestFollowPaneInfo(t *testing.T) {
	originalListFunc := listFunc
	defer func() { listFunc = originalListFunc }()
	listFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) (string, error) {
		return "1\t2025-01-01T12:00:00Z\tactive\t\t\t%123\tmessage\t\tinfo", nil
	}

	tickChan := make(chan time.Time)
	defer close(tickChan)
	var buf bytes.Buffer
	opts := FollowOptions{
		TickChan: tickChan,
		Output:   &buf,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- Follow(ctx, opts)
	}()

	tickChan <- time.Now()
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("Follow returned error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Follow did not exit after cancellation")
	}

	output := buf.String()
	if !strings.Contains(output, "└─ From pane: %123") {
		t.Error("Expected pane info line not found")
	}
}

func TestFollowDefaultInterval(t *testing.T) {
	opts := FollowOptions{}
	if opts.Interval != 0 {
		t.Errorf("Expected zero default interval, got %v", opts.Interval)
	}
	// Should be set inside Follow
}

func TestFollowDefaultOutput(t *testing.T) {
	// This test is just to ensure no panic
	originalListFunc := listFunc
	defer func() { listFunc = originalListFunc }()
	listFunc = func(state, level, session, window, pane, olderThan, newerThan, readFilter string) (string, error) {
		return "", nil
	}
	tickChan := make(chan time.Time)
	defer close(tickChan)
	opts := FollowOptions{
		TickChan: tickChan,
		// Output nil -> defaults to os.Stdout
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- Follow(ctx, opts)
	}()

	tickChan <- time.Now()
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("Follow returned error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Follow did not exit after cancellation")
	}
}
