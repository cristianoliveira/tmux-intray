// Package controller provides side-effect orchestration for TUI interactions.
package controller

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

var errTypedNotificationListingUnsupported = errors.New("typed notification listing unsupported")

type notificationStore interface {
	ListActiveNotifications() (string, error)
	ListAllNotifications() (string, error)
	DismissNotification(id string) error
	DismissByFilter(session, window, pane string) error
	MarkNotificationRead(id string) error
	MarkNotificationUnread(id string) error
}

type typedNotificationStore interface {
	ListActiveNotificationValues() ([]notification.Notification, error)
	ListAllNotificationValues() ([]notification.Notification, error)
}

type notificationParser interface {
	Parse(line string) (domain.Notification, error)
}

type storageNotificationStore struct{}

func (s storageNotificationStore) ListActiveNotifications() (string, error) {
	return storage.ListNotifications("active", "", "", "", "", "", "", "")
}

func (s storageNotificationStore) ListAllNotifications() (string, error) {
	return storage.ListNotifications("", "", "", "", "", "", "", "")
}

func (s storageNotificationStore) ListActiveNotificationValues() ([]notification.Notification, error) {
	return s.listNotificationValues("active")
}

func (s storageNotificationStore) ListAllNotificationValues() ([]notification.Notification, error) {
	return s.listNotificationValues("")
}

func (s storageNotificationStore) listNotificationValues(state string) ([]notification.Notification, error) {
	lines, err := storage.ListNotifications(state, "", "", "", "", "", "", "")
	if err != nil || lines == "" {
		return []notification.Notification{}, err
	}

	items := parseNotificationLines(lines, defaultNotificationParser{})
	values := make([]notification.Notification, 0, len(items))
	for i := range items {
		values = append(values, notification.FromDomain(&items[i]))
	}
	return values, nil
}

func (s storageNotificationStore) DismissNotification(id string) error {
	return storage.DismissNotification(id)
}

func (s storageNotificationStore) DismissByFilter(session, window, pane string) error {
	return storage.DismissByFilter(session, window, pane)
}

func (s storageNotificationStore) MarkNotificationRead(id string) error {
	return storage.MarkNotificationRead(id)
}

func (s storageNotificationStore) MarkNotificationUnread(id string) error {
	return storage.MarkNotificationUnread(id)
}

type defaultNotificationParser struct{}

func (p defaultNotificationParser) Parse(line string) (domain.Notification, error) {
	return domain.ParseNotificationLine(line)
}

// DefaultInteractionController is the production controller implementation.
type DefaultInteractionController struct {
	runtimeCoordinator model.RuntimeCoordinator
	store              notificationStore
	parser             notificationParser
}

// NewInteractionController builds a new interaction controller.
func NewInteractionController(runtimeCoordinator model.RuntimeCoordinator) model.InteractionController {
	return NewInteractionControllerWithAdapters(runtimeCoordinator, storageNotificationStore{}, defaultNotificationParser{})
}

// NewInteractionControllerWithAdapters builds a new interaction controller with injected adapters.
func NewInteractionControllerWithAdapters(runtimeCoordinator model.RuntimeCoordinator, store notificationStore, parser notificationParser) model.InteractionController {
	if store == nil {
		store = storageNotificationStore{}
	}
	if parser == nil {
		parser = defaultNotificationParser{}
	}

	return &DefaultInteractionController{
		runtimeCoordinator: runtimeCoordinator,
		store:              store,
		parser:             parser,
	}
}

// SetRuntimeCoordinator updates the runtime coordinator used by the controller.
func (c *DefaultInteractionController) SetRuntimeCoordinator(runtimeCoordinator model.RuntimeCoordinator) {
	c.runtimeCoordinator = runtimeCoordinator
}

// LoadActiveNotifications loads all active notifications from persistent storage.
func (c *DefaultInteractionController) LoadActiveNotifications() ([]domain.Notification, error) {
	items, err := c.loadNotificationValues(true)
	if err != nil {
		return nil, fmt.Errorf("failed to load notifications: %w", err)
	}
	return items, nil
}

// LoadAllNotifications loads all notifications (active and dismissed) from persistent storage.
func (c *DefaultInteractionController) LoadAllNotifications() ([]domain.Notification, error) {
	items, err := c.loadNotificationValues(false)
	if err != nil {
		return nil, fmt.Errorf("failed to load notifications: %w", err)
	}
	return items, nil
}

func (c *DefaultInteractionController) loadNotificationValues(activeOnly bool) ([]domain.Notification, error) {
	if typedStore, ok := c.store.(typedNotificationStore); ok {
		items, err := listTypedNotifications(typedStore, activeOnly)
		if err == nil {
			return notificationPointersToValues(notification.ToDomainSliceUnsafe(items)), nil
		}
		if !errors.Is(err, errTypedNotificationListingUnsupported) {
			return nil, err
		}
	}

	lines, err := listTextNotifications(c.store, activeOnly)
	if err != nil || lines == "" {
		return []domain.Notification{}, err
	}
	return parseNotificationLines(lines, c.parser), nil
}

func listTypedNotifications(store typedNotificationStore, activeOnly bool) ([]notification.Notification, error) {
	if activeOnly {
		return store.ListActiveNotificationValues()
	}
	return store.ListAllNotificationValues()
}

func listTextNotifications(store notificationStore, activeOnly bool) (string, error) {
	if activeOnly {
		return store.ListActiveNotifications()
	}
	return store.ListAllNotifications()
}

func parseNotificationLines(lines string, parser notificationParser) []domain.Notification {
	items := make([]domain.Notification, 0)
	for _, line := range strings.Split(lines, "\n") {
		if line == "" {
			continue
		}
		notif, parseErr := parser.Parse(line)
		if parseErr != nil {
			continue
		}
		items = append(items, notif)
	}
	return items
}

func notificationPointersToValues(notifs []*domain.Notification) []domain.Notification {
	items := make([]domain.Notification, 0, len(notifs))
	for _, notif := range notifs {
		if notif == nil {
			continue
		}
		items = append(items, *notif)
	}
	return items
}

// DismissNotification marks a notification as dismissed.
func (c *DefaultInteractionController) DismissNotification(id string) error {
	return c.store.DismissNotification(id)
}

// DismissByFilter dismisses notifications matching the provided tmux filter scope.
func (c *DefaultInteractionController) DismissByFilter(session, window, pane string) error {
	return c.store.DismissByFilter(session, window, pane)
}

// MarkNotificationRead marks a notification as read.
func (c *DefaultInteractionController) MarkNotificationRead(id string) error {
	return c.store.MarkNotificationRead(id)
}

// MarkNotificationUnread marks a notification as unread.
func (c *DefaultInteractionController) MarkNotificationUnread(id string) error {
	return c.store.MarkNotificationUnread(id)
}

// EnsureTmuxRunning verifies tmux is available.
func (c *DefaultInteractionController) EnsureTmuxRunning() bool {
	if c.runtimeCoordinator == nil {
		return false
	}
	return c.runtimeCoordinator.EnsureTmuxRunning()
}

// JumpToPane performs a tmux jump operation.
func (c *DefaultInteractionController) JumpToPane(sessionID, windowID, paneID string) bool {
	if c.runtimeCoordinator == nil {
		return false
	}
	return c.runtimeCoordinator.JumpToPane(sessionID, windowID, paneID)
}

// JumpToWindow performs a tmux window jump operation.
func (c *DefaultInteractionController) JumpToWindow(sessionID, windowID string) bool {
	if c.runtimeCoordinator == nil {
		return false
	}
	return c.runtimeCoordinator.JumpToWindow(sessionID, windowID)
}
