package state

import (
	"fmt"
	"reflect"

	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

type settingsService struct {
	loadedSettings *settings.Settings
}

func newSettingsService() *settingsService {
	return &settingsService{}
}

func (s *settingsService) setLoadedSettings(loaded *settings.Settings) {
	s.loadedSettings = loaded
}

func (s *settingsService) toState(uiState *UIState, columns []string, sortBy string, sortOrder string, filters settings.Filter) settings.TUIState {
	dto := uiState.ToDTO()
	return settings.TUIState{
		Columns:               columns,
		SortBy:                sortBy,
		SortOrder:             sortOrder,
		Filters:               filters,
		ViewMode:              string(dto.ViewMode),
		GroupBy:               string(dto.GroupBy),
		DefaultExpandLevel:    dto.ExpandLevel,
		DefaultExpandLevelSet: true,
		ExpansionState:        dto.ExpansionState,
	}
}

func (s *settingsService) fromState(state settings.TUIState, uiState *UIState, columns *[]string, sortBy *string, sortOrder *string, filters *settings.Filter) error {
	if state.GroupBy != "" && !settings.IsValidGroupBy(state.GroupBy) {
		return fmt.Errorf("invalid groupBy value: %s", state.GroupBy)
	}
	if state.DefaultExpandLevelSet {
		if state.DefaultExpandLevel < settings.MinExpandLevel || state.DefaultExpandLevel > settings.MaxExpandLevel {
			return fmt.Errorf("invalid defaultExpandLevel value: %d", state.DefaultExpandLevel)
		}
	}

	if len(state.Columns) > 0 {
		*columns = state.Columns
	}
	if state.SortBy != "" {
		*sortBy = state.SortBy
	}
	if state.SortOrder != "" {
		*sortOrder = state.SortOrder
	}

	dto := model.UIDTO{}
	if state.ViewMode != "" {
		dto.ViewMode = model.ViewMode(state.ViewMode)
	}
	if state.GroupBy != "" {
		dto.GroupBy = model.GroupBy(state.GroupBy)
	}
	if state.DefaultExpandLevelSet {
		dto.ExpandLevel = state.DefaultExpandLevel
		dto.ExpandLevelSet = true
	}
	if state.ExpansionState != nil {
		dto.ExpansionState = state.ExpansionState
	}

	if err := uiState.FromDTO(dto); err != nil {
		return err
	}

	if state.Filters.Level != "" ||
		state.Filters.State != "" ||
		state.Filters.Session != "" ||
		state.Filters.Window != "" ||
		state.Filters.Pane != "" {
		if state.Filters.Level != "" {
			filters.Level = state.Filters.Level
		}
		if state.Filters.State != "" {
			filters.State = state.Filters.State
		}
		if state.Filters.Session != "" {
			filters.Session = state.Filters.Session
		}
		if state.Filters.Window != "" {
			filters.Window = state.Filters.Window
		}
		if state.Filters.Pane != "" {
			filters.Pane = state.Filters.Pane
		}
	}

	return nil
}

func (s *settingsService) save(state settings.TUIState) error {
	nextSettings := state.ToSettings()
	if s.loadedSettings != nil && reflect.DeepEqual(*s.loadedSettings, *nextSettings) {
		return nil
	}

	if err := settings.Save(nextSettings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	s.loadedSettings = nextSettings
	return nil
}
