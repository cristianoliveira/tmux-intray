package formatter

import (
	"testing"
)

func TestNewPresetRegistry_DefaultPresets(t *testing.T) {
	registry := NewPresetRegistry()
	presets := registry.List()

	if len(presets) != 6 {
		t.Errorf("Expected 6 default presets, got %d", len(presets))
	}

	expectedNames := []string{"compact", "detailed", "json", "count-only", "levels", "panes"}
	for i, expected := range expectedNames {
		if i >= len(presets) {
			t.Errorf("Expected preset %s at index %d, but presets list is shorter", expected, i)
			continue
		}
		if presets[i].Name != expected {
			t.Errorf("Expected preset name %q at index %d, got %q", expected, i, presets[i].Name)
		}
	}
}

func TestPresetRegistry_Get(t *testing.T) {
	registry := NewPresetRegistry()

	tests := []struct {
		name          string
		presetName    string
		shouldExist   bool
		expectedTempl string
	}{
		{
			name:          "get compact preset",
			presetName:    "compact",
			shouldExist:   true,
			expectedTempl: "[{{unread-count}}] {{latest-message}}",
		},
		{
			name:          "get detailed preset",
			presetName:    "detailed",
			shouldExist:   true,
			expectedTempl: "{{unread-count}} unread, {{read-count}} read | Latest: {{latest-message}}",
		},
		{
			name:          "get json preset",
			presetName:    "json",
			shouldExist:   true,
			expectedTempl: `{"unread":{{unread-count}},"total":{{total-count}},"message":"{{latest-message}}"}`,
		},
		{
			name:          "get count-only preset",
			presetName:    "count-only",
			shouldExist:   true,
			expectedTempl: "{{unread-count}}",
		},
		{
			name:          "get levels preset",
			presetName:    "levels",
			shouldExist:   true,
			expectedTempl: "Severity: {{highest-severity}} | Unread: {{unread-count}}",
		},
		{
			name:          "get panes preset",
			presetName:    "panes",
			shouldExist:   true,
			expectedTempl: "{{pane-list}} ({{unread-count}})",
		},
		{
			name:        "get nonexistent preset",
			presetName:  "nonexistent",
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preset, err := registry.Get(tt.presetName)
			if tt.shouldExist {
				if err != nil {
					t.Errorf("Expected to find preset %q, got error: %v", tt.presetName, err)
					return
				}
				if preset.Template != tt.expectedTempl {
					t.Errorf("Expected template %q, got %q", tt.expectedTempl, preset.Template)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for preset %q, but got none", tt.presetName)
				}
			}
		})
	}
}

func TestPresetRegistry_List(t *testing.T) {
	registry := NewPresetRegistry()
	presets := registry.List()

	if len(presets) != 6 {
		t.Errorf("Expected 6 presets in list, got %d", len(presets))
		return
	}

	// Verify all presets have required fields
	for _, preset := range presets {
		if preset.Name == "" {
			t.Error("Preset has empty name")
		}
		if preset.Template == "" {
			t.Error("Preset has empty template")
		}
		if preset.Description == "" {
			t.Errorf("Preset %q has empty description", preset.Name)
		}
	}
}

func TestPresetRegistry_Register(t *testing.T) {
	registry := NewPresetRegistry()

	// Get initial count
	initialCount := len(registry.List())

	// Register a new preset
	newPreset := Preset{
		Name:        "custom",
		Template:    "Custom: {{unread-count}}",
		Description: "Custom preset for testing",
	}

	err := registry.Register(newPreset)
	if err != nil {
		t.Errorf("Failed to register preset: %v", err)
		return
	}

	// Verify it was added
	preset, err := registry.Get("custom")
	if err != nil {
		t.Errorf("Failed to get registered preset: %v", err)
		return
	}

	if preset.Template != newPreset.Template {
		t.Errorf("Expected template %q, got %q", newPreset.Template, preset.Template)
	}

	// Verify count increased
	finalCount := len(registry.List())
	if finalCount != initialCount+1 {
		t.Errorf("Expected %d presets, got %d", initialCount+1, finalCount)
	}
}

func TestPresetRegistry_RegisterOverwrite(t *testing.T) {
	registry := NewPresetRegistry()

	// Get compact preset
	original, _ := registry.Get("compact")

	// Register new preset with same name
	updated := Preset{
		Name:        "compact",
		Template:    "New: {{unread-count}}",
		Description: "Updated compact preset",
	}

	err := registry.Register(updated)
	if err != nil {
		t.Errorf("Failed to register preset: %v", err)
		return
	}

	// Verify it was updated
	preset, _ := registry.Get("compact")
	if preset.Template == original.Template {
		t.Error("Preset was not updated")
	}
	if preset.Template != updated.Template {
		t.Errorf("Expected template %q, got %q", updated.Template, preset.Template)
	}

	// Verify count didn't increase
	count := len(registry.List())
	if count != 6 {
		t.Errorf("Expected 6 presets after overwrite, got %d", count)
	}
}

func TestPresetRegistry_RegisterValidation(t *testing.T) {
	registry := NewPresetRegistry()

	tests := []struct {
		name      string
		preset    Preset
		shouldErr bool
	}{
		{
			name: "empty name",
			preset: Preset{
				Name:        "",
				Template:    "Template",
				Description: "Description",
			},
			shouldErr: true,
		},
		{
			name: "empty template",
			preset: Preset{
				Name:        "test",
				Template:    "",
				Description: "Description",
			},
			shouldErr: true,
		},
		{
			name: "valid preset",
			preset: Preset{
				Name:        "test",
				Template:    "Template",
				Description: "Description",
			},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Register(tt.preset)
			if (err != nil) != tt.shouldErr {
				t.Errorf("Register() error = %v, shouldErr %v", err, tt.shouldErr)
			}
		})
	}
}

func TestPresetRegistry_CompactPreset(t *testing.T) {
	registry := NewPresetRegistry()
	preset, _ := registry.Get("compact")

	engine := NewTemplateEngine()
	vars, _ := engine.Parse(preset.Template)

	expectedVars := []string{"unread-count", "latest-message"}
	if len(vars) != len(expectedVars) {
		t.Errorf("Expected %d variables, got %d", len(expectedVars), len(vars))
	}
}

func TestPresetRegistry_DetailedPreset(t *testing.T) {
	registry := NewPresetRegistry()
	preset, _ := registry.Get("detailed")

	engine := NewTemplateEngine()
	vars, _ := engine.Parse(preset.Template)

	expectedVars := []string{"unread-count", "read-count", "latest-message"}
	if len(vars) != len(expectedVars) {
		t.Errorf("Expected %d variables, got %d", len(expectedVars), len(vars))
	}
}

func TestPresetRegistry_JSONPreset(t *testing.T) {
	registry := NewPresetRegistry()
	preset, _ := registry.Get("json")

	if preset.Template == "" {
		t.Error("JSON preset template is empty")
	}

	// Basic validation that it looks like JSON template
	if !contains(preset.Template, "{") || !contains(preset.Template, "}") {
		t.Error("JSON preset doesn't contain expected JSON structure")
	}
}

func TestPresetRegistry_CountOnlyPreset(t *testing.T) {
	registry := NewPresetRegistry()
	preset, _ := registry.Get("count-only")

	engine := NewTemplateEngine()
	vars, _ := engine.Parse(preset.Template)

	if len(vars) != 1 || vars[0] != "unread-count" {
		t.Errorf("Count-only preset should have only unread-count variable, got %v", vars)
	}
}

func TestPresetRegistry_LevelsPreset(t *testing.T) {
	registry := NewPresetRegistry()
	preset, _ := registry.Get("levels")

	engine := NewTemplateEngine()
	vars, _ := engine.Parse(preset.Template)

	hasHighestSeverity := false
	for _, v := range vars {
		if v == "highest-severity" {
			hasHighestSeverity = true
		}
	}

	if !hasHighestSeverity {
		t.Error("Levels preset should have highest-severity variable")
	}
}

func TestPresetRegistry_PanesPreset(t *testing.T) {
	registry := NewPresetRegistry()
	preset, _ := registry.Get("panes")

	engine := NewTemplateEngine()
	vars, _ := engine.Parse(preset.Template)

	hasPaneList := false
	for _, v := range vars {
		if v == "pane-list" {
			hasPaneList = true
		}
	}

	if !hasPaneList {
		t.Error("Panes preset should have pane-list variable")
	}
}

// Helper function for string containment
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
