package settings

import "fmt"

// Validate checks that settings values are valid.
// Preconditions: settings must be non-nil.
func Validate(settings *Settings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}

	settings.GroupHeader.normalize()
	if err := settings.GroupHeader.Validate(); err != nil {
		return fmt.Errorf("invalid groupHeader options: %w", err)
	}
	if err := validateColumns(settings.Columns); err != nil {
		return err
	}
	if err := validateSortBy(settings.SortBy); err != nil {
		return err
	}
	if err := validateSortOrder(settings.SortOrder); err != nil {
		return err
	}
	if err := validateViewMode(settings.ViewMode); err != nil {
		return err
	}
	if err := validateGroupBySetting(settings.GroupBy); err != nil {
		return err
	}
	if err := validateExpandLevel(settings.DefaultExpandLevel); err != nil {
		return err
	}
	if err := validateFilters(settings.Filters); err != nil {
		return err
	}

	return nil
}

func validateColumns(columns []string) error {
	if len(columns) == 0 {
		return nil
	}
	validColumns := map[string]bool{
		ColumnID: true, ColumnTimestamp: true, ColumnState: true,
		ColumnSession: true, ColumnWindow: true, ColumnPane: true,
		ColumnMessage: true, ColumnPaneCreated: true, ColumnLevel: true,
	}
	for _, col := range columns {
		if !validColumns[col] {
			return fmt.Errorf("invalid column name: %s", col)
		}
	}
	return nil
}

func validateSortBy(sortBy string) error {
	if sortBy == "" {
		return nil
	}
	validSortBy := map[string]bool{
		SortByID: true, SortByTimestamp: true, SortByState: true,
		SortByLevel: true, SortBySession: true,
	}
	if !validSortBy[sortBy] {
		return fmt.Errorf("invalid sortBy value: %s", sortBy)
	}
	return nil
}

func validateSortOrder(order string) error {
	if order == "" {
		return nil
	}
	if order != SortOrderAsc && order != SortOrderDesc {
		return fmt.Errorf("invalid sortOrder value: %s", order)
	}
	return nil
}

func validateViewMode(mode string) error {
	if mode == "" {
		return nil
	}
	switch mode {
	case ViewModeCompact, ViewModeDetailed, ViewModeGrouped:
		return nil
	default:
		return fmt.Errorf("invalid viewMode value: %s", mode)
	}
}

func validateGroupBySetting(groupBy string) error {
	if groupBy == "" {
		return nil
	}
	if !IsValidGroupBy(groupBy) {
		return fmt.Errorf("invalid groupBy value: %s", groupBy)
	}
	return nil
}

func validateExpandLevel(level int) error {
	if level < MinExpandLevel || level > MaxExpandLevel {
		return fmt.Errorf("invalid defaultExpandLevel value: %d", level)
	}
	return nil
}

func validateFilters(filter Filter) error {
	validLevels := map[string]bool{
		"": true, LevelFilterInfo: true, LevelFilterWarning: true,
		LevelFilterError: true, LevelFilterCritical: true,
	}
	if !validLevels[filter.Level] {
		return fmt.Errorf("invalid filter level: %s", filter.Level)
	}

	validStates := map[string]bool{
		"": true, StateFilterActive: true, StateFilterDismissed: true,
	}
	if !validStates[filter.State] {
		return fmt.Errorf("invalid filter state: %s", filter.State)
	}

	validReadFilters := map[string]bool{
		"": true, ReadFilterRead: true, ReadFilterUnread: true,
	}
	if !validReadFilters[filter.Read] {
		return fmt.Errorf("invalid filter read value: %s", filter.Read)
	}

	return nil
}

// IsValidGroupBy returns true if groupBy is a supported grouping mode.
func IsValidGroupBy(groupBy string) bool {
	switch groupBy {
	case GroupByNone, GroupBySession, GroupByWindow, GroupByPane, GroupByMessage:
		return true
	case GroupByPaneMessage:
		return true
	default:
		return false
	}
}

// validate is an alias for Validate for internal use.
func validate(settings *Settings) error {
	return Validate(settings)
}
