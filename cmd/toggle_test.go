package cmd

import (
	"errors"
	"testing"
)

func TestGetCurrentVisibility(t *testing.T) {
	originalGetVisibilityFunc := toggleGetVisibilityFunc
	defer func() { toggleGetVisibilityFunc = originalGetVisibilityFunc }()

	// Test visible
	toggleGetVisibilityFunc = func() string {
		return "1"
	}
	if !GetCurrentVisibility() {
		t.Error("Expected visibility true for '1'")
	}

	// Test hidden
	toggleGetVisibilityFunc = func() string {
		return "0"
	}
	if GetCurrentVisibility() {
		t.Error("Expected visibility false for '0'")
	}

	// Test other value defaults to hidden
	toggleGetVisibilityFunc = func() string {
		return "something"
	}
	if GetCurrentVisibility() {
		t.Error("Expected visibility false for non '1' value")
	}
}

func TestToggleSuccess(t *testing.T) {
	originalGetVisibilityFunc := toggleGetVisibilityFunc
	originalSetVisibilityFunc := toggleSetVisibilityFunc
	defer func() {
		toggleGetVisibilityFunc = originalGetVisibilityFunc
		toggleSetVisibilityFunc = originalSetVisibilityFunc
	}()

	var gotVisible bool
	toggleGetVisibilityFunc = func() string {
		return "0" // hidden initially
	}
	toggleSetVisibilityFunc = func(visible bool) error {
		gotVisible = visible
		return nil
	}

	visible, err := Toggle()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !visible {
		t.Error("Expected new visibility true after toggle from hidden")
	}
	if !gotVisible {
		t.Error("Expected setVisibilityFunc called with true")
	}
}

func TestToggleError(t *testing.T) {
	originalGetVisibilityFunc := toggleGetVisibilityFunc
	originalSetVisibilityFunc := toggleSetVisibilityFunc
	defer func() {
		toggleGetVisibilityFunc = originalGetVisibilityFunc
		toggleSetVisibilityFunc = originalSetVisibilityFunc
	}()

	expectedErr := errors.New("tmux operation failed")
	toggleGetVisibilityFunc = func() string {
		return "1"
	}
	toggleSetVisibilityFunc = func(visible bool) error {
		return expectedErr
	}

	visible, err := Toggle()
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if visible {
		t.Error("Expected visible false when error occurs")
	}
}

func TestToggleMultipleCalls(t *testing.T) {
	originalGetVisibilityFunc := toggleGetVisibilityFunc
	originalSetVisibilityFunc := toggleSetVisibilityFunc
	defer func() {
		toggleGetVisibilityFunc = originalGetVisibilityFunc
		toggleSetVisibilityFunc = originalSetVisibilityFunc
	}()

	calls := 0
	toggleGetVisibilityFunc = func() string {
		// Alternate between "0" and "1"
		if calls%2 == 0 {
			return "0"
		}
		return "1"
	}
	toggleSetVisibilityFunc = func(visible bool) error {
		calls++
		return nil
	}

	visible1, _ := Toggle()
	if !visible1 {
		t.Error("Expected first toggle to return true (from hidden)")
	}
	visible2, _ := Toggle()
	if visible2 {
		t.Error("Expected second toggle to return false (from visible)")
	}
	if calls != 2 {
		t.Errorf("Expected setVisibilityFunc to be called 2 times, got %d", calls)
	}
}
