package errors

import (
	"sync"
)

// ErrorHandler is the interface for error handling.
// Different implementations can handle errors differently based on context.
type ErrorHandler interface {
	Error(msg string)
	Warning(msg string)
	Info(msg string)
	Success(msg string)
}

// CLIHandler handles errors by printing to stdout/stderr using the colors package.
type CLIHandler struct {
	colors     ColorOutput
	mu         sync.Mutex
	inHandling bool
}

type ColorOutput interface {
	Error(msgs ...string)
	Warning(msgs ...string)
	Info(msgs ...string)
	Success(msgs ...string)
}

func NewCLIHandler(colors ColorOutput) *CLIHandler {
	return &CLIHandler{colors: colors}
}

func (h *CLIHandler) Error(msg string) {
	h.mu.Lock()
	if h.inHandling {
		h.mu.Unlock()
		h.colors.Error(msg)
		return
	}
	h.inHandling = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		h.inHandling = false
		h.mu.Unlock()
	}()

	h.colors.Error(msg)
}

func (h *CLIHandler) Warning(msg string) {
	h.colors.Warning(msg)
}

func (h *CLIHandler) Info(msg string) {
	h.colors.Info(msg)
}

func (h *CLIHandler) Success(msg string) {
	h.colors.Success(msg)
}
