package app

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
)

// JumpClient defines dependencies required by jump command.
type JumpClient interface {
	EnsureTmuxRunning() bool
	GetNotificationByID(id string) (string, error)
	ValidatePaneExists(session, window, pane string) bool
	JumpToPane(session, window, pane string) bool
	MarkNotificationRead(id string) error
}

// JumpUseCase coordinates jump command behavior.
type JumpUseCase struct {
	client JumpClient
}

// NewJumpUseCase creates a jump use-case.
func NewJumpUseCase(client JumpClient) *JumpUseCase {
	if client == nil {
		panic("NewJumpUseCase: client dependency cannot be nil")
	}

	return &JumpUseCase{client: client}
}

// JumpInput contains jump options.
type JumpInput struct {
	ID         string
	NoMarkRead bool
}

type jumpDetails struct {
	state   string
	session string
	window  string
	pane    string
}

// Execute performs jump behavior preserving CLI output and errors.
func (u *JumpUseCase) Execute(input JumpInput) error {
	if !u.client.EnsureTmuxRunning() {
		return fmt.Errorf("tmux not running")
	}

	details, err := u.loadJumpDetails(input.ID)
	if err != nil {
		return err
	}

	paneExists, err := u.performJump(input.ID, details, input.NoMarkRead)
	if err != nil {
		return err
	}

	reportJumpOutcome(input.ID, details, paneExists)
	return nil
}

func (u *JumpUseCase) loadJumpDetails(id string) (jumpDetails, error) {
	line, err := u.client.GetNotificationByID(id)
	if err != nil {
		return jumpDetails{}, fmt.Errorf("jump: %w", err)
	}

	return parseJumpDetails(id, line)
}

func parseJumpDetails(id, line string) (jumpDetails, error) {
	fields := strings.Split(line, "\t")
	if len(fields) <= storage.FieldPane {
		return jumpDetails{}, fmt.Errorf("jump: invalid notification line format")
	}

	details := jumpDetails{
		state:   fields[storage.FieldState],
		session: fields[storage.FieldSession],
		window:  fields[storage.FieldWindow],
		pane:    fields[storage.FieldPane],
	}

	if err := validateJumpFields(id, details); err != nil {
		return jumpDetails{}, err
	}

	return details, nil
}

func validateJumpFields(id string, details jumpDetails) error {
	if details.session == "" || details.window == "" || details.pane == "" {
		var missingFields []string
		if details.session == "" {
			missingFields = append(missingFields, "session")
		}
		if details.window == "" {
			missingFields = append(missingFields, "window")
		}
		if details.pane == "" {
			missingFields = append(missingFields, "pane")
		}

		return fmt.Errorf(
			"jump: notification %s missing required fields:\n"+
				"  missing: %s\n"+
				"  required fields: session, window, pane\n"+
				"  hint: notifications must be created from within an active tmux session for jump to work",
			id, strings.Join(missingFields, ", "))
	}

	return nil
}

func (u *JumpUseCase) performJump(id string, details jumpDetails, noMarkRead bool) (bool, error) {
	paneExists := u.client.ValidatePaneExists(details.session, details.window, details.pane)

	if !u.client.JumpToPane(details.session, details.window, details.pane) {
		return paneExists, fmt.Errorf("jump: failed to jump because pane or window does not exist")
	}

	if !noMarkRead {
		if err := u.client.MarkNotificationRead(id); err != nil {
			return paneExists, fmt.Errorf("jump: failed to mark notification as read: %w", err)
		}
	}

	return paneExists, nil
}

func reportJumpOutcome(id string, details jumpDetails, paneExists bool) {
	if details.state == "dismissed" {
		colors.Info(fmt.Sprintf("Notification %s is dismissed, but jumping anyway", id))
	}

	if paneExists {
		colors.Success(fmt.Sprintf("Jumped to session %s, window %s, pane %s", details.session, details.window, details.pane))
		return
	}

	colors.Warning(fmt.Sprintf("Pane %s no longer exists (jumped to window %s:%s instead)", details.pane, details.session, details.window))
}
