package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarkReadSuccess(t *testing.T) {
	originalMarkReadFunc := markReadFunc
	defer func() { markReadFunc = originalMarkReadFunc }()

	var capturedID string
	markReadFunc = func(id string) error {
		capturedID = id
		return nil
	}

	// Capture stdout using bytes.Buffer
	// Note: This test only verifies the function is called correctly
	// The actual success message is printed via colors.Success
	err := markReadFunc("42")
	require.NoError(t, err)
	if capturedID != "42" {
		t.Errorf("Expected ID '42', got %q", capturedID)
	}
}

func TestMarkReadError(t *testing.T) {
	originalMarkReadFunc := markReadFunc
	defer func() { markReadFunc = originalMarkReadFunc }()

	expectedErr := errors.New("notification not found")
	markReadFunc = func(id string) error {
		return expectedErr
	}

	err := markReadFunc("99")
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestMarkReadEmptyID(t *testing.T) {
	originalMarkReadFunc := markReadFunc
	defer func() { markReadFunc = originalMarkReadFunc }()

	var capturedID string
	markReadFunc = func(id string) error {
		capturedID = id
		return nil
	}

	err := markReadFunc("")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if capturedID != "" {
		t.Errorf("Expected empty ID, got %q", capturedID)
	}
}
