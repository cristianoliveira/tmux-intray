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

## Proposed TUI Package Structure

```
github.com/cristianoliveira/tmux-intray/
├── cmd/
│   ├── tui.go                  # TUI command entry point
│   └── follow.go               # Follow command entry point
├── internal/
│   └── tui/
│       ├── state/              # UI state model and reducers
│       ├── render/             # Pure view rendering
│       ├── input/              # Keybindings and event mapping
│       ├── runtime/            # TUI loop, lifecycle, error handling
│       └── follow/             # Follow orchestration and integration
```

## Package Descriptions

### `cmd/` (TUI Commands)

The command layer should remain a thin wrapper:
- Parse CLI flags and configuration
- Assemble dependencies
- Invoke `internal/tui/runtime` or `internal/tui/follow` entry points

### `internal/tui/state`

State model and reducers:
- Defines view state and derived values
- Exposes action types for user input and side effects
- Updates state via pure reducer-style functions

### `internal/tui/render`

Rendering helpers:
- Pure view construction (state in, view out)
- No file, network, or tmux I/O
- Rendering decisions are deterministic and testable

### `internal/tui/input`

Input handling:
- Defines keybindings and event mapping
- Translates terminal events into state actions
- Avoids direct state mutation

### `internal/tui/runtime`

Lifecycle wiring:
- Runs the event loop and orchestrates state, input, and rendering
- Owns cancellation, shutdown, and recovery behavior
- Centralizes error handling and user-facing failure states

### `internal/tui/follow`

Follow mode orchestration:
- Integrates follow behavior with the runtime via a narrow interface
- Avoids direct access to internal UI state where possible
- Keeps follow logic isolated from view rendering

### Command Implementation in `cmd/`

The command files should be small and declarative:
- Each command defines flags and invokes a single runtime entry point
- Business logic remains inside `internal/tui/*` packages
- Errors are surfaced through consistent, user-facing messages

## Design Principles

1. **Separation of Concerns**: State, rendering, input, and effects live in
   dedicated packages.
2. **Pure Rendering**: Rendering functions are deterministic and side-effect
   free.
3. **Resilience**: Runtime handles cancellations, errors, and cleanup in one
   place.
4. **Testability**: State transitions and render output are directly testable.
5. **Minimal Command Layer**: CLI files are wiring only, not logic.

## Migration Strategy

1. **Phase 1**: Extract state and reducers into `internal/tui/state`.
2. **Phase 2**: Move rendering helpers into `internal/tui/render`.
3. **Phase 3**: Isolate input mapping into `internal/tui/input`.
4. **Phase 4**: Introduce `internal/tui/runtime` and rewire the TUI loop.
5. **Phase 5**: Move follow orchestration into `internal/tui/follow`.

## Implementation Notes

- Prefer explicit error propagation and surface failures via colors.Error.
- Keep runtime shutdown idempotent; always release resources on exit.
- Use context cancellation for long-running operations and follow mode.
- Avoid direct tmux calls from render/input packages.

## Next Steps

1. Define the `state` model and action types
2. Extract render helpers and add snapshot-style tests
3. Map keybindings in `input` and validate action dispatch
4. Build a runtime wrapper with centralized error handling
5. Move follow-mode orchestration behind a narrow interface

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

#### Mock External Commands

Replace command execution with predictable behavior:

```go
type MockCommandRunner struct {
	expectations map[string]MockCommand
	calls        []string
	t            *testing.T
}

type MockCommand struct {
	Output string
	Error  error
}

func (m *MockCommandRunner) Run(cmd string, args ...string) (string, error) {
	key := strings.Join(append([]string{cmd}, args...), " ")
	m.calls = append(m.calls, key)
	mock, ok := m.expectations[key]
	if !ok {
		m.t.Fatalf("unexpected command: %s", key)
	}
	return mock.Output, mock.Error
}

func TestWithMockedCommand(t *testing.T) {
	runner := &MockCommandRunner{
		expectations: map[string]MockCommand{
			"tmux display-message -p '#{pane_id}'": {Output: "%1"},
		},
		t: t,
	}
	m := model{runner: runner}
	// Test model behavior with mocked runner
}
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
   - Mock command execution, time, and I/O
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
