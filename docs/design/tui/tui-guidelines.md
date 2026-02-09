# TUI Guidelines and Resilience Standards

## Overview

This document defines the TUI architecture boundaries, package layout, and
resiliency practices for `tmux-intray`. The goal is to preserve a clean
separation between state, rendering, input, and side effects while keeping the
TUI easy to test and safe to evolve.

## Previous TUI Structure

The earlier implementation is concentrated in a small number of command files:

- **`cmd/tmux-intray/tui.go`**: UI model, rendering, input handling, and control
  flow in one file
- **`cmd/tmux-intray/follow.go`**: Follow mode orchestration tied to TUI behavior

## Current TUI Implementation

The project currently exposes a single TUI entry point that mixes concerns:

- `cmd/tmux-intray/tui.go` renders views, handles input, updates state, and
  coordinates follow behavior
- `cmd/tmux-intray/follow.go` integrates with the TUI loop and shares logic with
  the UI state

## Refined TUI Architecture with Interface Contracts

The TUI architecture has been refined with five core interface contracts that
define clear boundaries between components:

```
github.com/cristianoliveira/tmux-intray/
├── cmd/
│   ├── tui.go                  # TUI command entry point
│   └── follow.go               # Follow command entry point
├── internal/
│   └── tui/
│       ├── model/              # Interface contracts defining TUI component boundaries
│       │   ├── ui_state.go     # UIState interface for view state management
│       │   ├── repository.go   # NotificationRepository interface for data access
│       │   ├── tree_service.go # TreeService interface for tree management
│       │   ├── command_service.go # CommandService interface for command handling
│       │   └── runtime_coordinator.go # RuntimeCoordinator interface for tmux integration
│       ├── state/              # UI state model and reducers (implements UIState)
│       ├── render/             # Pure view rendering
│       └── follow/             # Follow orchestration and integration
```

## Interface Contracts

### 1. NotificationRepository Interface

The `NotificationRepository` interface defines the contract for notification data access operations:

```go
type NotificationRepository interface {
    LoadNotifications() ([]notification.Notification, error)
    LoadFilteredNotifications(stateFilter, levelFilter, sessionFilter, windowFilter, paneFilter string) ([]notification.Notification, error)
    DismissNotification(id string) error
    MarkAsRead(id string) error
    MarkAsUnread(id string) error
    GetByID(id string) (notification.Notification, error)
    GetActiveCount() int
}
```

This interface abstracts the storage layer, allowing different backends (TSV, SQLite) to be swapped without affecting TUI logic.

### 2. TreeService Interface

The `TreeService` interface defines operations for building and managing hierarchical notification trees:

```go
type TreeService interface {
    BuildTree(notifications []notification.Notification, groupBy string) (*TreeNode, error)
    FindNotificationPath(notif notification.Notification) ([]*TreeNode, error)
    FindNodeByID(identifier string) *TreeNode
    GetVisibleNodes(root *TreeNode) []*TreeNode
    GetNodeIdentifier(node *TreeNode) string
    PruneEmptyGroups(root *TreeNode) *TreeNode
    ApplyExpansionState(root *TreeNode, expansionState map[string]bool)
    ExpandNode(node *TreeNode)
    CollapseNode(node *TreeNode)
    ToggleNodeExpansion(node *TreeNode)
    GetTreeLevel(node *TreeNode) int
}
```

This interface encapsulates tree operations for grouped views, handling session/window/pane hierarchies.

### 3. UIState Interface

The `UIState` interface manages the interactive state of the TUI:

```go
type UIState interface {
    // Cursor management
    GetCursor() int
    SetCursor(pos int)
    ResetCursor()
    AdjustCursorBounds(listLength int)
    
    // Search mode
    GetSearchMode() bool
    SetSearchMode(enabled bool)
    GetSearchQuery() string
    SetSearchQuery(query string)
    
    // Command mode
    GetCommandMode() bool
    SetCommandMode(enabled bool)
    GetCommandQuery() string
    SetCommandQuery(query string)
    
    // View configuration
    GetViewMode() ViewMode
    SetViewMode(mode ViewMode)
    CycleViewMode()
    GetGroupBy() GroupBy
    SetGroupBy(groupBy GroupBy)
    GetExpandLevel() int
    SetExpandLevel(level int)
    
    // Tree state
    IsGroupedView() bool
    GetExpansionState() map[string]bool
    SetExpansionState(state map[string]bool)
    UpdateExpansionState(nodeIdentifier string, expanded bool)
    
    // Selection helpers
    GetSelectedNotification(notifications []notification.Notification, visibleNodes []*TreeNode) (notification.Notification, bool)
    GetSelectedNode(visibleNodes []*TreeNode) *TreeNode
    
    // Viewport management
    GetViewportDimensions() (width, height int)
    SetViewportDimensions(width, height int)
    GetDimensions() (width, height int)
    SetDimensions(width, height int)
    
    // Persistence
    Save() error
    Load() error
    ToDTO() UIDTO
    FromDTO(dto UIDTO) error
}
```

This interface defines all UI state operations, separating state management from rendering and input handling.

### 4. CommandService Interface

The `CommandService` interface handles command parsing and execution:

```go
type CommandService interface {
    ParseCommand(command string) (name string, args []string, err error)
    ExecuteCommand(name string, args []string) (*CommandResult, error)
    ValidateCommand(name string, args []string) error
    GetAvailableCommands() []CommandInfo
    GetCommandHelp(name string) string
    GetCommandSuggestions(partial string) []string
}
```

This interface enables extensible command handling within the TUI, supporting commands like `:q`, `:w`, `:group-by`.

### 5. RuntimeCoordinator Interface

The `RuntimeCoordinator` interface handles tmux integration and coordination:

```go
type RuntimeCoordinator interface {
    EnsureTmuxRunning() bool
    JumpToPane(sessionID, windowID, paneID string) bool
    ValidatePaneExists(sessionID, windowID, paneID string) (bool, error)
    GetCurrentContext() (*TmuxContext, error)
    ListSessions() (map[string]string, error)
    ListWindows() (map[string]string, error)
    ListPanes() (map[string]string, error)
    GetSessionName(sessionID string) (string, error)
    GetWindowName(windowID string) (string, error)
    GetPaneName(paneID string) (string, error)
    RefreshNames() error
    GetTmuxVisibility() (bool, error)
    SetTmuxVisibility(visible bool) error
}
```

This interface abstracts tmux operations, allowing the TUI to interact with tmux sessions, windows, and panes.

## Package Descriptions

### `internal/tui/model/` (Interface Contracts)

The model package contains the five interface contracts that define the TUI architecture:
- **`ui_state.go`**: Defines the UIState interface for view state management
- **`repository.go`**: Defines the NotificationRepository interface for data access
- **`tree_service.go`**: Defines the TreeService interface for tree operations
- **`command_service.go`**: Defines the CommandService interface for command handling
- **`runtime_coordinator.go`**: Defines the RuntimeCoordinator interface for tmux integration

These interfaces establish clear contracts between TUI components, enabling:
- Testable implementations with mocks
- Swappable implementations (e.g., different storage backends)
- Clear separation of concerns
- Extensible architecture

### `cmd/` (TUI Commands)

The command layer remains a thin wrapper:
- Parse CLI flags and configuration
- Assemble dependencies (implementations of the interfaces)
- Invoke the appropriate TUI entry point

### `internal/tui/state`

State model and reducers (implements UIState):
- Contains the Model struct that implements the UIState interface
- Defines view state and derived values
- Exposes action types for user input and side effects
- Updates state via pure reducer-style functions
- Manages viewport, cursor position, and view configuration

### `internal/tui/render`

Rendering helpers:
- Pure view construction (state in, view out)
- No file, network, or tmux I/O
- Rendering decisions are deterministic and testable
- Supports different view modes (compact, detailed, grouped)

### `internal/tui/follow`

Follow mode orchestration:
- Integrates follow behavior with the TUI via the RuntimeCoordinator interface
- Avoids direct access to internal UI state where possible
- Keeps follow logic isolated from view rendering

## Design Principles

1. **Interface-Driven Development**: Core interfaces define component contracts
2. **Separation of Concerns**: State, rendering, input, and effects live in dedicated packages
3. **Pure Rendering**: Rendering functions are deterministic and side-effect free
4. **Resilience**: Runtime handles cancellations, errors, and cleanup in one place
5. **Testability**: Interface contracts enable comprehensive testing with mocks
6. **Minimal Command Layer**: CLI files are wiring only, not logic

## Implementation Status

### Completed Components

1. **Interface Contracts**: All five interface contracts have been defined in `internal/tui/model/`
2. **UI State Implementation**: The Model struct in `internal/tui/state/model.go` implements UIState
3. **Tree Operations**: Tree building and management functionality in `internal/tui/state/tree.go`
4. **Rendering**: View rendering in `internal/tui/render/render.go`

### Migration Progress

The architecture has evolved from the proposed package structure:

1. ✓ **Phase 1**: Interface contracts defined in `internal/tui/model/`
2. ✓ **Phase 2**: State implementation with Model implementing UIState
3. ✓ **Phase 3**: Rendering separated into `internal/tui/render/`
4. ✓ **Phase 4**: Tree operations implemented
5. ⏳ **Phase 5**: CommandService and RuntimeCoordinator implementations (partial)

## Implementation Notes

- Interface contracts enable dependency injection for testability
- The Model struct implements UIState, maintaining backward compatibility
- Tree operations support grouped views with session/window/pane hierarchies
- Rendering is pure and deterministic, supporting snapshot testing
- Commands and runtime coordination are abstracted through interfaces

## Future Development Guidance

### Adding New Features

1. **New View Modes**: Extend the ViewMode type and add rendering logic in `internal/tui/render/`
2. **New Commands**: Implement handlers via the CommandService interface
3. **Storage Backends**: Implement the NotificationRepository interface
4. **UI State Extensions**: Extend the UIState interface and Model implementation

### Testing Strategy

1. **Interface Testing**: Mock interface contracts for unit testing
2. **State Testing**: Test Model transitions directly
3. **Rendering Testing**: Use snapshot tests for view output
4. **Integration Testing**: Test implementations of interface contracts

### Performance Considerations

1. **Caching**: The visible nodes cache improves rendering performance
2. **Lazy Loading**: Load notifications on demand for large datasets
3. **Batch Operations**: Use bulk operations for storage updates

## Testing

### Testing Philosophy

TUI testing follows a **pure functional approach** focused on testing Model and View
components separately without terminal interaction. The three pillars of bubbletea
testing are:

1. **Model** (state) - The data structure representing UI state
2. **View** (rendering) - Pure function converting state to display output
3. **Update** (message handling) - Pure function converting (model, message) → (model, command)

The golden rule: **Don't Test the Terminal** - test logic, not rendering frames. Focus on
state transitions and view content rather than full program execution.

### Testing Patterns

#### Model Initialization Testing

Verify that models initialize correctly with expected state:

```go
func TestNew(t *testing.T) {
	// Arrange & Act
	m := New("initial state")

	// Assert
	assert.NotNil(t, m)
	assert.Equal(t, "initial state", m.State)
	assert.Empty(t, m.Items)
}
```

#### Message Handling Testing

Test that models update correctly in response to messages:

```go
func TestUpdate_HandlesKeyMsg(t *testing.T) {
	// Arrange
	m := model{count: 5}
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}

	// Act
	newModel, cmd := m.Update(keyMsg)

	// Assert
	assert.Equal(t, 6, newModel.count)
	assert.Nil(t, cmd)
}

func TestUpdate_HandlesTick(t *testing.T) {
	// Arrange
	m := model{timer: time.Now()}
	tickMsg := tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})(time.Now())

	// Act
	newModel, cmd := m.Update(tickMsg)

	// Assert
	assert.Greater(t, newModel.timer, m.timer)
	assert.NotNil(t, cmd)
}
```

#### View Rendering Testing

Verify that view output contains expected content:

```go
func TestView_ContainsExpectedContent(t *testing.T) {
	// Arrange
	m := model{count: 42, items: []string{"item1", "item2"}}

	// Act
	output := m.View()

	// Assert
	assert.Contains(t, output, "42")
	assert.Contains(t, output, "item1")
	assert.Contains(t, output, "item2")
}

func TestView_MatchesGoldenFile(t *testing.T) {
	// Arrange
	m := model{state: "ready", items: []string{"test"}}

	// Act
	output := m.View()

	// Assert
	golden := "testdata/model_view.golden"
	if *updateGoldenFiles {
		t.Logf("updating golden file %s", golden)
		os.WriteFile(golden, []byte(output), 0644)
	}
	expected, _ := os.ReadFile(golden)
	assert.Equal(t, string(expected), output)
}
```

### Mock Strategies

#### Mock Interface Contracts

Test with mocked implementations of interface contracts:

```go
type MockNotificationRepository struct {
	notifications []notification.Notification
}

func (m *MockNotificationRepository) LoadNotifications() ([]notification.Notification, error) {
	return m.notifications, nil
}

func (m *MockNotificationRepository) DismissNotification(id string) error {
	for i, notif := range m.notifications {
		if notif.ID == id {
			m.notifications[i].State = "dismissed"
			break
		}
	}
	return nil
}

// ... implement other interface methods
```

#### Mock Input/Output

Control program input and capture output for testing:

```go
func TestWithCustomInput(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	in := bytes.NewBufferString("test input\n")
	m := model{}

	// Act - create program with custom input/output
	p := tea.NewProgram(m, tea.WithInput(in), tea.WithOutput(&buf))
	_, err := p.Run()

	// Assert
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "test input")
}
```

### Using teatest

The `teatest` library (from `charmbracelet/x/exp/teatest`) provides utilities for
testing TUI models in a controlled environment:

```go
import "github.com/charmbracelet/x/exp/teatest"

func TestModelIntegration(t *testing.T) {
	// Create test model with initial term size
	tm := teatest.NewTestModel(t, model{}, teatest.WithInitialTermSize(80, 24))
	defer tm.Quit()

	// Type input into the model
	tm.Type("hello world")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Wait for expected output
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("expected text"))
	})

	// Verify final state
	finalModel := tm.FinalModel(t)
	require.NotNil(t, finalModel)

	// Compare output
	teatest.RequireEqualOutput(t, []byte("expected\noutput\n"), tm.Output())
}
```

Key `teatest` functions:
- `NewTestModel()` - Creates a test model with optional configuration
- `Type()` - Simulates typing text
- `Send()` - Sends arbitrary messages to the model
- `WaitFor()` - Waits for output matching a predicate
- `FinalModel()` - Retrieves the model's final state
- `RequireEqualOutput()` - Asserts output matches expected

### Best Practices

1. **Separate Model, View, and Update**
   - Keep components pure and independently testable
   - Model is data only; View is pure function; Update is pure function
   - Avoid side effects in Model and View

2. **Avoid Testing the Full Program**
   - Test components, not the running TUI
   - Use `teatest` for integration-style tests when needed
   - Prefer unit tests for business logic

3. **Use Test Doubles for External Dependencies**
   - Mock interface contracts (NotificationRepository, TreeService, etc.)
   - Use interfaces to enable mocking
   - Keep test doubles simple and focused

4. **Test State Transitions**
   - Verify model updates with different messages
   - Test edge cases (empty state, nil messages, etc.)
   - Use table-driven tests for comprehensive coverage

5. **Test View Output**
   - Verify rendering contains expected content
   - Use golden files for snapshot testing
   - Test responsive layout (different term sizes)

6. **Use Golden Files**
   - Capture and compare full output for integration tests
   - Support `UPDATE_GOLDEN` flag for updating snapshots
   - Store golden files in `testdata/` directory

7. **Test Keybindings**
   - Verify input mapping triggers correct actions
   - Test both key press and key release events
   - Validate focus management (if applicable)

### Anti-Patterns to Avoid

1. **Don't Test the Terminal**
   - Avoid tests that require a real terminal
   - Don't test raw terminal output or escape sequences
   - Focus on model logic and view content

2. **Don't Use time.Sleep Excessively**
   - Use synchronization primitives (channels, WaitFor)
   - Avoid flaky tests caused by timing issues
   - Mock time-dependent operations when possible

3. **Don't Test Animation Frames**
   - Test logic, not rendering
   - Don't verify exact frame timing or tick rates
   - Test state changes caused by animations, not animations themselves

4. **Don't Ignore Cleanup**
   - Always verify resources released (goroutines, files, etc.)
   - Use `defer` for cleanup
   - Test error scenarios and ensure proper shutdown

5. **Don't Mix Concerns**
   - Keep tests focused on single behaviors
   - Avoid testing multiple features in one test
   - Use descriptive test names and subtests

6. **Don't Test Implementation Details**
   - Test observable behavior, not internal structure
   - Avoid testing private helper functions
   - Focus on public APIs and user-facing behavior

### References/Resources

- **charmbracelet/bubbletea test files**: Model/Update/View testing examples
- **charmbracelet/bubbles component tests**: Reusable component test patterns
- **charmbracelet/x/exp/teatest**: Official TUI testing library
- **idursun/jjui**: Production examples of TUI testing patterns
- **Bubbletea Documentation**: https://github.com/charmbracelet/bubbletea
- **Teatest Documentation**: https://github.com/charmbracelet/x/tree/main/exp/teatest

## References

- `cmd/tmux-intray/tui.go`
- `cmd/tmux-intray/follow.go`
- `internal/tui/model/*.go` - Interface contracts
- `internal/tui/state/model.go` - UIState implementation
- `internal/tui/render/render.go` - Rendering logic
- `internal/tui/state/tree.go` - Tree operations