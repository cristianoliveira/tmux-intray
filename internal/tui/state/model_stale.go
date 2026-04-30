package state

// SetShowStale controls whether notifications for stale tmux targets remain visible.
func (m *Model) SetShowStale(show bool) {
	m.showStale = show
	if svc, ok := m.notificationService.(interface{ SetShowStale(bool) }); ok {
		svc.SetShowStale(show)
	}
}
