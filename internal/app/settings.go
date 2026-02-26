package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
)

// SettingsClient defines dependencies required by settings commands.
type SettingsClient interface {
	ResetSettings() (*settings.Settings, error)
	LoadSettings() (*settings.Settings, error)
}

// SettingsUseCase coordinates settings command behavior.
type SettingsUseCase struct {
	client SettingsClient
}

// NewSettingsUseCase creates a settings use-case.
func NewSettingsUseCase(client SettingsClient) *SettingsUseCase {
	if client == nil {
		panic("NewSettingsUseCase: client dependency cannot be nil")
	}

	return &SettingsUseCase{client: client}
}

// ResetSettingsInput contains reset options and environment adapters.
type ResetSettingsInput struct {
	Force     bool
	GetEnv    func(string) string
	ConfirmFn func() bool
}

// Reset executes settings reset behavior.
func (u *SettingsUseCase) Reset(input ResetSettingsInput) error {
	getEnv := input.GetEnv
	if getEnv == nil {
		getEnv = func(string) string { return "" }
	}

	if !input.Force && getEnv("CI") == "" && getEnv("BATS_TMPDIR") == "" {
		if input.ConfirmFn != nil && !input.ConfirmFn() {
			colors.Info("Operation cancelled")
			return nil
		}
	}

	_, err := u.client.ResetSettings()
	if err != nil {
		return fmt.Errorf("failed to reset settings: %w", err)
	}

	colors.Success("Settings reset to defaults")
	return nil
}

// Show loads current settings and writes JSON-formatted output.
func (u *SettingsUseCase) Show() error {
	currentSettings, err := u.client.LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	data, err := json.MarshalIndent(currentSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	colors.Info(strings.TrimSpace(string(data)))
	return nil
}
