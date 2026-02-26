// Package formatter provides template parsing, variable resolution, and preset management
// for formatting notification output with customizable templates and variables.
package formatter

import "fmt"

// Preset represents a template preset with name, template string, and description.
type Preset struct {
	Name        string
	Template    string
	Description string
}

// PresetRegistry manages template presets.
type PresetRegistry interface {
	// Get returns a preset by name.
	Get(name string) (*Preset, error)

	// List returns all available presets.
	List() []Preset

	// Register adds a new preset.
	Register(preset Preset) error
}

// presetRegistry implements PresetRegistry interface.
type presetRegistry struct {
	presets map[string]Preset
	order   []string // To maintain order
}

// NewPresetRegistry creates a new preset registry with all default presets.
func NewPresetRegistry() PresetRegistry {
	registry := &presetRegistry{
		presets: make(map[string]Preset),
		order:   []string{},
	}

	// Register all 6 default presets
	registry.registerDefaults()

	return registry
}

// registerDefaults registers the 6 default presets.
func (pr *presetRegistry) registerDefaults() {
	presets := []Preset{
		{
			Name:        "compact",
			Template:    "[${unread-count}] ${latest-message}",
			Description: "Compact format showing unread count and latest message",
		},
		{
			Name:        "detailed",
			Template:    "${unread-count} unread, ${read-count} read | Latest: ${latest-message}",
			Description: "Detailed format with counts and latest message",
		},
		{
			Name:        "json",
			Template:    `{"unread":${unread-count},"total":${total-count},"message":"${latest-message}"}`,
			Description: "JSON format for programmatic consumption",
		},
		{
			Name:        "count-only",
			Template:    "${unread-count}",
			Description: "Only unread count",
		},
		{
			Name:        "levels",
			Template:    "Severity: ${highest-severity} | Unread: ${unread-count}",
			Description: "Format showing severity level and unread count",
		},
		{
			Name:        "panes",
			Template:    "${pane-list} (${unread-count})",
			Description: "Format showing pane list with count",
		},
	}

	for _, preset := range presets {
		pr.presets[preset.Name] = preset
		pr.order = append(pr.order, preset.Name)
	}
}

// Get returns a preset by name, or an error if not found.
func (pr *presetRegistry) Get(name string) (*Preset, error) {
	preset, ok := pr.presets[name]
	if !ok {
		return nil, fmt.Errorf("preset not found: %s", name)
	}
	return &preset, nil
}

// List returns all available presets in registration order.
func (pr *presetRegistry) List() []Preset {
	result := make([]Preset, 0, len(pr.order))
	for _, name := range pr.order {
		if preset, ok := pr.presets[name]; ok {
			result = append(result, preset)
		}
	}
	return result
}

// Register adds a new preset or overwrites an existing one.
func (pr *presetRegistry) Register(preset Preset) error {
	if preset.Name == "" {
		return fmt.Errorf("preset name cannot be empty")
	}

	if preset.Template == "" {
		return fmt.Errorf("preset template cannot be empty")
	}

	// Check if this is a new preset
	if _, exists := pr.presets[preset.Name]; !exists {
		pr.order = append(pr.order, preset.Name)
	}

	pr.presets[preset.Name] = preset
	return nil
}
