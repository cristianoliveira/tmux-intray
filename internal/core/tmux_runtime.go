package core

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/colors"
	"github.com/cristianoliveira/tmux-intray/internal/ports"
)

const tmuxIntrayVisibleEnv = "TMUX_INTRAY_VISIBLE"

type tmuxRuntime struct {
	client ports.TmuxClient
}

func newTmuxRuntime(client ports.TmuxClient) *tmuxRuntime {
	return &tmuxRuntime{client: client}
}

func (r *tmuxRuntime) ensureTmuxRunning() bool {
	running, err := r.client.HasSession()
	if err != nil {
		colors.Debug("EnsureTmuxRunning: tmux has-session failed: " + err.Error())
		return false
	}
	return running
}

func (r *tmuxRuntime) currentContext() TmuxContext {
	ctx, err := r.client.GetCurrentContext()
	if err != nil {
		colors.Error("get current tmux context: failed to get tmux context: " + err.Error())
		return TmuxContext{}
	}

	return TmuxContext{
		SessionID:   ctx.SessionID,
		WindowID:    ctx.WindowID,
		PaneID:      ctx.PaneID,
		PaneCreated: ctx.PanePID,
	}
}

func (r *tmuxRuntime) validatePaneExists(sessionID, windowID, paneID string) bool {
	exists, err := r.client.ValidatePaneExists(sessionID, windowID, paneID)
	if err != nil {
		colors.Debug(fmt.Sprintf("ValidatePaneExists: tmux list-panes failed for %s:%s.%s: %v", sessionID, windowID, paneID, err))
		return false
	}
	return exists
}

func (r *tmuxRuntime) getVisibility() string {
	value, err := r.client.GetEnvironment(tmuxIntrayVisibleEnv)
	if err != nil {
		colors.Debug("GetTmuxVisibility: tmux show-environment failed: " + err.Error())
		return "0"
	}
	return value
}

func (r *tmuxRuntime) setVisibility(value string) (bool, error) {
	err := r.client.SetEnvironment(tmuxIntrayVisibleEnv, value)
	if err != nil {
		colors.Error(fmt.Sprintf("set tmux visibility: failed to set %s to '%s': %v", tmuxIntrayVisibleEnv, value, err))
		return false, fmt.Errorf("set tmux visibility: %w", err)
	}
	return true, nil
}
