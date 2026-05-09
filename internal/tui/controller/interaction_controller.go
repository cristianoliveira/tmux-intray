// Package controller provides side-effect orchestration for TUI interactions.
package controller

import (
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/domain"
	"github.com/cristianoliveira/tmux-intray/internal/storage"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

type notificationStore interface {
	ListActiveDomainNotifications() ([]domain.Notification, error)
	ListAllDomainNotifications() ([]domain.Notification, error)
	DismissNotification(id string) error
	DismissByFilter(session, window, pane string) error
	MarkNotificationRead(id string) error
	MarkNotificationUnread(id string) error
}

type storageNotificationStore struct{}

func (s storageNotificationStore) ListActiveDomainNotifications() ([]domain.Notification, error) {
	return storage.ListNotificationValues("active", "", "", "", "", "", "", "")
}

func (s storageNotificationStore) ListAllDomainNotifications() ([]domain.Notification, error) {
	return storage.ListNotificationValues("", "", "", "", "", "", "", "")
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

// DefaultInteractionController is the production controller implementation.
type DefaultInteractionController struct {
	runtimeCoordinator model.RuntimeCoordinator
	store              notificationStore
}

// NewInteractionController builds a new interaction controller.
func NewInteractionController(runtimeCoordinator model.RuntimeCoordinator) model.InteractionController {
	return NewInteractionControllerWithAdapters(runtimeCoordinator, storageNotificationStore{})
}

// NewInteractionControllerWithAdapters builds a new interaction controller with injected adapters.
func NewInteractionControllerWithAdapters(runtimeCoordinator model.RuntimeCoordinator, store notificationStore) model.InteractionController {
	if store == nil {
		store = storageNotificationStore{}
	}

	return &DefaultInteractionController{
		runtimeCoordinator: runtimeCoordinator,
		store:              store,
	}
}

// SetRuntimeCoordinator updates the runtime coordinator used by the controller.
func (c *DefaultInteractionController) SetRuntimeCoordinator(runtimeCoordinator model.RuntimeCoordinator) {
	c.runtimeCoordinator = runtimeCoordinator
}

// LoadActiveNotifications loads all active notifications from persistent storage.
func (c *DefaultInteractionController) LoadActiveNotifications() ([]domain.Notification, error) {
	notifs, err := c.store.ListActiveDomainNotifications()
	if err != nil {
		return nil, fmt.Errorf("failed to load notifications: %w", err)
	}
	return notifs, nil
}

// LoadAllNotifications loads all notifications (active and dismissed) from persistent storage.
func (c *DefaultInteractionController) LoadAllNotifications() ([]domain.Notification, error) {
	notifs, err := c.store.ListAllDomainNotifications()
	if err != nil {
		return nil, fmt.Errorf("failed to load notifications: %w", err)
	}
	return notifs, nil
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
