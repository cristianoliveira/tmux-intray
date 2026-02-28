# Tabs Architecture

## Overview

This document defines the tabs architecture for tmux-intray, establishing tabs as data views rather than UI modes. The architecture provides a clear pipeline for data selection, filtering, sorting, and rendering with well-defined ownership boundaries.

## Core Principles

### 1. Tabs are Dataset Selectors
Tabs function as **dataset selectors**, not UI modes. Each tab represents a different view of the same underlying data, filtered by specific criteria:

- **Recents Tab**: Shows recently active notifications (configurable limit, e.g., 20)
- **All Tab**: Shows the complete dataset of all notifications

### 2. Processing Pipeline
The tabs architecture follows a strict processing pipeline:

```
select dataset by tab → apply search/filters → apply sort → render
```

This pipeline ensures deterministic, predictable behavior across all tab views.

### 3. Active-Only MVP Rule
For MVP implementation, tabs operate on **active notifications only**:

- Dismissed notifications are filtered out at the dataset selection level
- The "dismissed" filter implication: dismissed items are automatically excluded from both Recents and All views
- This simplifies the initial implementation while providing core functionality

## Architecture Components

### Tab Contract (from tmux-intray-ow7)

The tabs architecture depends on a shared tab contract defined in tmux-intray-ow7. This contract specifies:

```go
type TabContract interface {
    // Dataset selection
    GetDataset(tabType TabType) ([]Notification, error)
    
    // Tab types
    GetTabTypes() []TabType
    SetActiveTab(tabType TabType) error
    GetActiveTab() TabType
    
    // Dataset filtering
    ApplyFilters(notifications []Notification, filters FilterSet) []Notification
    
    // Sorting
    ApplySort(notifications []Notification, sortConfig SortConfig) []Notification
}
```

### Ownership Boundaries

#### 1. State Layer (`internal/tui/state/`)
- **Owns**: Active tab state, filter state, sort state
- **Responsible for**: Tab persistence, state transitions
- **Does NOT**: Handle data loading, rendering logic

```go
type TabState struct {
    activeTab   TabType
    filters     FilterSet
    sortConfig  SortConfig
}
```

#### 2. Service Layer (`internal/tui/service/`)
- **Owns**: Tab contract implementation, dataset selection logic
- **Responsible for**: Data loading, filter application, sorting
- **Does NOT**: Handle UI rendering, input management

```go
type TabService struct {
    repository NotificationRepository
    tabConfig  TabConfiguration
}
```

#### 3. Render Layer (`internal/tui/render/`)
- **Owns**: Tab UI rendering, visual state representation
- **Responsible for**: Tab display, highlighting active tab
- **Does NOT**: Handle data selection, state management

#### 4. Input Layer (`internal/tui/state/`)
- **Owns**: Tab switching input handling
- **Responsible for**: Keybinding to tab transitions
- **Does NOT**: Handle data loading, rendering

## Data Flow

### Tab Selection Flow
1. User triggers tab switch (keybinding `r` for Recents, `a` for All)
2. Input layer validates and forwards to state layer
3. State layer updates active tab, persists state
4. State layer triggers data refresh via service layer
5. Service layer applies dataset selection, filters, sort
6. Render layer receives updated data and renders

### Dataset Selection Logic
```go
func (s *TabService) GetDataset(tabType TabType) ([]Notification, error) {
    switch tabType {
    case TabTypeRecents:
        // Apply recents filter + active-only rule
        return s.repository.LoadFilteredNotifications("active", "", "", "", "")
    case TabTypeAll:
        // Apply active-only rule only
        return s.repository.LoadFilteredNotifications("active", "", "", "", "")
    default:
        return nil, errors.New("unknown tab type")
    }
}
```

### Filter Pipeline
```go
func (s *TabService) ApplyFilters(notifications []Notification, filters FilterSet) []Notification {
    result := notifications
    
    // Apply dismissed filter (always active for MVP)
    result = filterDismissed(result)
    
    // Apply user filters
    if filters.SearchQuery != "" {
        result = filterBySearch(result, filters.SearchQuery)
    }
    
    if filters.SessionFilter != "" {
        result = filterBySession(result, filters.SessionFilter)
    }
    
    return result
}
```

## Implementation Guidelines

### State Management
- Tab state is persisted through the existing UI state persistence mechanism
- Active tab should be restored across TUI sessions
- Filter state is tab-specific and persisted separately

### Performance Considerations
- Dataset selection should be optimized for large datasets
- Recents tab uses database-level LIMIT for efficiency
- All tab should paginate for large datasets (future enhancement)

### Error Handling
- Invalid tab transitions should be handled gracefully
- Dataset loading failures should display user-friendly errors
- Tab switching should not corrupt UI state

## Integration Points

### With Existing TUI Architecture
- Tabs integrate with existing ViewMode system (compact, detailed, grouped)
- Tab state is separate from ViewMode state
- Both systems work together: tab selects dataset, ViewMode controls presentation

### With Storage Layer
- Tab service uses existing NotificationRepository interface
- Leverages existing filter capabilities in storage layer
- Maintains backward compatibility with current notification model

### With Input System
- Tab keybindings integrate with existing keybinding system
- Respects existing keybinding conflict resolution
- Uses existing input validation patterns

## Migration Path

### Phase 1: Core Tab Infrastructure
1. Implement tab contract interface
2. Add tab state to existing UI state
3. Implement basic tab service with dataset selection
4. Add tab keybindings and basic rendering

### Phase 2: Enhanced Filtering
1. Implement advanced filtering pipeline
2. Add search functionality specific to tabs
3. Optimize dataset selection for performance

### Phase 3: Advanced Features
1. Add custom tabs (user-defined filters)
2. Implement tab-specific settings
3. Add tab export/import functionality

## Testing Strategy

### Unit Tests
- Tab state transitions
- Dataset selection logic
- Filter pipeline operations
- Sort behavior verification

### Integration Tests
- Tab switching with real data
- State persistence across sessions
- Performance with large datasets
- Error handling scenarios

### UI Tests
- Tab rendering accuracy
- Keybinding functionality
- Visual state consistency

## Best Practices

### When Adding New Tabs
1. Define clear dataset selection criteria
2. Ensure performance with large datasets
3. Follow existing naming conventions
4. Add appropriate test coverage

### When Modifying Filter Pipeline
1. Maintain deterministic behavior
2. Preserve filter order consistency
3. Document filter performance implications
4. Test with various dataset sizes

### When Extending Tab Contract
1. Maintain backward compatibility
2. Update all implementations
3. Consider impact on persistence
4. Document breaking changes

## References

- [TUI Guidelines](./tui/tui-guidelines.md) - Overall TUI architecture
- [Import Layering Map](./import-layering-map.md) - Package dependency rules
- [Recents/All Implementation Plan](../../plans/recents-all-and-jump-navigation-breakdown.md) - Feature-specific details
- [tmux-intray-ow7] - Shared tab contract dependency