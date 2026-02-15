package service

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cristianoliveira/tmux-intray/internal/errors"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// saveSettingsSuccessMsg is sent when settings are saved successfully.
type saveSettingsSuccessMsg struct{}

// saveSettingsFailedMsg is sent when settings save fails.
type saveSettingsFailedMsg struct {
	err error
}

// DefaultCommandService implements the CommandService interface.
type DefaultCommandService struct {
	model        ModelInterface // Reference to the TUI model for state access
	errorHandler errors.ErrorHandler
	handlers     map[string]model.CommandHandler

	help model.HelpProvider
}

// NewCommandService creates a new DefaultCommandService.
func NewCommandService(modelInterface ModelInterface, errorHandler errors.ErrorHandler) model.CommandService {
	service := &DefaultCommandService{
		model:        modelInterface,
		errorHandler: errorHandler,
		handlers:     make(map[string]model.CommandHandler),

		help: NewDefaultHelpProvider(),
	}

	// Register default command handlers
	service.registerDefaultHandlers()

	return service
}

// ParseCommand parses a command string into its constituent parts.
func (s *DefaultCommandService) ParseCommand(command string) (name string, args []string, err error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", nil, fmt.Errorf("command is empty")
	}

	parts := strings.Fields(command)
	name = strings.ToLower(parts[0])
	args = parts[1:]

	return name, args, nil
}

// ExecuteCommand executes a parsed command and returns the result.
func (s *DefaultCommandService) ExecuteCommand(name string, args []string) (*model.CommandResult, error) {
	// Validate command first
	if err := s.ValidateCommand(name, args); err != nil {
		s.errorHandler.Warning(fmt.Sprintf("Command validation failed: %v", err))
		return &model.CommandResult{Error: true, Message: err.Error()}, nil
	}

	// Get handler
	handler, ok := s.handlers[name]
	if !ok {
		msg := fmt.Sprintf("Unknown command: %s", name)
		s.errorHandler.Warning(msg)
		return &model.CommandResult{Error: true, Message: msg}, nil
	}

	// Execute command
	result, err := handler.Execute(args)
	if err != nil {
		s.errorHandler.Error(fmt.Sprintf("Command execution failed: %v", err))
		return &model.CommandResult{Error: true, Message: err.Error()}, nil
	}

	return result, nil
}

// ValidateCommand checks if a command and its arguments are valid.
func (s *DefaultCommandService) ValidateCommand(name string, args []string) error {
	handler, ok := s.handlers[name]
	if !ok {
		return fmt.Errorf("unknown command: %s", name)
	}

	return handler.Validate(args)
}

// GetAvailableCommands returns a list of all available commands.
func (s *DefaultCommandService) GetAvailableCommands() []model.CommandInfo {
	var commands []model.CommandInfo

	// Collect commands from handlers
	for name := range s.handlers {
		// Get help info for this command
		help := s.help.GetHelp(name)
		if help != "" {
			// Parse help to extract description and usage
			commands = append(commands, model.CommandInfo{
				Name:        name,
				Description: help, // Simplified - in real implementation, parse structured help
				Usage:       help,
			})
		} else {
			commands = append(commands, model.CommandInfo{
				Name: name,
			})
		}
	}

	return commands
}

// GetCommandHelp returns help text for a specific command.
func (s *DefaultCommandService) GetCommandHelp(name string) string {
	return s.help.GetHelp(name)
}

// GetCommandSuggestions returns suggestions for command completion.
func (s *DefaultCommandService) GetCommandSuggestions(partial string) []string {
	var suggestions []string

	// Check for command name suggestions
	for name := range s.handlers {
		if strings.HasPrefix(name, partial) {
			suggestions = append(suggestions, name)
		}
	}

	return suggestions
}

// registerDefaultHandlers registers the default command handlers.
func (s *DefaultCommandService) registerDefaultHandlers() {
	// Quit command
	s.handlers["q"] = &QuitCommandHandler{model: s.model, errorHandler: s.errorHandler}

	// Write/save settings command
	s.handlers["w"] = &WriteCommandHandler{model: s.model, errorHandler: s.errorHandler}

	// Group by command
	s.handlers["group-by"] = &GroupByCommandHandler{model: s.model, errorHandler: s.errorHandler}

	// Expand level command
	s.handlers["expand-level"] = &ExpandLevelCommandHandler{model: s.model, errorHandler: s.errorHandler}

	// Toggle view command
	s.handlers["toggle-view"] = &ToggleViewCommandHandler{model: s.model, errorHandler: s.errorHandler}

	// Read filter command
	s.handlers["filter-read"] = &FilterReadCommandHandler{model: s.model, errorHandler: s.errorHandler}
}

// QuitCommandHandler handles the quit command.
type QuitCommandHandler struct {
	model        ModelInterface
	errorHandler errors.ErrorHandler
}

func (h *QuitCommandHandler) Execute(args []string) (*model.CommandResult, error) {
	if len(args) > 0 {
		return &model.CommandResult{Error: true, Message: "Invalid usage: q"}, nil
	}

	if err := h.model.SaveSettings(); err != nil {
		h.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
		// Still proceed with quit
	}

	return &model.CommandResult{Quit: true}, nil
}

func (h *QuitCommandHandler) Validate(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("invalid usage: q")
	}
	return nil
}

func (h *QuitCommandHandler) Complete(args []string) []string {
	return []string{} // No completion for quit
}

// WriteCommandHandler handles the write/save settings command.
type WriteCommandHandler struct {
	model        ModelInterface
	errorHandler errors.ErrorHandler
}

func (h *WriteCommandHandler) Execute(args []string) (*model.CommandResult, error) {
	if len(args) > 0 {
		return &model.CommandResult{Error: true, Message: "Invalid usage: w"}, nil
	}

	return &model.CommandResult{
		Cmd: func() tea.Msg {
			if err := h.model.SaveSettings(); err != nil {
				h.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
				return saveSettingsFailedMsg{err: err}
			}
			return saveSettingsSuccessMsg{}
		},
	}, nil
}

func (h *WriteCommandHandler) Validate(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("invalid usage: w")
	}
	return nil
}

func (h *WriteCommandHandler) Complete(args []string) []string {
	return []string{} // No completion for write
}

// GroupByCommandHandler handles the group-by command.
type GroupByCommandHandler struct {
	model        ModelInterface
	errorHandler errors.ErrorHandler
}

func (h *GroupByCommandHandler) Execute(args []string) (*model.CommandResult, error) {
	if len(args) != 1 {
		return &model.CommandResult{Error: true, Message: "Invalid usage: group-by <none|session|window|pane|message>"}, nil
	}

	groupBy := strings.ToLower(args[0])
	if !settings.IsValidGroupBy(groupBy) {
		msg := fmt.Sprintf("Invalid group-by value: %s (expected one of: none, session, window, pane, message)", args[0])
		return &model.CommandResult{Error: true, Message: msg}, nil
	}

	// Check if the group-by value is already set
	if h.model.GetGroupBy() == groupBy {
		return &model.CommandResult{Message: fmt.Sprintf("Group by: %s (already set)", groupBy)}, nil
	}

	// Set the group-by value
	if err := h.model.SetGroupBy(groupBy); err != nil {
		return &model.CommandResult{Error: true, Message: fmt.Sprintf("Failed to set group-by: %v", err)}, nil
	}

	h.model.ApplySearchFilter()
	h.model.ResetCursor()

	if err := h.model.SaveSettings(); err != nil {
		h.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
	}

	return &model.CommandResult{Message: fmt.Sprintf("Group by: %s", groupBy)}, nil
}

func (h *GroupByCommandHandler) Validate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("invalid usage: group-by <none|session|window|pane|message>")
	}

	groupBy := strings.ToLower(args[0])
	if !settings.IsValidGroupBy(groupBy) {
		return fmt.Errorf("invalid group-by value: %s (expected one of: none, session, window, pane, message)", args[0])
	}

	return nil
}

func (h *GroupByCommandHandler) Complete(args []string) []string {
	if len(args) == 0 {
		return []string{"none", "session", "window", "pane", "message"}
	}
	return []string{}
}

// ExpandLevelCommandHandler handles the expand-level command.
type ExpandLevelCommandHandler struct {
	model        ModelInterface
	errorHandler errors.ErrorHandler
}

func (h *ExpandLevelCommandHandler) Execute(args []string) (*model.CommandResult, error) {
	if len(args) != 1 {
		return &model.CommandResult{Error: true, Message: "Invalid usage: expand-level <0|1|2|3>"}, nil
	}

	level, err := strconv.Atoi(args[0])
	if err != nil || level < settings.MinExpandLevel || level > settings.MaxExpandLevel {
		msg := fmt.Sprintf("Invalid expand-level value: %s (expected %d-%d)", args[0], settings.MinExpandLevel, settings.MaxExpandLevel)
		return &model.CommandResult{Error: true, Message: msg}, nil
	}

	// Check if the expand level is already set
	if h.model.GetExpandLevel() == level {
		return &model.CommandResult{Message: fmt.Sprintf("Default expand level: %d (already set)", level)}, nil
	}

	// Set the expand level
	if err := h.model.SetExpandLevel(level); err != nil {
		return &model.CommandResult{Error: true, Message: fmt.Sprintf("Failed to set expand-level: %v", err)}, nil
	}

	if h.model.IsGroupedView() {
		h.model.ApplyDefaultExpansion()
	}

	if err := h.model.SaveSettings(); err != nil {
		h.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
	}

	return &model.CommandResult{Message: fmt.Sprintf("Default expand level: %d", level)}, nil
}

func (h *ExpandLevelCommandHandler) Validate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("invalid usage: expand-level <0|1|2|3>")
	}

	level, err := strconv.Atoi(args[0])
	if err != nil || level < settings.MinExpandLevel || level > settings.MaxExpandLevel {
		return fmt.Errorf("invalid expand-level value: %s (expected %d-%d)", args[0], settings.MinExpandLevel, settings.MaxExpandLevel)
	}

	return nil
}

func (h *ExpandLevelCommandHandler) Complete(args []string) []string {
	if len(args) == 0 {
		var suggestions []string
		for i := settings.MinExpandLevel; i <= settings.MaxExpandLevel; i++ {
			suggestions = append(suggestions, strconv.Itoa(i))
		}
		return suggestions
	}
	return []string{}
}

// ToggleViewCommandHandler handles the toggle-view command.
type ToggleViewCommandHandler struct {
	model        ModelInterface
	errorHandler errors.ErrorHandler
}

func (h *ToggleViewCommandHandler) Execute(args []string) (*model.CommandResult, error) {
	if len(args) > 0 {
		return &model.CommandResult{Error: true, Message: "Invalid usage: toggle-view"}, nil
	}

	// Toggle view mode
	if err := h.model.ToggleViewMode(); err != nil {
		return &model.CommandResult{Error: true, Message: fmt.Sprintf("Failed to toggle view: %v", err)}, nil
	}

	h.model.ApplySearchFilter()
	h.model.ResetCursor()

	if err := h.model.SaveSettings(); err != nil {
		h.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
	}

	return &model.CommandResult{Message: fmt.Sprintf("View mode: %s", h.model.GetViewMode())}, nil
}

func (h *ToggleViewCommandHandler) Validate(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("invalid usage: toggle-view")
	}
	return nil
}

func (h *ToggleViewCommandHandler) Complete(args []string) []string {
	return []string{} // No completion for toggle-view
}

// FilterReadCommandHandler handles the filter-read command.
type FilterReadCommandHandler struct {
	model        ModelInterface
	errorHandler errors.ErrorHandler
}

func (h *FilterReadCommandHandler) Execute(args []string) (*model.CommandResult, error) {
	if len(args) != 1 {
		return &model.CommandResult{Error: true, Message: "Invalid usage: filter-read <read|unread|all>"}, nil
	}

	normalized, label, err := normalizeReadFilterArg(args[0])
	if err != nil {
		return &model.CommandResult{Error: true, Message: err.Error()}, nil
	}

	if h.model.GetReadFilter() == normalized {
		return &model.CommandResult{Message: fmt.Sprintf("Read filter: %s (already set)", label)}, nil
	}

	if err := h.model.SetReadFilter(normalized); err != nil {
		return &model.CommandResult{Error: true, Message: fmt.Sprintf("Failed to set read filter: %v", err)}, nil
	}

	h.model.ApplySearchFilter()
	h.model.ResetCursor()

	if err := h.model.SaveSettings(); err != nil {
		h.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
	}

	return &model.CommandResult{Message: fmt.Sprintf("Read filter: %s", label)}, nil
}

func (h *FilterReadCommandHandler) Validate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("invalid usage: filter-read <read|unread|all>")
	}
	_, _, err := normalizeReadFilterArg(args[0])
	return err
}

func (h *FilterReadCommandHandler) Complete(args []string) []string {
	if len(args) == 0 {
		return []string{"all", settings.ReadFilterUnread, settings.ReadFilterRead}
	}
	if len(args) == 1 {
		var suggestions []string
		for _, candidate := range []string{"all", settings.ReadFilterUnread, settings.ReadFilterRead} {
			if strings.HasPrefix(candidate, strings.ToLower(args[0])) {
				suggestions = append(suggestions, candidate)
			}
		}
		return suggestions
	}
	return []string{}
}

func normalizeReadFilterArg(raw string) (value string, label string, err error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "all", "":
		return "", "all", nil
	case settings.ReadFilterRead:
		return settings.ReadFilterRead, settings.ReadFilterRead, nil
	case settings.ReadFilterUnread:
		return settings.ReadFilterUnread, settings.ReadFilterUnread, nil
	default:
		return "", "", fmt.Errorf("invalid read filter value: %s (expected read|unread|all)", raw)
	}
}
