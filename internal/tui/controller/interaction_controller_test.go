package controller

import (
	"errors"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

type fakeNotificationStore struct {
	listOutput         string
	listErr            error
	dismissID          string
	dismissFilter      [3]string
	markReadID         string
	markUnreadID       string
	dismissErr         error
	dismissByFilterErr error
	markReadErr        error
	markUnreadErr      error
}

func (f *fakeNotificationStore) ListActiveNotifications() (string, error) {
	return f.listOutput, f.listErr
}

func (f *fakeNotificationStore) ListAllNotifications() (string, error) {
	return f.listOutput, f.listErr
}

func (f *fakeNotificationStore) DismissNotification(id string) error {
	f.dismissID = id
	return f.dismissErr
}

func (f *fakeNotificationStore) DismissByFilter(session, window, pane string) error {
	f.dismissFilter = [3]string{session, window, pane}
	return f.dismissByFilterErr
}

func (f *fakeNotificationStore) MarkNotificationRead(id string) error {
	f.markReadID = id
	return f.markReadErr
}

func (f *fakeNotificationStore) MarkNotificationUnread(id string) error {
	f.markUnreadID = id
	return f.markUnreadErr
}

type fakeNotificationParser struct {
	parsed map[string]notification.Notification
	errFor map[string]error
}

func (f *fakeNotificationParser) Parse(line string) (notification.Notification, error) {
	if err, ok := f.errFor[line]; ok {
		return notification.Notification{}, err
	}
	notif, ok := f.parsed[line]
	if !ok {
		return notification.Notification{}, errors.New("unexpected line")
	}
	return notif, nil
}

type fakeRuntimeCoordinator struct{}

func (f fakeRuntimeCoordinator) EnsureTmuxRunning() bool                            { return true }
func (f fakeRuntimeCoordinator) JumpToPane(sessionID, windowID, paneID string) bool { return true }
func (f fakeRuntimeCoordinator) JumpToWindow(sessionID, windowID string) bool       { return true }
func (f fakeRuntimeCoordinator) ValidatePaneExists(sessionID, windowID, paneID string) (bool, error) {
	return true, nil
}
func (f fakeRuntimeCoordinator) GetCurrentContext() (*model.TmuxContext, error)  { return nil, nil }
func (f fakeRuntimeCoordinator) ListSessions() (map[string]string, error)        { return nil, nil }
func (f fakeRuntimeCoordinator) ListWindows() (map[string]string, error)         { return nil, nil }
func (f fakeRuntimeCoordinator) ListPanes() (map[string]string, error)           { return nil, nil }
func (f fakeRuntimeCoordinator) GetSessionName(sessionID string) (string, error) { return "", nil }
func (f fakeRuntimeCoordinator) GetWindowName(windowID string) (string, error)   { return "", nil }
func (f fakeRuntimeCoordinator) GetPaneName(paneID string) (string, error)       { return "", nil }
func (f fakeRuntimeCoordinator) RefreshNames() error                             { return nil }
func (f fakeRuntimeCoordinator) GetTmuxVisibility() (bool, error)                { return true, nil }
func (f fakeRuntimeCoordinator) SetTmuxVisibility(visible bool) error            { return nil }
func (f fakeRuntimeCoordinator) ResolveSessionName(sessionID string) string      { return "" }
func (f fakeRuntimeCoordinator) ResolveWindowName(windowID string) string        { return "" }
func (f fakeRuntimeCoordinator) ResolvePaneName(paneID string) string            { return "" }
func (f fakeRuntimeCoordinator) GetSessionNames() map[string]string              { return nil }
func (f fakeRuntimeCoordinator) GetWindowNames() map[string]string               { return nil }
func (f fakeRuntimeCoordinator) GetPaneNames() map[string]string                 { return nil }
func (f fakeRuntimeCoordinator) SetSessionNames(names map[string]string)         {}
func (f fakeRuntimeCoordinator) SetWindowNames(names map[string]string)          {}
func (f fakeRuntimeCoordinator) SetPaneNames(names map[string]string)            {}

type trackingRuntimeCoordinator struct {
	fakeRuntimeCoordinator
	ensureResult     bool
	jumpPaneResult   bool
	jumpWindowResult bool
	ensureCalls      int
	jumpPaneCalls    int
	jumpWindowCalls  int
	jumpPaneArgs     [3]string
	jumpWindowArgs   [2]string
}

func (t *trackingRuntimeCoordinator) EnsureTmuxRunning() bool {
	t.ensureCalls++
	return t.ensureResult
}

func (t *trackingRuntimeCoordinator) JumpToPane(sessionID, windowID, paneID string) bool {
	t.jumpPaneCalls++
	t.jumpPaneArgs = [3]string{sessionID, windowID, paneID}
	return t.jumpPaneResult
}

func (t *trackingRuntimeCoordinator) JumpToWindow(sessionID, windowID string) bool {
	t.jumpWindowCalls++
	t.jumpWindowArgs = [2]string{sessionID, windowID}
	return t.jumpWindowResult
}

func TestLoadActiveNotifications_UsesInjectedAdapters(t *testing.T) {
	store := &fakeNotificationStore{listOutput: "line-1\nline-bad\nline-2\n"}
	parser := &fakeNotificationParser{
		parsed: map[string]notification.Notification{
			"line-1": {ID: 1, Message: "one"},
			"line-2": {ID: 2, Message: "two"},
		},
		errFor: map[string]error{"line-bad": errors.New("bad")},
	}

	controller := NewInteractionControllerWithAdapters(fakeRuntimeCoordinator{}, store, parser)

	notifications, err := controller.LoadActiveNotifications()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(notifications) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(notifications))
	}
	if notifications[0].ID != 1 || notifications[1].ID != 2 {
		t.Fatalf("unexpected notifications returned: %#v", notifications)
	}
}

func TestLoadActiveNotifications_ReturnsStoreErrors(t *testing.T) {
	store := &fakeNotificationStore{listErr: errors.New("storage down")}
	parser := &fakeNotificationParser{parsed: map[string]notification.Notification{}}

	controller := NewInteractionControllerWithAdapters(fakeRuntimeCoordinator{}, store, parser)

	_, err := controller.LoadActiveNotifications()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestMutationMethods_DelegateToStore(t *testing.T) {
	store := &fakeNotificationStore{}
	parser := &fakeNotificationParser{parsed: map[string]notification.Notification{}}

	controller := NewInteractionControllerWithAdapters(fakeRuntimeCoordinator{}, store, parser)

	if err := controller.DismissNotification("7"); err != nil {
		t.Fatalf("dismiss failed: %v", err)
	}
	if err := controller.DismissByFilter("$1", "@2", "%3"); err != nil {
		t.Fatalf("dismiss by filter failed: %v", err)
	}
	if err := controller.MarkNotificationRead("8"); err != nil {
		t.Fatalf("mark read failed: %v", err)
	}
	if err := controller.MarkNotificationUnread("9"); err != nil {
		t.Fatalf("mark unread failed: %v", err)
	}

	if store.dismissID != "7" {
		t.Fatalf("expected dismiss id 7, got %s", store.dismissID)
	}
	if store.dismissFilter != [3]string{"$1", "@2", "%3"} {
		t.Fatalf("unexpected dismiss filter values: %#v", store.dismissFilter)
	}
	if store.markReadID != "8" {
		t.Fatalf("expected mark read id 8, got %s", store.markReadID)
	}
	if store.markUnreadID != "9" {
		t.Fatalf("expected mark unread id 9, got %s", store.markUnreadID)
	}
}

func TestLoadActiveNotifications_ReturnsEmptySliceForNoRows(t *testing.T) {
	store := &fakeNotificationStore{listOutput: ""}
	parser := &fakeNotificationParser{parsed: map[string]notification.Notification{}}

	controller := NewInteractionControllerWithAdapters(fakeRuntimeCoordinator{}, store, parser)

	notifications, err := controller.LoadActiveNotifications()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(notifications) != 0 {
		t.Fatalf("expected empty notifications, got %#v", notifications)
	}
}

func TestNewInteractionController_UsesDefaultAdapters(t *testing.T) {
	controller := NewInteractionController(fakeRuntimeCoordinator{})
	impl, ok := controller.(*DefaultInteractionController)
	if !ok {
		t.Fatalf("expected *DefaultInteractionController, got %T", controller)
	}
	if _, ok := impl.store.(storageNotificationStore); !ok {
		t.Fatalf("expected storageNotificationStore, got %T", impl.store)
	}
	if _, ok := impl.parser.(defaultNotificationParser); !ok {
		t.Fatalf("expected defaultNotificationParser, got %T", impl.parser)
	}
}

func TestNewInteractionControllerWithAdapters_DefaultsNilAdapters(t *testing.T) {
	controller := NewInteractionControllerWithAdapters(fakeRuntimeCoordinator{}, nil, nil)
	impl, ok := controller.(*DefaultInteractionController)
	if !ok {
		t.Fatalf("expected *DefaultInteractionController, got %T", controller)
	}
	if _, ok := impl.store.(storageNotificationStore); !ok {
		t.Fatalf("expected default store when nil, got %T", impl.store)
	}
	if _, ok := impl.parser.(defaultNotificationParser); !ok {
		t.Fatalf("expected default parser when nil, got %T", impl.parser)
	}
}

func TestDefaultNotificationParser_ParseInvalidLine(t *testing.T) {
	_, err := (defaultNotificationParser{}).Parse("not-a-valid-notification-line")
	if err == nil {
		t.Fatal("expected parser error for invalid notification line")
	}
}

func TestRuntimeMethods_DelegateAndHandleNilCoordinator(t *testing.T) {
	controller := NewInteractionControllerWithAdapters(nil, &fakeNotificationStore{}, &fakeNotificationParser{parsed: map[string]notification.Notification{}})

	if controller.EnsureTmuxRunning() {
		t.Fatal("expected EnsureTmuxRunning to be false with nil runtime coordinator")
	}
	if controller.JumpToPane("$1", "1", "%1") {
		t.Fatal("expected JumpToPane to be false with nil runtime coordinator")
	}
	if controller.JumpToWindow("$1", "1") {
		t.Fatal("expected JumpToWindow to be false with nil runtime coordinator")
	}

	tracking := &trackingRuntimeCoordinator{
		ensureResult:     true,
		jumpPaneResult:   true,
		jumpWindowResult: true,
	}

	impl, ok := controller.(*DefaultInteractionController)
	if !ok {
		t.Fatalf("expected *DefaultInteractionController, got %T", controller)
	}
	impl.SetRuntimeCoordinator(tracking)

	if !controller.EnsureTmuxRunning() {
		t.Fatal("expected EnsureTmuxRunning to delegate to runtime coordinator")
	}
	if !controller.JumpToPane("$2", "3", "%4") {
		t.Fatal("expected JumpToPane to delegate to runtime coordinator")
	}
	if !controller.JumpToWindow("$2", "3") {
		t.Fatal("expected JumpToWindow to delegate to runtime coordinator")
	}

	if tracking.ensureCalls != 1 {
		t.Fatalf("expected one EnsureTmuxRunning call, got %d", tracking.ensureCalls)
	}
	if tracking.jumpPaneCalls != 1 || tracking.jumpPaneArgs != [3]string{"$2", "3", "%4"} {
		t.Fatalf("unexpected JumpToPane calls/args: calls=%d args=%#v", tracking.jumpPaneCalls, tracking.jumpPaneArgs)
	}
	if tracking.jumpWindowCalls != 1 || tracking.jumpWindowArgs != [2]string{"$2", "3"} {
		t.Fatalf("unexpected JumpToWindow calls/args: calls=%d args=%#v", tracking.jumpWindowCalls, tracking.jumpWindowArgs)
	}
}
