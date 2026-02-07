# Settings Package

## Overview

The `settings` package provides TUI user preferences persistence for tmux-intray.

## Settings Structure

Settings are persisted to `~/.config/tmux-intray/settings.json` as JSON:

```json
{
  "columns": ["id", "timestamp", "state", "level", "session", "window", "pane", "message"],
  "sortBy": "timestamp",
  "sortOrder": "desc",
  "filters": {
    "level": "",
    "state": "",
    "session": "",
    "window": "",
    "pane": ""
  },
  "viewMode": "compact"
}
```

## API

### Load()

```go
settings, err := settings.Load()
if err != nil {
    // handle error
}
```

Loads settings from disk. Returns default settings if file doesn't exist.

### Save()

```go
err := settings.Save(userSettings)
if err != nil {
    // handle error
}
```

Saves settings to disk, creating the config directory if needed.

### DefaultSettings()

```go
settings := settings.DefaultSettings()
```

Returns a Settings struct with all default values populated.

## Constants

### Column Names
- `ColumnID` = "id"
- `ColumnTimestamp` = "timestamp"
- `ColumnState` = "state"
- `ColumnSession` = "session"
- `ColumnWindow` = "window"
- `ColumnPane` = "pane"
- `ColumnMessage` = "message"
- `ColumnPaneCreated` = "pane_created"
- `ColumnLevel` = "level"

### Sort Direction
- `SortOrderAsc` = "asc"
- `SortOrderDesc` = "desc"

### Sort By
- `SortByID` = "id"
- `SortByTimestamp` = "timestamp"
- `SortByState` = "state"
- `SortByLevel` = "level"
- `SortBySession` = "session"

### View Mode
- `ViewModeCompact` = "compact"
- `ViewModeDetailed` = "detailed"

### Filter Levels
- `LevelFilterInfo` = "info"
- `LevelFilterWarning` = "warning"
- `LevelFilterError` = "error"
- `LevelFilterCritical` = "critical"

### Filter States
- `StateFilterActive` = "active"
- `StateFilterDismissed` = "dismissed"
