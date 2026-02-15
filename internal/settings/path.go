package settings

import (
	"os"
	"path/filepath"

	"github.com/cristianoliveira/tmux-intray/internal/config"
)

const tuiSettingsFilename = "tui" + FileExtTOML

// getSettingsPath returns the filesystem path for the TUI settings file.
// It respects the optional tui_settings_path override.
func getSettingsPath() string {
	if override := config.Get("tui_settings_path", ""); override != "" {
		return override
	}

	configDir := resolveConfigDir()
	settingsPath := filepath.Join(configDir, tuiSettingsFilename)
	return settingsPath
}

// resolveConfigDir returns the configured tmux-intray config directory,
// falling back to the XDG default if needed.
func resolveConfigDir() string {
	configDir := config.Get("config_dir", "")
	if configDir != "" {
		return configDir
	}
	home, _ := os.UserHomeDir()
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		xdgConfigHome = filepath.Join(home, ".config")
	}
	return filepath.Join(xdgConfigHome, "tmux-intray")
}
