package state

import (
	"fmt"
	"strings"
	"sync"

	"github.com/cristianoliveira/tmux-intray/internal/tmux"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

// tmuxRuntimeCoordinator coordinates tmux interactions and caches tmux names.
type tmuxRuntimeCoordinator struct {
	client tmux.TmuxClient

	mu             sync.RWMutex
	sessionNames   map[string]string
	windowNames    map[string]string
	paneNames      map[string]string
	sessionsLoaded bool
	windowsLoaded  bool
	panesLoaded    bool
}

// NewRuntimeCoordinator creates a runtime coordinator backed by the provided tmux client.
func NewRuntimeCoordinator(client tmux.TmuxClient) model.RuntimeCoordinator {
	if client == nil {
		client = tmux.NewDefaultClient()
	}

	return &tmuxRuntimeCoordinator{
		client:       client,
		sessionNames: map[string]string{},
		windowNames:  map[string]string{},
		paneNames:    map[string]string{},
	}
}

// EnsureTmuxRunning checks whether tmux server is available.
func (r *tmuxRuntimeCoordinator) EnsureTmuxRunning() bool {
	running, err := r.client.HasSession()
	return err == nil && running
}

// JumpToPane jumps to the target pane through the tmux client.
func (r *tmuxRuntimeCoordinator) JumpToPane(sessionID, windowID, paneID string) bool {
	ok, err := r.client.JumpToPane(sessionID, windowID, paneID)
	return err == nil && ok
}

// ValidatePaneExists checks if the target pane exists.
func (r *tmuxRuntimeCoordinator) ValidatePaneExists(sessionID, windowID, paneID string) (bool, error) {
	return r.client.ValidatePaneExists(sessionID, windowID, paneID)
}

// GetCurrentContext returns current tmux context with best-effort resolved names.
func (r *tmuxRuntimeCoordinator) GetCurrentContext() (*model.TmuxContext, error) {
	ctx, err := r.client.GetCurrentContext()
	if err != nil {
		return nil, err
	}

	sessionName, _ := r.GetSessionName(ctx.SessionID)
	windowName, _ := r.GetWindowName(ctx.WindowID)
	paneName, _ := r.GetPaneName(ctx.PaneID)

	return &model.TmuxContext{
		SessionID:   ctx.SessionID,
		SessionName: sessionName,
		WindowID:    ctx.WindowID,
		WindowName:  windowName,
		PaneID:      ctx.PaneID,
		PaneName:    paneName,
		PanePID:     ctx.PanePID,
	}, nil
}

// ListSessions returns cached session names and lazy-loads cache on first request.
// TODO: Extract duplicate lazy-loading pattern (ListSessions, ListWindows, ListPanes) into a helper function.
func (r *tmuxRuntimeCoordinator) ListSessions() (map[string]string, error) {
	r.mu.RLock()
	if r.sessionsLoaded {
		names := cloneNames(r.sessionNames)
		r.mu.RUnlock()
		return names, nil
	}
	r.mu.RUnlock()

	names, err := r.client.ListSessions()
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.sessionNames = cloneNames(names)
	r.sessionsLoaded = true
	r.mu.Unlock()

	return cloneNames(names), nil
}

// ListWindows returns cached window names and lazy-loads cache on first request.
func (r *tmuxRuntimeCoordinator) ListWindows() (map[string]string, error) {
	r.mu.RLock()
	if r.windowsLoaded {
		names := cloneNames(r.windowNames)
		r.mu.RUnlock()
		return names, nil
	}
	r.mu.RUnlock()

	names, err := r.client.ListWindows()
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.windowNames = cloneNames(names)
	r.windowsLoaded = true
	r.mu.Unlock()

	return cloneNames(names), nil
}

// ListPanes returns cached pane names and lazy-loads cache on first request.
func (r *tmuxRuntimeCoordinator) ListPanes() (map[string]string, error) {
	r.mu.RLock()
	if r.panesLoaded {
		names := cloneNames(r.paneNames)
		r.mu.RUnlock()
		return names, nil
	}
	r.mu.RUnlock()

	names, err := r.client.ListPanes()
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.paneNames = cloneNames(names)
	r.panesLoaded = true
	r.mu.Unlock()

	return cloneNames(names), nil
}

// GetSessionName resolves a session ID into a session name.
func (r *tmuxRuntimeCoordinator) GetSessionName(sessionID string) (string, error) {
	if sessionID == "" {
		return "", nil
	}

	r.mu.RLock()
	if name, ok := r.sessionNames[sessionID]; ok {
		r.mu.RUnlock()
		return name, nil
	}
	loaded := r.sessionsLoaded
	r.mu.RUnlock()

	// Always try to fetch the name if not in cache, even if we previously loaded all sessions
	name, err := r.client.GetSessionName(sessionID)
	if err != nil {
		// If we already loaded all sessions and the session is missing, treat ID as name
		// (session might have been deleted)
		if loaded {
			return sessionID, nil
		}
		return sessionID, err
	}

	r.mu.Lock()
	r.sessionNames[sessionID] = name
	r.sessionsLoaded = true
	r.mu.Unlock()

	return name, nil
}

// GetWindowName resolves a window ID into a window name.
// TODO: RefreshNames refreshes all three caches; consider refreshing only missing cache (windows/panes).
func (r *tmuxRuntimeCoordinator) GetWindowName(windowID string) (string, error) {
	if windowID == "" {
		return "", nil
	}

	r.mu.RLock()
	if name, ok := r.windowNames[windowID]; ok {
		r.mu.RUnlock()
		return name, nil
	}
	r.mu.RUnlock()

	if err := r.RefreshNames(); err != nil {
		return windowID, err
	}

	r.mu.RLock()
	name, ok := r.windowNames[windowID]
	r.mu.RUnlock()
	if ok {
		return name, nil
	}

	return windowID, nil
}

// GetPaneName resolves a pane ID into a pane name.
func (r *tmuxRuntimeCoordinator) GetPaneName(paneID string) (string, error) {
	if paneID == "" {
		return "", nil
	}

	r.mu.RLock()
	if name, ok := r.paneNames[paneID]; ok {
		r.mu.RUnlock()
		return name, nil
	}
	r.mu.RUnlock()

	if err := r.RefreshNames(); err != nil {
		return paneID, err
	}

	r.mu.RLock()
	name, ok := r.paneNames[paneID]
	r.mu.RUnlock()
	if ok {
		return name, nil
	}

	return paneID, nil
}

// RefreshNames updates all tmux name caches with latest data.
func (r *tmuxRuntimeCoordinator) RefreshNames() error {
	sessionNames, sessionErr := r.client.ListSessions()
	windowNames, windowErr := r.client.ListWindows()
	paneNames, paneErr := r.client.ListPanes()

	r.mu.Lock()
	if sessionErr == nil {
		r.sessionNames = cloneNames(sessionNames)
		r.sessionsLoaded = true
	}
	if windowErr == nil {
		r.windowNames = cloneNames(windowNames)
		r.windowsLoaded = true
	}
	if paneErr == nil {
		r.paneNames = cloneNames(paneNames)
		r.panesLoaded = true
	}
	r.mu.Unlock()

	return combineRefreshErrors(sessionErr, windowErr, paneErr)
}

// GetTmuxVisibility retrieves tmux visibility state.
func (r *tmuxRuntimeCoordinator) GetTmuxVisibility() (bool, error) {
	return r.client.GetTmuxVisibility()
}

// SetTmuxVisibility updates tmux visibility state.
func (r *tmuxRuntimeCoordinator) SetTmuxVisibility(visible bool) error {
	return r.client.SetTmuxVisibility(visible)
}

func cloneNames(names map[string]string) map[string]string {
	clone := make(map[string]string, len(names))
	for k, v := range names {
		clone[k] = v
	}
	return clone
}

func combineRefreshErrors(errs ...error) error {
	messages := make([]string, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			messages = append(messages, err.Error())
		}
	}
	if len(messages) == 0 {
		return nil
	}
	return fmt.Errorf("refresh tmux names: %s", strings.Join(messages, "; "))
}

// TODO: Extract duplicate lazy-loading pattern (ListSessions, ListWindows, ListPanes) into a helper function.
