package main

import (
	"errors"
	"testing"
)

func TestClearAllSuccess(t *testing.T) {
	originalClearAllFunc := clearAllFunc
	defer func() { clearAllFunc = originalClearAllFunc }()

	called := false
	clearAllFunc = func() error {
		called = true
		return nil
	}

	err := ClearAll()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !called {
		t.Error("Expected clearAllFunc to be called")
	}
}

func TestClearAllError(t *testing.T) {
	originalClearAllFunc := clearAllFunc
	defer func() { clearAllFunc = originalClearAllFunc }()

	expectedErr := errors.New("storage error")
	clearAllFunc = func() error {
		return expectedErr
	}

	err := ClearAll()
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestClearAllMultipleCalls(t *testing.T) {
	originalClearAllFunc := clearAllFunc
	defer func() { clearAllFunc = originalClearAllFunc }()

	count := 0
	clearAllFunc = func() error {
		count++
		return nil
	}

	_ = ClearAll()
	_ = ClearAll()
	if count != 2 {
		t.Errorf("Expected clearAllFunc to be called 2 times, got %d", count)
	}
}
