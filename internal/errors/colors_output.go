package errors

import "github.com/cristianoliveira/tmux-intray/internal/colors"

// ColorsOutput adapts the colors package to implement ColorOutput.
type ColorsOutput struct{}

var _ ColorOutput = (*ColorsOutput)(nil)

func (o *ColorsOutput) Error(msgs ...string) {
	colors.Error(msgs...)
}

func (o *ColorsOutput) Warning(msgs ...string) {
	colors.Warning(msgs...)
}

func (o *ColorsOutput) Info(msgs ...string) {
	colors.Info(msgs...)
}

func (o *ColorsOutput) Success(msgs ...string) {
	colors.Success(msgs...)
}

// NewDefaultCLIHandler creates a CLI handler using ColorsOutput.
func NewDefaultCLIHandler() *CLIHandler {
	return NewCLIHandler(&ColorsOutput{})
}
