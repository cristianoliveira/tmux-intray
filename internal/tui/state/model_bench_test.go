package state

import (
	"fmt"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
	"github.com/cristianoliveira/tmux-intray/internal/tui/service"
)

func BenchmarkBuildTree(b *testing.B) {
	for _, size := range []int{1000, 5000, 10000} {
		notifications := benchmarkNotifications(size)

		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = BuildTree(notifications, settings.GroupByPane)
			}
		})
	}
}

func BenchmarkComputeVisibleNodes(b *testing.B) {
	for _, size := range []int{1000, 5000, 10000} {
		notifications := benchmarkNotifications(size)
		tree := benchmarkBuildTree(notifications, settings.GroupByPane)

		scenarios := []struct {
			name    string
			expands map[model.NodeKind]bool
		}{
			{
				name: "all-expanded",
				expands: map[model.NodeKind]bool{
					model.NodeKindSession: true,
					model.NodeKindWindow:  true,
					model.NodeKindPane:    true,
				},
			},
			{
				name: "session-collapsed",
				expands: map[model.NodeKind]bool{
					model.NodeKindSession: false,
					model.NodeKindWindow:  true,
					model.NodeKindPane:    true,
				},
			},
			{
				name: "window-collapsed",
				expands: map[model.NodeKind]bool{
					model.NodeKindSession: true,
					model.NodeKindWindow:  false,
					model.NodeKindPane:    true,
				},
			},
		}

		for _, scenario := range scenarios {
			benchmarkSetExpansionState(tree, scenario.expands)
			benchModel := benchmarkModel(notifications)
			benchModel.treeService = &dummyTreeService{treeRoot: tree}

			name := fmt.Sprintf("n=%d/%s", size, scenario.name)
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					benchModel.invalidateCache()
					_ = benchModel.computeVisibleNodes()
				}
			})
		}
	}
}

func BenchmarkUpdateViewportContentGrouped(b *testing.B) {
	for _, size := range []int{1000, 5000, 10000} {
		notifications := benchmarkNotifications(size)
		benchModel := benchmarkModel(notifications)
		visibleNodes := benchModel.computeVisibleNodes()
		benchModel.uiState.SetCursor(len(visibleNodes) / 2)

		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchModel.updateViewportContent()
			}
		})
	}
}

func BenchmarkApplySearchFilterGrouped(b *testing.B) {
	queries := []struct {
		name  string
		query string
	}{
		{name: "match-all", query: ""},
		{name: "filtered", query: "error session-03"},
	}

	for _, size := range []int{1000, 5000, 10000} {
		notifications := benchmarkNotifications(size)

		for _, query := range queries {
			name := fmt.Sprintf("n=%d/%s", size, query.name)
			b.Run(name, func(b *testing.B) {
				model := benchmarkModel(notifications)
				model.uiState.SetSearchQuery(query.query)

				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					model.applySearchFilter()
					model.resetCursor()
				}
			})
		}
	}
}

func benchmarkModel(notifications []notification.Notification) *Model {
	notificationService := service.NewNotificationService(nil, nil)
	notificationService.SetNotifications(notifications)

	dummySvc := &dummyNotificationService{}
	dummySvc.SetNotifications(notifications)

	m := &Model{
		uiState:             NewUIState(),
		notifications:       notifications,
		treeService:         &dummyTreeService{},
		runtimeCoordinator:  &dummyRuntimeCoordinator{},
		notificationService: dummySvc,
	}
	m.syncNotificationMirrors()
	m.uiState.SetWidth(120)
	m.uiState.SetHeight(40)
	m.uiState.UpdateViewportSize()

	m.filtered = m.notifications
	_ = m.treeService.BuildTree(m.filtered, string(m.uiState.GetGroupBy()))
	benchmarkSetExpansionState(m.getTreeRootForTest(), map[model.NodeKind]bool{
		model.NodeKindSession: true,
		model.NodeKindWindow:  true,
		model.NodeKindPane:    true,
	})
	m.invalidateCache()
	_ = m.computeVisibleNodes()
	return m
}

// benchmarkBuildTree builds a tree for benchmarking using TreeService.
func benchmarkBuildTree(notifications []notification.Notification, groupBy string) *model.TreeNode {
	// For benchmarking, use a simple service without full initialization
	treeService := &dummyTreeService{}
	_ = treeService.BuildTree(notifications, groupBy)
	return treeService.GetTreeRoot()
}

// benchmarkSetExpansionState sets expansion state for benchmarking.
func benchmarkSetExpansionState(root *model.TreeNode, expanded map[model.NodeKind]bool) {
	var walk func(node *model.TreeNode)
	walk = func(node *model.TreeNode) {
		if node == nil {
			return
		}

		if node.Kind == model.NodeKindSession || node.Kind == model.NodeKindWindow || node.Kind == model.NodeKindPane {
			value, ok := expanded[node.Kind]
			if !ok {
				value = true
			}
			node.Expanded = value
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(root)
}

type dummyTreeService struct {
	treeRoot          *model.TreeNode
	visibleNodesCache []*model.TreeNode
	cacheValid        bool
}

func (s *dummyTreeService) BuildTree(notifications []notification.Notification, groupBy string) error {
	// Minimal implementation - convert from state.BuildTree result
	stateTree := BuildTree(notifications, groupBy)
	if stateTree == nil {
		s.treeRoot = nil
		s.InvalidateCache()
		return nil
	}
	s.treeRoot = s.convertNode(stateTree)
	s.InvalidateCache()
	return nil
}

func (s *dummyTreeService) RebuildTreeForFilter(notifications []notification.Notification, groupBy string, expansionState map[string]bool) error {
	if err := s.BuildTree(notifications, groupBy); err != nil {
		return err
	}
	return nil
}

func (s *dummyTreeService) ClearTree() {
	s.treeRoot = nil
	s.InvalidateCache()
}

func (s *dummyTreeService) GetTreeRoot() *model.TreeNode {
	return s.treeRoot
}

func (s *dummyTreeService) convertNode(stateNode *Node) *model.TreeNode {
	if stateNode == nil {
		return nil
	}

	modelNode := &model.TreeNode{
		Kind:          model.NodeKind(stateNode.Kind),
		Title:         stateNode.Title,
		Display:       stateNode.Display,
		Expanded:      stateNode.Expanded,
		Notification:  stateNode.Notification,
		Count:         stateNode.Count,
		UnreadCount:   stateNode.UnreadCount,
		LatestEvent:   stateNode.LatestEvent,
		EarliestEvent: stateNode.EarliestEvent,
	}

	if len(stateNode.LevelCounts) > 0 {
		modelNode.LevelCounts = make(map[string]int, len(stateNode.LevelCounts))
		for level, count := range stateNode.LevelCounts {
			modelNode.LevelCounts[level] = count
		}
	}
	if len(stateNode.Sources) > 0 {
		modelNode.Sources = make(map[string]model.NotificationSource, len(stateNode.Sources))
		for key, src := range stateNode.Sources {
			modelNode.Sources[key] = src
		}
	}

	for _, child := range stateNode.Children {
		modelNode.Children = append(modelNode.Children, s.convertNode(child))
	}

	return modelNode
}

// Other required TreeService methods (not used in benchmarking)
func (s *dummyTreeService) FindNotificationPath(root *model.TreeNode, notif notification.Notification) ([]*model.TreeNode, error) {
	return nil, nil
}
func (s *dummyTreeService) FindNodeByID(root *model.TreeNode, identifier string) *model.TreeNode {
	return nil
}
func (s *dummyTreeService) GetVisibleNodes() []*model.TreeNode {
	if s.cacheValid {
		return s.visibleNodesCache
	}
	if s.treeRoot == nil {
		s.visibleNodesCache = nil
		s.cacheValid = true
		return nil
	}

	var visible []*model.TreeNode
	var walk func(node *model.TreeNode)
	walk = func(node *model.TreeNode) {
		if node == nil {
			return
		}
		if node.Kind != model.NodeKindRoot {
			visible = append(visible, node)
		}
		if node.Kind == model.NodeKindNotification {
			return
		}
		if node.Kind != model.NodeKindRoot && !node.Expanded {
			return
		}
		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(s.treeRoot)
	s.visibleNodesCache = visible
	s.cacheValid = true
	return s.visibleNodesCache
}
func (s *dummyTreeService) InvalidateCache() {
	s.visibleNodesCache = nil
	s.cacheValid = false
}
func (s *dummyTreeService) GetNodeIdentifier(node *model.TreeNode) string {
	return ""
}
func (s *dummyTreeService) PruneEmptyGroups() {
}
func (s *dummyTreeService) ApplyExpansionState(expansionState map[string]bool) {
}
func (s *dummyTreeService) ExpandNode(node *model.TreeNode)          {}
func (s *dummyTreeService) CollapseNode(node *model.TreeNode)        {}
func (s *dummyTreeService) ToggleNodeExpansion(node *model.TreeNode) {}
func (s *dummyTreeService) GetTreeLevel(node *model.TreeNode) int {
	return 0
}

func benchmarkNotifications(size int) []notification.Notification {
	notifications := make([]notification.Notification, size)
	for i := 0; i < size; i++ {
		notifications[i] = notification.Notification{
			ID:        i + 1,
			Message:   fmt.Sprintf("%s session-%02d event-%d", benchmarkLevel(i), i%20, i),
			Timestamp: fmt.Sprintf("2024-01-%02dT%02d:%02d:%02dZ", (i%28)+1, i%24, i%60, i%60),
			Session:   fmt.Sprintf("$%02d", i%20),
			Window:    fmt.Sprintf("@%02d", i%10),
			Pane:      fmt.Sprintf("%%%02d", i%5),
			Level:     benchmarkLevel(i),
			State:     "active",
		}
	}

	return notifications
}

func benchmarkLevel(index int) string {
	switch index % 3 {
	case 0:
		return "error"
	case 1:
		return "warning"
	default:
		return "info"
	}
}

type dummyRuntimeCoordinator struct{}

func (d *dummyRuntimeCoordinator) ResolveSessionName(sessionID string) string {
	return sessionID
}

func (d *dummyRuntimeCoordinator) ResolveWindowName(windowID string) string {
	return windowID
}

func (d *dummyRuntimeCoordinator) ResolvePaneName(paneID string) string {
	return paneID
}

func (d *dummyRuntimeCoordinator) GetSessionNames() map[string]string {
	return nil
}

func (d *dummyRuntimeCoordinator) GetWindowNames() map[string]string {
	return nil
}

func (d *dummyRuntimeCoordinator) GetPaneNames() map[string]string {
	return nil
}

func (d *dummyRuntimeCoordinator) SetSessionNames(names map[string]string) {}

func (d *dummyRuntimeCoordinator) SetWindowNames(names map[string]string) {}

func (d *dummyRuntimeCoordinator) SetPaneNames(names map[string]string) {}

func (d *dummyRuntimeCoordinator) EnsureTmuxRunning() bool {
	return true
}

func (d *dummyRuntimeCoordinator) JumpToPane(sessionID, windowID, paneID string) bool {
	return true
}

func (d *dummyRuntimeCoordinator) JumpToWindow(sessionID, windowID string) bool {
	return true
}

func (d *dummyRuntimeCoordinator) ValidatePaneExists(sessionID, windowID, paneID string) (bool, error) {
	return true, nil
}

func (d *dummyRuntimeCoordinator) GetCurrentContext() (*model.TmuxContext, error) {
	return nil, nil
}

func (d *dummyRuntimeCoordinator) ListSessions() (map[string]string, error) {
	return nil, nil
}

func (d *dummyRuntimeCoordinator) ListWindows() (map[string]string, error) {
	return nil, nil
}

func (d *dummyRuntimeCoordinator) ListPanes() (map[string]string, error) {
	return nil, nil
}

func (d *dummyRuntimeCoordinator) GetSessionName(sessionID string) (string, error) {
	return sessionID, nil
}

func (d *dummyRuntimeCoordinator) GetWindowName(windowID string) (string, error) {
	return windowID, nil
}

func (d *dummyRuntimeCoordinator) GetPaneName(paneID string) (string, error) {
	return paneID, nil
}

func (d *dummyRuntimeCoordinator) RefreshNames() error {
	return nil
}

func (d *dummyRuntimeCoordinator) GetTmuxVisibility() (bool, error) {
	return true, nil
}

func (d *dummyRuntimeCoordinator) SetTmuxVisibility(visible bool) error {
	return nil
}

type dummyNotificationService struct {
	notifications []notification.Notification
	filtered      []notification.Notification
}

func (d *dummyNotificationService) FilterNotifications(notifications []notification.Notification, query string) []notification.Notification {
	if query == "" {
		return notifications
	}
	var result []notification.Notification
	for _, n := range notifications {
		if contains(n.Message, query) {
			result = append(result, n)
		}
	}
	return result
}

func (d *dummyNotificationService) FilterByState(notifications []notification.Notification, state string) []notification.Notification {
	var result []notification.Notification
	for _, n := range notifications {
		if n.State == state {
			result = append(result, n)
		}
	}
	return result
}

func (d *dummyNotificationService) FilterByLevel(notifications []notification.Notification, level string) []notification.Notification {
	var result []notification.Notification
	for _, n := range notifications {
		if n.Level == level {
			result = append(result, n)
		}
	}
	return result
}

func (d *dummyNotificationService) FilterBySession(notifications []notification.Notification, sessionID string) []notification.Notification {
	var result []notification.Notification
	for _, n := range notifications {
		if n.Session == sessionID {
			result = append(result, n)
		}
	}
	return result
}

func (d *dummyNotificationService) FilterByWindow(notifications []notification.Notification, windowID string) []notification.Notification {
	var result []notification.Notification
	for _, n := range notifications {
		if n.Window == windowID {
			result = append(result, n)
		}
	}
	return result
}

func (d *dummyNotificationService) FilterByPane(notifications []notification.Notification, paneID string) []notification.Notification {
	var result []notification.Notification
	for _, n := range notifications {
		if n.Pane == paneID {
			result = append(result, n)
		}
	}
	return result
}

func (d *dummyNotificationService) SortNotifications(notifications []notification.Notification, sortBy, sortOrder string) []notification.Notification {
	return notifications
}

func (d *dummyNotificationService) GetUnreadCount(notifications []notification.Notification) int {
	count := 0
	for _, n := range notifications {
		if !n.IsRead() {
			count++
		}
	}
	return count
}

func (d *dummyNotificationService) GetReadCount(notifications []notification.Notification) int {
	count := 0
	for _, n := range notifications {
		if n.IsRead() {
			count++
		}
	}
	return count
}

func (d *dummyNotificationService) GetCountsByLevel(notifications []notification.Notification) map[string]int {
	counts := map[string]int{}
	for _, n := range notifications {
		counts[n.Level]++
	}
	return counts
}

func (d *dummyNotificationService) Search(notifications []notification.Notification, query string, caseSensitive bool) []notification.Notification {
	return d.FilterNotifications(notifications, query)
}

func (d *dummyNotificationService) SetNotifications(notifications []notification.Notification) {
	d.notifications = notifications
	d.filtered = notifications
}

func (d *dummyNotificationService) GetNotifications() []notification.Notification {
	return d.notifications
}

func (d *dummyNotificationService) GetFilteredNotifications() []notification.Notification {
	return d.filtered
}

func (d *dummyNotificationService) FilterByReadStatus(notifications []notification.Notification, readFilter string) []notification.Notification {
	if readFilter == "" {
		return notifications
	}
	var filtered []notification.Notification
	for _, n := range notifications {
		isRead := n.IsRead()
		if readFilter == settings.ReadFilterUnread && !isRead {
			filtered = append(filtered, n)
		}
		if readFilter == settings.ReadFilterRead && isRead {
			filtered = append(filtered, n)
		}
	}
	return filtered
}

func (d *dummyNotificationService) ApplyFiltersAndSearch(query, state, level, sessionID, windowID, paneID, readFilter, sortBy, sortOrder string) {
	result := d.notifications

	if state != "" {
		result = d.FilterByState(result, state)
	}

	if level != "" {
		result = d.FilterByLevel(result, level)
	}

	if sessionID != "" {
		result = d.FilterBySession(result, sessionID)
	}

	if windowID != "" {
		result = d.FilterByWindow(result, windowID)
	}

	if paneID != "" {
		result = d.FilterByPane(result, paneID)
	}

	if readFilter != "" {
		result = d.FilterByReadStatus(result, readFilter)
	}

	if query != "" {
		result = d.FilterNotifications(result, query)
	}

	result = d.SortNotifications(result, sortBy, sortOrder)

	d.filtered = result
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || len(s) > 0 && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
