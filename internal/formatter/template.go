// Package formatter provides template parsing, variable resolution, and preset management
// for formatting notification output with customizable templates and variables.
package formatter

import (
	"fmt"
	"regexp"
	"strings"
)

// TemplateEngine provides template parsing and variable substitution.
type TemplateEngine interface {
	// Parse returns a list of variables found in the template.
	Parse(template string) ([]string, error)

	// Substitute replaces variables in the template with values from the context.
	Substitute(template string, ctx VariableContext) (string, error)
}

// templateEngine implements TemplateEngine interface.
type templateEngine struct {
	variablePattern *regexp.Regexp
}

// NewTemplateEngine creates a new template engine instance.
func NewTemplateEngine() TemplateEngine {
	return &templateEngine{
		variablePattern: regexp.MustCompile(`\{\{([a-z0-9-]+)\}\}`),
	}
}

// Parse identifies all variables in a template string using {{variable-name}} syntax.
// Returns a list of variable names found, without duplicates.
func (te *templateEngine) Parse(template string) ([]string, error) {
	if template == "" {
		return []string{}, nil
	}

	matches := te.variablePattern.FindAllStringSubmatch(template, -1)
	if matches == nil {
		return []string{}, nil
	}

	// Use a map to track unique variables
	seen := make(map[string]bool)
	var variables []string

	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			if !seen[varName] {
				variables = append(variables, varName)
				seen[varName] = true
			}
		}
	}

	return variables, nil
}

// Substitute replaces all variables in the template with values from the context.
func (te *templateEngine) Substitute(template string, ctx VariableContext) (string, error) {
	if template == "" {
		return "", nil
	}

	resolver := NewVariableResolver()
	result := template

	// Find all variables
	matches := te.variablePattern.FindAllStringSubmatch(template, -1)
	if matches == nil {
		return result, nil
	}

	// Replace each variable with its value
	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			value, err := resolver.Resolve(varName, ctx)
			if err != nil {
				// Return error for unknown variables with available variables list
				return "", err
			}
			result = strings.ReplaceAll(result, match[0], value)
		}
	}

	return result, nil
}

// ValidateTemplate checks if a template has valid syntax.
func (te *templateEngine) ValidateTemplate(template string) error {
	if template == "" {
		return nil
	}

	// Check for unclosed braces
	openCount := strings.Count(template, "{{")
	closeCount := strings.Count(template, "}}")

	if openCount != closeCount {
		return fmt.Errorf("mismatched variable delimiters: %d opens, %d closes", openCount, closeCount)
	}

	return nil
}
