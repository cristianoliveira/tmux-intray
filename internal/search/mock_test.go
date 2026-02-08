package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMockProvider tests the mock provider implementation.
func TestMockProvider(t *testing.T) {
	mockProvider := new(MockProvider)

	// Set up mock expectations
	mockProvider.On("Name").Return("mock-provider")
	mockProvider.On("Match", testNotification, "test").Return(true)
	mockProvider.On("Match", testNotification, "other").Return(false)

	// Test Name
	assert.Equal(t, "mock-provider", mockProvider.Name())

	// Test Match
	result := mockProvider.Match(testNotification, "test")
	assert.True(t, result)

	result2 := mockProvider.Match(testNotification, "other")
	assert.False(t, result2)

	// Assert expectations were met
	mockProvider.AssertExpectations(t)
}

// TestMockProviderMethod tests mock provider methods work correctly.
func TestMockProviderMethod(t *testing.T) {
	mockProvider := new(MockProvider)

	// Set up expectations
	mockProvider.On("Match", testNotification, "query1").Return(true)
	mockProvider.On("Match", testNotification, "query2").Return(false)
	mockProvider.On("Name").Return("mock-search")

	// Test Match
	result1 := mockProvider.Match(testNotification, "query1")
	assert.True(t, result1)

	result2 := mockProvider.Match(testNotification, "query2")
	assert.False(t, result2)

	// Test Name
	name := mockProvider.Name()
	assert.Equal(t, "mock-search", name)

	// Verify expectations
	mockProvider.AssertExpectations(t)
	mockProvider.AssertNumberOfCalls(t, "Match", 2)
	mockProvider.AssertNumberOfCalls(t, "Name", 1)
}
