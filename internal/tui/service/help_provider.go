package service

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// DefaultHelpProvider implements the HelpProvider interface.
type DefaultHelpProvider struct {
	commands map[string]*model.CommandHelp
}

// NewDefaultHelpProvider creates a new DefaultHelpProvider with default command help.
func NewDefaultHelpProvider() model.HelpProvider {
	provider := &DefaultHelpProvider{
		commands: make(map[string]*model.CommandHelp),
	}

	// Register default command help
	provider.registerDefaultHelp()

	return provider
}

// GetHelp returns help text for a command.
func (p *DefaultHelpProvider) GetHelp(name string) string {
	help, ok := p.commands[name]
	if !ok {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s - %s\n", help.Name, help.Description))
	sb.WriteString(fmt.Sprintf("Usage: %s\n", help.Usage))

	if len(help.Aliases) > 0 {
		sb.WriteString(fmt.Sprintf("Aliases: %s\n", strings.Join(help.Aliases, ", ")))
	}

	if len(help.Arguments) > 0 {
		sb.WriteString("Arguments:\n")
		for _, arg := range help.Arguments {
			required := "optional"
			if arg.Required {
				required = "required"
			}
			sb.WriteString(fmt.Sprintf("  %s - %s (%s)\n", arg.Name, arg.Description, required))
		}
	}

	if len(help.Examples) > 0 {
		sb.WriteString("Examples:\n")
		for _, example := range help.Examples {
			sb.WriteString(fmt.Sprintf("  %s\n", example))
		}
	}

	return sb.String()
}

// GetAllHelp returns help text for all commands.
func (p *DefaultHelpProvider) GetAllHelp() []model.CommandHelp {
	var helps []model.CommandHelp
	for _, help := range p.commands {
		helps = append(helps, *help)
	}
	return helps
}

// registerDefaultHelp registers help for the default commands.
func (p *DefaultHelpProvider) registerDefaultHelp() {
	// Quit command
	p.commands["q"] = &model.CommandHelp{
		Name:        "q",
		Description: "Quit the application",
		Usage:       "q",
		Arguments:   []model.ArgumentHelp{},
		Examples:    []string{":q"},
	}

	// Write command
	p.commands["w"] = &model.CommandHelp{
		Name:        "w",
		Description: "Save settings",
		Usage:       "w",
		Arguments:   []model.ArgumentHelp{},
		Examples:    []string{":w"},
	}

	// Group-by command
	p.commands["group-by"] = &model.CommandHelp{
		Name:        "group-by",
		Description: "Set the grouping mode for notifications",
		Usage:       "group-by <none|session|window|pane>",
		Arguments: []model.ArgumentHelp{
			{
				Name:        "mode",
				Description: "Grouping mode",
				Required:    true,
				Options:     []string{"none", "session", "window", "pane"},
			},
		},
		Examples: []string{
			":group-by none",
			":group-by session",
			":group-by window",
			":group-by pane",
		},
	}

	// Expand-level command
	p.commands["expand-level"] = &model.CommandHelp{
		Name:        "expand-level",
		Description: "Set the default expansion level for grouped views",
		Usage:       "expand-level <0|1|2|3>",
		Arguments: []model.ArgumentHelp{
			{
				Name:        "level",
				Description: "Expansion level (0=closed, 1=session, 2=window, 3=pane)",
				Required:    true,
				Options:     []string{"0", "1", "2", "3"},
			},
		},
		Examples: []string{
			":expand-level 0",
			":expand-level 1",
			":expand-level 2",
			":expand-level 3",
		},
	}

	// Toggle-view command
	p.commands["toggle-view"] = &model.CommandHelp{
		Name:        "toggle-view",
		Description: "Toggle between detailed and grouped view modes",
		Usage:       "toggle-view",
		Arguments:   []model.ArgumentHelp{},
		Examples:    []string{":toggle-view"},
	}
}
