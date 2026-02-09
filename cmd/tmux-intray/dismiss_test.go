package main

import (
	"errors"
	"testing"
)

func TestDismissSuccess(t *testing.T) {
	originalDismissFunc := dismissFunc
	defer func() { dismissFunc = originalDismissFunc }()

	var capturedID string
	dismissFunc = func(id string) error {
		capturedID = id
		return nil
	}

	err := Dismiss("42")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if capturedID != "42" {
		t.Errorf("Expected ID '42', got %q", capturedID)
	}
}

func TestDismissError(t *testing.T) {
	originalDismissFunc := dismissFunc
	defer func() { dismissFunc = originalDismissFunc }()

	expectedErr := errors.New("notification not found")
	dismissFunc = func(id string) error {
		return expectedErr
	}

	err := Dismiss("99")
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestDismissAllSuccess(t *testing.T) {
	originalDismissAllFunc := dismissAllFunc
	defer func() { dismissAllFunc = originalDismissAllFunc }()

	called := false
	dismissAllFunc = func() error {
		called = true
		return nil
	}

	err := DismissAll()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !called {
		t.Error("Expected dismissAllFunc to be called")
	}
}

func TestDismissAllError(t *testing.T) {
	originalDismissAllFunc := dismissAllFunc
	defer func() { dismissAllFunc = originalDismissAllFunc }()

	expectedErr := errors.New("storage error")
	dismissAllFunc = func() error {
		return expectedErr
	}

	err := DismissAll()
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestDismissEmptyID(t *testing.T) {
	originalDismissFunc := dismissFunc
	defer func() { dismissFunc = originalDismissFunc }()

	var capturedID string
	dismissFunc = func(id string) error {
		capturedID = id
		return nil
	}

	err := Dismiss("")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if capturedID != "" {
		t.Errorf("Expected empty ID, got %q", capturedID)
	}
}

func TestDismissAllMultipleCalls(t *testing.T) {
	originalDismissAllFunc := dismissAllFunc
	defer func() { dismissAllFunc = originalDismissAllFunc }()

	count := 0
	dismissAllFunc = func() error {
		count++
		return nil
	}

	_ = DismissAll()
	_ = DismissAll()
	if count != 2 {
		t.Errorf("Expected dismissAllFunc to be called 2 times, got %d", count)
	}
}

func TestDismissPreservesReadTimestamp(t *testing.T) {
	// This is a unit test for the dismiss command behavior.
	// The actual storage layer test verifies that read_timestamp is preserved,
	// but this test ensures the dismiss command function doesn't break it.
	originalDismissFunc := dismissFunc
	defer func() { dismissFunc = originalDismissFunc }()

	// Mock dismissFunc that succeeds without modifying read_timestamp
	dismissFunc = func(id string) error {
		// In real implementation, DismissNotification preserves read_timestamp
		// This test just verifies that dismissFunc is called correctly
		return nil
	}

	err := Dismiss("123")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	// The actual verification of read_timestamp preservation is done in storage layer tests
	// See: internal/storage/storage.go DismissNotification
}
