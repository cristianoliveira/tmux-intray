package state

import (
	"fmt"
	"testing"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
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
			model := benchmarkModel(notifications)
			model.treeRoot = tree

			name := fmt.Sprintf("n=%d/%s", size, scenario.name)
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					model.invalidateCache()
					_ = model.computeVisibleNodes()
				}
			})
		}
	}
}

func BenchmarkUpdateViewportContentGrouped(b *testing.B) {
	for _, size := range []int{1000, 5000, 10000} {
		notifications := benchmarkNotifications(size)
		model := benchmarkModel(notifications)
		model.visibleNodes = model.computeVisibleNodes()
		model.uiState.SetCursor(len(model.visibleNodes) / 2)

		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				model.updateViewportContent()
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
	m := &Model{
		uiState:       NewUIState(),
		notifications: notifications,
	}
	m.uiState.SetWidth(120)
	m.uiState.SetHeight(40)
	m.uiState.UpdateViewportSize()

	m.applySearchFilter()
	m.resetCursor()
	benchmarkSetExpansionState(m.treeRoot, map[model.NodeKind]bool{
		model.NodeKindSession: true,
		model.NodeKindWindow:  true,
		model.NodeKindPane:    true,
	})
	m.invalidateCache()
	m.visibleNodes = m.computeVisibleNodes()
	return m
}

// benchmarkBuildTree builds a tree for benchmarking using TreeService.
func benchmarkBuildTree(notifications []notification.Notification, groupBy string) *model.TreeNode {
	// For benchmarking, use a simple service without full initialization
	treeService := &dummyTreeService{}
	tree, _ := treeService.BuildTree(notifications, groupBy)
	return tree
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

// dummyTreeService is a minimal tree service for benchmarking.
type dummyTreeService struct{}

func (s *dummyTreeService) BuildTree(notifications []notification.Notification, groupBy string) (*model.TreeNode, error) {
	// Minimal implementation - convert from state.BuildTree result
	stateTree := BuildTree(notifications, groupBy)
	if stateTree == nil {
		return nil, nil
	}
	return s.convertNode(stateTree), nil
}

func (s *dummyTreeService) convertNode(stateNode *Node) *model.TreeNode {
	if stateNode == nil {
		return nil
	}

	modelNode := &model.TreeNode{
		Kind:         model.NodeKind(stateNode.Kind),
		Title:        stateNode.Title,
		Display:      stateNode.Display,
		Expanded:     stateNode.Expanded,
		Notification: stateNode.Notification,
		Count:        stateNode.Count,
		UnreadCount:  stateNode.UnreadCount,
		LatestEvent:  stateNode.LatestEvent,
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
func (s *dummyTreeService) GetVisibleNodes(root *model.TreeNode) []*model.TreeNode {
	return nil
}
func (s *dummyTreeService) GetNodeIdentifier(node *model.TreeNode) string {
	return ""
}
func (s *dummyTreeService) PruneEmptyGroups(root *model.TreeNode) *model.TreeNode {
	return root
}
func (s *dummyTreeService) ApplyExpansionState(root *model.TreeNode, expansionState map[string]bool) {
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
