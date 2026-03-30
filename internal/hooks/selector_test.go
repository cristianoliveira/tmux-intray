package hooks

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSelectorEmpty(t *testing.T) {
	selector, err := ParseSelector("   ")
	require.NoError(t, err)
	require.True(t, selector.Match(map[string]string{"LEVEL": "info"}))
}

func TestParseSelectorSingleClause(t *testing.T) {
	selector, err := ParseSelector("LEVEL==warning")
	require.NoError(t, err)

	require.True(t, selector.Match(map[string]string{"LEVEL": "warning"}))
	require.False(t, selector.Match(map[string]string{"LEVEL": "info"}))
}

func TestParseSelectorMultipleClauses(t *testing.T) {
	selector, err := ParseSelector("LEVEL==warning && SESSION!=build")
	require.NoError(t, err)

	require.True(t, selector.Match(map[string]string{"LEVEL": "warning", "SESSION": "ops"}))
	require.False(t, selector.Match(map[string]string{"LEVEL": "warning", "SESSION": "build"}))
	require.False(t, selector.Match(map[string]string{"LEVEL": "info", "SESSION": "ops"}))
}

func TestParseSelectorQuotedValue(t *testing.T) {
	selector, err := ParseSelector(`MESSAGE=="build failed: core"`)
	require.NoError(t, err)

	require.True(t, selector.Match(map[string]string{"MESSAGE": "build failed: core"}))
	require.False(t, selector.Match(map[string]string{"MESSAGE": "build failed"}))
}

func TestSelectorMatchMissingKey(t *testing.T) {
	selector, err := ParseSelector("SESSION!=build")
	require.NoError(t, err)

	require.False(t, selector.Match(map[string]string{"LEVEL": "warning"}))
}

func TestParseSelectorInvalid(t *testing.T) {
	tests := []struct {
		name        string
		selector    string
		errorSubstr string
	}{
		{name: "missing operator", selector: "LEVEL", errorSubstr: "missing operator == or !="},
		{name: "unsupported operator", selector: "LEVEL=warning", errorSubstr: "expected operator == or !="},
		{name: "missing key", selector: "==warning", errorSubstr: "missing key"},
		{name: "invalid key", selector: "level==warning", errorSubstr: "invalid key"},
		{name: "missing value", selector: "LEVEL==", errorSubstr: "missing value"},
		{name: "whitespace in bare value", selector: "MESSAGE==build failed", errorSubstr: "unquoted value cannot contain whitespace"},
		{name: "empty clause", selector: "LEVEL==warning &&", errorSubstr: "empty clause"},
		{name: "unterminated quoted value", selector: `MESSAGE=="oops`, errorSubstr: "invalid quoted value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSelector(tt.selector)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errorSubstr)
		})
	}
}

func TestEvaluateSelector(t *testing.T) {
	match, err := EvaluateSelector("LEVEL==info", map[string]string{"LEVEL": "info"})
	require.NoError(t, err)
	require.True(t, match)

	_, err = EvaluateSelector("LEVEL=info", map[string]string{"LEVEL": "info"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected operator == or !=")
}
