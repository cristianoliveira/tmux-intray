// Package model provides interface contracts for TUI components.
// These interfaces define the contracts between different parts of the TUI system.
package model

// CommandService defines the interface for command parsing and execution.
// It handles the command-line interface within the TUI (e.g., :q, :w, :group-by).
type CommandService interface {
	// ParseCommand parses a command string into its constituent parts.
	// Returns the command name, arguments, and an error if parsing fails.
	ParseCommand(command string) (name string, args []string, err error)

	// ExecuteCommand executes a parsed command and returns the result.
	// The command should update the appropriate state and return a CommandResult.
	ExecuteCommand(name string, args []string) (*CommandResult, error)

	// ValidateCommand checks if a command and its arguments are valid.
	// Returns an error if the command or arguments are invalid.
	ValidateCommand(name string, args []string) error

	// GetAvailableCommands returns a list of all available commands.
	// Useful for help text and command completion.
	GetAvailableCommands() []CommandInfo

	// GetCommandHelp returns help text for a specific command.
	// Returns empty string if the command doesn't exist.
	GetCommandHelp(name string) string

	// GetCommandSuggestions returns suggestions for command completion.
	// Takes a partial command string and returns matching commands or arguments.
	GetCommandSuggestions(partial string) []string
}

// CommandInfo provides metadata about a command.
type CommandInfo struct {
	// Name is the command name.
	Name string

	// Description is a short description of what the command does.
	Description string

	// Usage shows the expected syntax for the command.
	Usage string

	// Examples show example invocations of the command.
	Examples []string

	// Aliases are alternative names for the command.
	Aliases []string
}

// CommandServiceBuilder is a builder for creating configured CommandService instances.
type CommandServiceBuilder interface {
	// WithCommandHandler registers a handler for a specific command.
	WithCommandHandler(name string, handler CommandHandler) CommandServiceBuilder

	// WithCommandAlias registers an alias for an existing command.
	WithCommandAlias(alias, target string) CommandServiceBuilder

	// WithHelpProvider sets a custom help provider.
	WithHelpProvider(provider HelpProvider) CommandServiceBuilder

	// Build creates and returns a configured CommandService.
	Build() (CommandService, error)
}

// CommandHandler handles the execution of a specific command.
type CommandHandler interface {
	// Execute executes the command with the given arguments.
	Execute(args []string) (*CommandResult, error)

	// Validate checks if the arguments are valid for this command.
	Validate(args []string) error

	// Complete returns completion suggestions for the arguments.
	Complete(args []string) []string
}

// HelpProvider provides help text for commands.
type HelpProvider interface {
	// GetHelp returns help text for a command.
	GetHelp(name string) string

	// GetAllHelp returns help text for all commands.
	GetAllHelp() []CommandHelp
}

// CommandHelp contains formatted help information for a command.
type CommandHelp struct {
	// Name is the command name.
	Name string

	// Description is a short description.
	Description string

	// Usage shows the command syntax.
	Usage string

	// Aliases lists alternative names.
	Aliases []string

	// Arguments describes the command arguments.
	Arguments []ArgumentHelp

	// Examples show example usage.
	Examples []string
}

// ArgumentHelp describes a command argument.
type ArgumentHelp struct {
	// Name is the argument name.
	Name string

	// Description explains what the argument does.
	Description string

	// Required indicates whether the argument is required.
	Required bool

	// Options lists valid values for the argument (if applicable).
	Options []string
}
