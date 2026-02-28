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
		ShowHelp:              dto.ShowHelp,
		ActiveTab:             settings.NormalizeTab(string(dto.ActiveTab)),
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
	dto.ShowHelp = state.ShowHelp
	dto.ActiveTab = settings.NormalizeTab(string(state.ActiveTab))

	if err := uiState.FromDTO(dto); err != nil {
		return err
	}

	applyNonEmptyFilters(state.Filters, filters)

	return nil
}

// applyNonEmptyFilters copies non-empty filter values from source to dest.
func applyNonEmptyFilters(src settings.Filter, dest *settings.Filter) {
	if src.Level != "" {
		dest.Level = src.Level
	}
	if src.State != "" {
		dest.State = src.State
	}
	if src.Read != "" {
		dest.Read = src.Read
	}
	if src.Session != "" {
		dest.Session = src.Session
	}
	if src.Window != "" {
		dest.Window = src.Window
	}
	if src.Pane != "" {
		dest.Pane = src.Pane
	}
}

func (s *settingsService) save(state settings.TUIState) error {
	nextSettings := state.ToSettings()
	if s.loadedSettings != nil {
		nextSettings.GroupHeader = s.loadedSettings.GroupHeader.Clone()
	} else {
		defaults := settings.DefaultGroupHeaderOptions()
		nextSettings.GroupHeader = defaults
	}
	if s.loadedSettings != nil && reflect.DeepEqual(*s.loadedSettings, *nextSettings) {
		return nil
	}

	if err := settings.Save(nextSettings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	s.loadedSettings = nextSettings
	return nil
}
