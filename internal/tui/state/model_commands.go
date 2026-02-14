package state

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// executeCommandViaService executes the current command query using the CommandService and returns a command to run.
func (m *Model) executeCommandViaService() tea.Cmd {
	// If commandService is not initialized, fall back to legacy implementation
	if m.commandService == nil {
		return m.executeCommand()
	}

	cmd := strings.TrimSpace(m.uiState.GetCommandQuery())
	if cmd == "" {
		m.errorHandler.Warning("Command is empty")
		return errorMsgAfter(errorClearDuration)
	}

	// Parse and execute command using CommandService
	name, args, err := m.commandService.ParseCommand(cmd)
	if err != nil {
		m.errorHandler.Warning(fmt.Sprintf("Failed to parse command: %v", err))
		return errorMsgAfter(errorClearDuration)
	}

	result, err := m.commandService.ExecuteCommand(name, args)
	if err != nil {
		m.errorHandler.Error(fmt.Sprintf("Failed to execute command: %v", err))
		return errorMsgAfter(errorClearDuration)
	}

	// Handle result
	if result.Message != "" {
		if result.Error {
			m.errorHandler.Warning(result.Message)
		} else {
			m.errorHandler.Info(result.Message)
		}
	}

	if result.Quit {
		return tea.Quit
	}

	return result.Cmd
}

// executeCommand executes the current command query and returns a command to run.
// This is the legacy implementation kept for reference.
func (m *Model) executeCommand() tea.Cmd {
	cmd := strings.TrimSpace(m.uiState.GetCommandQuery())
	if cmd == "" {
		m.errorHandler.Warning("Command is empty")
		return errorMsgAfter(errorClearDuration)
	}

	parts := strings.Fields(cmd)
	command := strings.ToLower(parts[0])
	args := parts[1:]

	switch command {
	case "q":
		return m.handleQuitCommand(args)
	case "w":
		return m.handleWriteCommand(args)
	case "group-by":
		return m.handleGroupByCommand(args)
	case "expand-level":
		return m.handleExpandLevelCommand(args)

	case "toggle-view":
		return m.handleToggleViewCommand(args)
	default:
		m.errorHandler.Warning(fmt.Sprintf("Unknown command: %s", command))
		return errorMsgAfter(errorClearDuration)
	}
}

func (m *Model) handleQuitCommand(args []string) tea.Cmd {
	if len(args) > 0 {
		m.errorHandler.Warning("Invalid usage: q")
		return errorMsgAfter(errorClearDuration)
	}
	if err := m.saveSettings(); err != nil {
		m.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
	}
	return tea.Quit
}

func (m *Model) handleWriteCommand(args []string) tea.Cmd {
	if len(args) > 0 {
		m.errorHandler.Warning("Invalid usage: w")
		return errorMsgAfter(errorClearDuration)
	}
	return func() tea.Msg {
		if err := m.saveSettings(); err != nil {
			return saveSettingsFailedMsg{err: err}
		}
		return saveSettingsSuccessMsg{}
	}
}

func (m *Model) handleGroupByCommand(args []string) tea.Cmd {
	if len(args) != 1 {
		m.errorHandler.Warning("Invalid usage: group-by <none|session|window|pane>")
		return errorMsgAfter(errorClearDuration)
	}

	groupBy := strings.ToLower(args[0])
	if !settings.IsValidGroupBy(groupBy) {
		m.errorHandler.Warning(fmt.Sprintf("Invalid group-by value: %s (expected one of: none, session, window, pane)", args[0]))
		return errorMsgAfter(errorClearDuration)
	}

	if string(m.uiState.GetGroupBy()) == groupBy {
		return nil
	}

	m.uiState.SetGroupBy(model.GroupBy(groupBy))
	m.applySearchFilter()
	m.resetCursor()
	if err := m.saveSettings(); err != nil {
		m.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
		return errorMsgAfter(errorClearDuration)
	}
	m.errorHandler.Info(fmt.Sprintf("Group by: %s", groupBy))
	return nil
}

func (m *Model) handleExpandLevelCommand(args []string) tea.Cmd {
	if len(args) != 1 {
		m.errorHandler.Warning("Invalid usage: expand-level <0|1|2|3>")
		return errorMsgAfter(errorClearDuration)
	}

	level, err := strconv.Atoi(args[0])
	if err != nil || level < settings.MinExpandLevel || level > settings.MaxExpandLevel {
		m.errorHandler.Warning(fmt.Sprintf("Invalid expand-level value: %s (expected %d-%d)", args[0], settings.MinExpandLevel, settings.MaxExpandLevel))
		return errorMsgAfter(errorClearDuration)
	}

	if m.uiState.GetExpandLevel() == level {
		return nil
	}

	m.uiState.SetExpandLevel(level)
	if m.isGroupedView() {
		m.applyDefaultExpansion()
	}
	if err := m.saveSettings(); err != nil {
		m.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
		return errorMsgAfter(errorClearDuration)
	}
	m.errorHandler.Info(fmt.Sprintf("Default expand level: %d", m.uiState.GetExpandLevel()))
	return nil
}

func (m *Model) handleToggleViewCommand(args []string) tea.Cmd {
	if len(args) > 0 {
		m.errorHandler.Warning("Invalid usage: toggle-view")
		return errorMsgAfter(errorClearDuration)
	}

	if m.uiState.IsGroupedView() {
		m.uiState.SetViewMode(model.ViewModeDetailed)
	} else {
		m.uiState.SetViewMode(model.ViewModeGrouped)
	}
	m.applySearchFilter()
	m.resetCursor()
	if err := m.saveSettings(); err != nil {
		m.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
	}
	m.errorHandler.Info(fmt.Sprintf("View mode: %s", m.uiState.GetViewMode()))
	return nil
}
