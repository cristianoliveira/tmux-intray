package service

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/errors"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockModelInterface is a mock implementation of ModelInterface for testing.
type MockModelInterface struct {
	mock.Mock
}

func (m *MockModelInterface) SaveSettings() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockModelInterface) GetGroupBy() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockModelInterface) SetGroupBy(groupBy string) error {
	args := m.Called(groupBy)
	return args.Error(0)
}

func (m *MockModelInterface) ApplySearchFilter() {
	m.Called()
}

func (m *MockModelInterface) ResetCursor() {
	m.Called()
}

func (m *MockModelInterface) GetReadFilter() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockModelInterface) SetReadFilter(value string) error {
	args := m.Called(value)
	return args.Error(0)
}

func (m *MockModelInterface) GetExpandLevel() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockModelInterface) SetExpandLevel(level int) error {
	args := m.Called(level)
	return args.Error(0)
}

func (m *MockModelInterface) IsGroupedView() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockModelInterface) ApplyDefaultExpansion() {
	m.Called()
}

func (m *MockModelInterface) ToggleViewMode() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockModelInterface) GetViewMode() string {
	args := m.Called()
	return args.String(0)
}

func TestNewCommandService(t *testing.T) {
	mockModel := new(MockModelInterface)
	errorHandler := errors.NewTUIHandler(nil)
	service := NewCommandService(mockModel, errorHandler)

	assert.NotNil(t, service)
	assert.Equal(t, mockModel, service.(*DefaultCommandService).model)
	assert.Equal(t, errorHandler, service.(*DefaultCommandService).errorHandler)
}

func TestParseCommand(t *testing.T) {
	mockModel := new(MockModelInterface)
	service := NewCommandService(mockModel, errors.NewTUIHandler(nil))

	tests := []struct {
		name         string
		command      string
		expectedCmd  string
		expectedArgs []string
		expectError  bool
	}{
		{
			name:         "empty command",
			command:      "",
			expectedCmd:  "",
			expectedArgs: nil,
			expectError:  true,
		},
		{
			name:         "command only",
			command:      "q",
			expectedCmd:  "q",
			expectedArgs: []string{},
			expectError:  false,
		},
		{
			name:         "command with args",
			command:      "group-by session",
			expectedCmd:  "group-by",
			expectedArgs: []string{"session"},
			expectError:  false,
		},
		{
			name:         "command with extra spaces",
			command:      "  expand-level   2  ",
			expectedCmd:  "expand-level",
			expectedArgs: []string{"2"},
			expectError:  false,
		},
		{
			name:         "uppercase command",
			command:      "GROUP-BY SESSION",
			expectedCmd:  "group-by",
			expectedArgs: []string{"SESSION"},
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, args, err := service.ParseCommand(tt.command)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCmd, cmd)
				assert.Equal(t, tt.expectedArgs, args)
			}
		})
	}
}

func TestGetAvailableCommands(t *testing.T) {
	mockModel := new(MockModelInterface)
	service := NewCommandService(mockModel, errors.NewTUIHandler(nil))

	commands := service.GetAvailableCommands()

	// Should have 6 default commands
	assert.Len(t, commands, 6)

	// Check for expected commands
	cmdNames := make([]string, len(commands))
	for i, cmd := range commands {
		cmdNames[i] = cmd.Name
	}
	assert.Contains(t, cmdNames, "q")
	assert.Contains(t, cmdNames, "w")
	assert.Contains(t, cmdNames, "group-by")
	assert.Contains(t, cmdNames, "expand-level")
	assert.Contains(t, cmdNames, "toggle-view")
	assert.Contains(t, cmdNames, "filter-read")
}

func TestGetCommandHelp(t *testing.T) {
	mockModel := new(MockModelInterface)
	service := NewCommandService(mockModel, errors.NewTUIHandler(nil))

	// Test valid command
	help := service.GetCommandHelp("q")
	assert.NotEmpty(t, help)
	assert.Contains(t, help, "q")
	assert.Contains(t, help, "Quit")

	// Test invalid command
	help = service.GetCommandHelp("filter-read")
	assert.NotEmpty(t, help)
	assert.Contains(t, help, "filter-read")
	assert.Contains(t, help, "read")

	// Test invalid command
	help = service.GetCommandHelp("nonexistent")
	assert.Empty(t, help)
}

func TestGetCommandSuggestions(t *testing.T) {
	mockModel := new(MockModelInterface)
	service := NewCommandService(mockModel, errors.NewTUIHandler(nil))

	// Test partial suggestions
	suggestions := service.GetCommandSuggestions("g")
	assert.Contains(t, suggestions, "group-by")

	// Test full match
	suggestions = service.GetCommandSuggestions("q")
	assert.Contains(t, suggestions, "q")

	// Test filter-read suggestion
	suggestions = service.GetCommandSuggestions("f")
	assert.Contains(t, suggestions, "filter-read")

	// Test no match
	suggestions = service.GetCommandSuggestions("nonexistent")
	assert.Empty(t, suggestions)
}

func TestValidateCommand(t *testing.T) {
	mockModel := new(MockModelInterface)
	service := NewCommandService(mockModel, errors.NewTUIHandler(nil))

	// Test valid command
	err := service.ValidateCommand("q", []string{})
	assert.NoError(t, err)

	// Test invalid command name
	err = service.ValidateCommand("nonexistent", []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")

	// Test valid command with invalid args
	err = service.ValidateCommand("group-by", []string{"too", "many", "args"})
	assert.Error(t, err)
}

func TestExecuteCommand_Quit(t *testing.T) {
	mockModel := new(MockModelInterface)
	mockModel.On("SaveSettings").Return(nil)
	service := NewCommandService(mockModel, errors.NewTUIHandler(nil))

	result, err := service.ExecuteCommand("q", []string{})

	assert.NoError(t, err)
	assert.True(t, result.Quit)
	assert.Nil(t, result.Cmd)
	mockModel.AssertExpectations(t)
}

func TestExecuteCommand_QuitWithError(t *testing.T) {
	mockModel := new(MockModelInterface)
	mockModel.On("SaveSettings").Return(assert.AnError)
	service := NewCommandService(mockModel, errors.NewTUIHandler(nil))

	result, err := service.ExecuteCommand("q", []string{})

	assert.NoError(t, err)
	assert.True(t, result.Quit) // Should still quit even if save fails
	assert.Nil(t, result.Cmd)
	mockModel.AssertExpectations(t)
}

func TestExecuteCommand_InvalidCommand(t *testing.T) {
	mockModel := new(MockModelInterface)
	service := NewCommandService(mockModel, errors.NewTUIHandler(nil))

	result, err := service.ExecuteCommand("nonexistent", []string{})

	assert.NoError(t, err)
	assert.True(t, result.Error)
	assert.Contains(t, result.Message, "unknown command")
	assert.Nil(t, result.Cmd)
}

func TestExecuteCommand_InvalidArgs(t *testing.T) {
	mockModel := new(MockModelInterface)
	service := NewCommandService(mockModel, errors.NewTUIHandler(nil))

	result, err := service.ExecuteCommand("q", []string{"invalid"})

	assert.NoError(t, err)
	assert.True(t, result.Error)
	assert.Contains(t, result.Message, "invalid usage")
	assert.Nil(t, result.Cmd)
}

func TestExecuteCommand_GroupBy(t *testing.T) {
	mockModel := new(MockModelInterface)
	mockModel.On("GetGroupBy").Return("none")
	mockModel.On("SetGroupBy", "session").Return(nil)
	mockModel.On("ApplySearchFilter")
	mockModel.On("ResetCursor")
	mockModel.On("SaveSettings").Return(nil)
	service := NewCommandService(mockModel, errors.NewTUIHandler(nil))

	result, err := service.ExecuteCommand("group-by", []string{"session"})

	assert.NoError(t, err)
	assert.False(t, result.Error)
	assert.Contains(t, result.Message, "Group by: session")
	assert.Nil(t, result.Cmd)
	mockModel.AssertExpectations(t)
}

func TestExecuteCommand_ExpandLevel(t *testing.T) {
	mockModel := new(MockModelInterface)
	mockModel.On("GetExpandLevel").Return(0)
	mockModel.On("SetExpandLevel", 2).Return(nil)
	mockModel.On("IsGroupedView").Return(false) // Not grouped view
	mockModel.On("SaveSettings").Return(nil)
	service := NewCommandService(mockModel, errors.NewTUIHandler(nil))

	result, err := service.ExecuteCommand("expand-level", []string{"2"})

	assert.NoError(t, err)
	assert.False(t, result.Error)
	assert.Contains(t, result.Message, "Default expand level: 2")
	assert.Nil(t, result.Cmd)
	mockModel.AssertExpectations(t)
}

func TestExecuteCommand_ToggleView(t *testing.T) {
	mockModel := new(MockModelInterface)
	mockModel.On("ToggleViewMode").Return(nil)
	mockModel.On("ApplySearchFilter")
	mockModel.On("ResetCursor")
	mockModel.On("SaveSettings").Return(nil)
	mockModel.On("GetViewMode").Return("grouped")
	service := NewCommandService(mockModel, errors.NewTUIHandler(nil))

	result, err := service.ExecuteCommand("toggle-view", []string{})

	assert.NoError(t, err)
	assert.False(t, result.Error)
	assert.Contains(t, result.Message, "View mode: grouped")
	assert.Nil(t, result.Cmd)
	mockModel.AssertExpectations(t)
}

func TestExecuteCommand_FilterRead(t *testing.T) {
	mockModel := new(MockModelInterface)
	mockModel.On("GetReadFilter").Return("")
	mockModel.On("SetReadFilter", settings.ReadFilterUnread).Return(nil)
	mockModel.On("ApplySearchFilter")
	mockModel.On("ResetCursor")
	mockModel.On("SaveSettings").Return(nil)
	service := NewCommandService(mockModel, errors.NewTUIHandler(nil))

	result, err := service.ExecuteCommand("filter-read", []string{"unread"})

	assert.NoError(t, err)
	assert.False(t, result.Error)
	assert.Contains(t, result.Message, "Read filter: unread")
	assert.Nil(t, result.Cmd)
	mockModel.AssertExpectations(t)
}

func TestExecuteCommand_FilterReadAlreadySet(t *testing.T) {
	mockModel := new(MockModelInterface)
	mockModel.On("GetReadFilter").Return(settings.ReadFilterRead)
	service := NewCommandService(mockModel, errors.NewTUIHandler(nil))

	result, err := service.ExecuteCommand("filter-read", []string{"read"})
	assert.NoError(t, err)
	assert.False(t, result.Error)
	assert.Contains(t, result.Message, "already set")
	assert.Nil(t, result.Cmd)
	mockModel.AssertExpectations(t)
}

func TestExecuteCommand_Write(t *testing.T) {
	mockModel := new(MockModelInterface)
	service := NewCommandService(mockModel, errors.NewTUIHandler(nil))

	result, err := service.ExecuteCommand("w", []string{})

	assert.NoError(t, err)
	assert.False(t, result.Error)
	assert.NotNil(t, result.Cmd)
}

// Test individual command handlers

func TestQuitCommandHandler(t *testing.T) {
	mockModel := new(MockModelInterface)
	handler := &QuitCommandHandler{model: mockModel}

	// Test Execute
	mockModel.On("SaveSettings").Return(nil)
	result, err := handler.Execute([]string{})
	assert.NoError(t, err)
	assert.True(t, result.Quit)

	// Test Validate
	err = handler.Validate([]string{})
	assert.NoError(t, err)

	err = handler.Validate([]string{"invalid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid usage")

	// Test Complete
	suggestions := handler.Complete([]string{})
	assert.Empty(t, suggestions)
}

func TestGroupByCommandHandler(t *testing.T) {
	mockModel := new(MockModelInterface)
	handler := &GroupByCommandHandler{model: mockModel}

	// Test Execute
	mockModel.On("GetGroupBy").Return("none")
	mockModel.On("SetGroupBy", "session").Return(nil)
	mockModel.On("ApplySearchFilter")
	mockModel.On("ResetCursor")
	mockModel.On("SaveSettings").Return(nil)
	result, err := handler.Execute([]string{"session"})
	assert.NoError(t, err)
	assert.Contains(t, result.Message, "Group by: session")

	// Test Validate
	err = handler.Validate([]string{"session"})
	assert.NoError(t, err)

	err = handler.Validate([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid usage")

	err = handler.Validate([]string{"invalid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid group-by value")

	// Test Complete
	suggestions := handler.Complete([]string{})
	assert.Contains(t, suggestions, "none")
	assert.Contains(t, suggestions, "session")
	assert.Contains(t, suggestions, "window")
	assert.Contains(t, suggestions, "pane")
	assert.Contains(t, suggestions, "message")
}

func TestExpandLevelCommandHandler(t *testing.T) {
	mockModel := new(MockModelInterface)
	handler := &ExpandLevelCommandHandler{model: mockModel}

	// Test Execute
	mockModel.On("GetExpandLevel").Return(0)
	mockModel.On("SetExpandLevel", 2).Return(nil)
	mockModel.On("IsGroupedView").Return(false)
	mockModel.On("SaveSettings").Return(nil)
	result, err := handler.Execute([]string{"2"})
	assert.NoError(t, err)
	assert.Contains(t, result.Message, "Default expand level: 2")

	// Test Validate
	err = handler.Validate([]string{"2"})
	assert.NoError(t, err)

	err = handler.Validate([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid usage")

	err = handler.Validate([]string{"invalid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid expand-level value")

	// Test Complete
	suggestions := handler.Complete([]string{})
	assert.Contains(t, suggestions, "0")
	assert.Contains(t, suggestions, "1")
	assert.Contains(t, suggestions, "2")
	assert.Contains(t, suggestions, "3")
}
