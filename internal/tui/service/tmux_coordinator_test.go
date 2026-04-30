package service

import "testing"

func TestRuntimeCoordinatorGetNamesUsesReadableFallbackForStaleTmuxIDs(t *testing.T) {
	coordinator := &DefaultRuntimeCoordinator{
		sessionNames: map[string]string{},
		windowNames:  map[string]string{},
		paneNames:    map[string]string{},
	}

	if got, err := coordinator.GetSessionName("$180"); err != nil || got != "stale-session:$180" {
		t.Fatalf("GetSessionName() = %q, %v; want %q, nil", got, err, "stale-session:$180")
	}
	if got, err := coordinator.GetWindowName("@329"); err != nil || got != "stale-window:@329" {
		t.Fatalf("GetWindowName() = %q, %v; want %q, nil", got, err, "stale-window:@329")
	}
	if got, err := coordinator.GetPaneName("%703"); err != nil || got != "stale-pane:%703" {
		t.Fatalf("GetPaneName() = %q, %v; want %q, nil", got, err, "stale-pane:%703")
	}
}

func TestRuntimeCoordinatorResolveNamesUsesReadableFallbackForStaleTmuxIDs(t *testing.T) {
	coordinator := &DefaultRuntimeCoordinator{
		sessionNames: map[string]string{},
		windowNames:  map[string]string{},
		paneNames:    map[string]string{},
	}

	if got := coordinator.ResolveSessionName("$180"); got != "stale-session:$180" {
		t.Fatalf("ResolveSessionName() = %q, want %q", got, "stale-session:$180")
	}
	if got := coordinator.ResolveWindowName("@329"); got != "stale-window:@329" {
		t.Fatalf("ResolveWindowName() = %q, want %q", got, "stale-window:@329")
	}
	if got := coordinator.ResolvePaneName("%703"); got != "stale-pane:%703" {
		t.Fatalf("ResolvePaneName() = %q, want %q", got, "stale-pane:%703")
	}
}

func TestRuntimeCoordinatorResolveNamesKeepsKnownTmuxNames(t *testing.T) {
	coordinator := &DefaultRuntimeCoordinator{
		sessionNames: map[string]string{"$1": "work"},
		windowNames:  map[string]string{"@2": "editor"},
		paneNames:    map[string]string{"%3": "server"},
	}

	if got := coordinator.ResolveSessionName("$1"); got != "work" {
		t.Fatalf("ResolveSessionName() = %q, want %q", got, "work")
	}
	if got := coordinator.ResolveWindowName("@2"); got != "editor" {
		t.Fatalf("ResolveWindowName() = %q, want %q", got, "editor")
	}
	if got := coordinator.ResolvePaneName("%3"); got != "server" {
		t.Fatalf("ResolvePaneName() = %q, want %q", got, "server")
	}
}
