package service

import (
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
)

type testNameResolver struct {
	sessions map[string]string
	windows  map[string]string
	panes    map[string]string
}

func (r *testNameResolver) ResolveSessionName(id string) string     { return r.sessions[id] }
func (r *testNameResolver) ResolveWindowName(id string) string      { return r.windows[id] }
func (r *testNameResolver) ResolvePaneName(id string) string        { return r.panes[id] }
func (r *testNameResolver) GetSessionNames() map[string]string      { return r.sessions }
func (r *testNameResolver) GetWindowNames() map[string]string       { return r.windows }
func (r *testNameResolver) GetPaneNames() map[string]string         { return r.panes }
func (r *testNameResolver) SetSessionNames(names map[string]string) { r.sessions = names }
func (r *testNameResolver) SetWindowNames(names map[string]string)  { r.windows = names }
func (r *testNameResolver) SetPaneNames(names map[string]string)    { r.panes = names }

func TestApplyFiltersAndSearchHidesStaleTmuxTargetsByDefault(t *testing.T) {
	svc := NewNotificationService(nil, &testNameResolver{
		sessions: map[string]string{"$1": "work"},
		windows:  map[string]string{"@1": "editor"},
		panes:    map[string]string{"%1": "shell"},
	})
	svc.SetNotifications([]domain.Notification{
		{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "live", State: domain.StateActive},
		{ID: 2, Session: "$2", Window: "@2", Pane: "%2", Message: "stale", State: domain.StateActive},
	})

	svc.ApplyFiltersAndSearch(settings.TabAll, "", "", "", "", "", "", "", "timestamp", "desc")

	got := svc.GetFilteredNotifications()
	if len(got) != 1 || got[0].Message != "live" {
		t.Fatalf("expected only live notification, got %#v", got)
	}
}

func TestApplyFiltersAndSearchShowsStaleTmuxTargetsWhenEnabled(t *testing.T) {
	svc := NewNotificationService(nil, &testNameResolver{
		sessions: map[string]string{"$1": "work"},
		windows:  map[string]string{"@1": "editor"},
		panes:    map[string]string{"%1": "shell"},
	})
	svc.SetShowStale(true)
	svc.SetNotifications([]domain.Notification{
		{ID: 1, Session: "$1", Window: "@1", Pane: "%1", Message: "live", State: domain.StateActive},
		{ID: 2, Session: "$2", Window: "@2", Pane: "%2", Message: "stale", State: domain.StateActive},
	})

	svc.ApplyFiltersAndSearch(settings.TabAll, "", "", "", "", "", "", "", "timestamp", "desc")

	if got := svc.GetFilteredNotifications(); len(got) != 2 {
		t.Fatalf("expected live and stale notifications, got %#v", got)
	}
}
