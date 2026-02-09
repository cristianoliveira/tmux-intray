package state

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
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
		tree := BuildTree(notifications, settings.GroupByPane)

		scenarios := []struct {
			name    string
			expands map[NodeKind]bool
		}{
			{
				name: "all-expanded",
				expands: map[NodeKind]bool{
					NodeKindSession: true,
					NodeKindWindow:  true,
					NodeKindPane:    true,
				},
			},
			{
				name: "session-collapsed",
				expands: map[NodeKind]bool{
					NodeKindSession: false,
					NodeKindWindow:  true,
					NodeKindPane:    true,
				},
			},
			{
				name: "window-collapsed",
				expands: map[NodeKind]bool{
					NodeKindSession: true,
					NodeKindWindow:  false,
					NodeKindPane:    true,
				},
			},
		}

		for _, scenario := range scenarios {
			setExpansionState(tree, scenario.expands)
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
		model.cursor = len(model.visibleNodes) / 2

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
				model.searchQuery = query.query

				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					model.applySearchFilter(false)
				}
			})
		}
	}
}

func benchmarkModel(notifications []notification.Notification) *Model {
	model := &Model{
		viewMode:       viewModeGrouped,
		groupBy:        settings.GroupByPane,
		notifications:  notifications,
		expansionState: map[string]bool{},
		viewport:       viewport.New(120, 40),
		width:          120,
	}

	model.applySearchFilter(false)
	setExpansionState(model.treeRoot, map[NodeKind]bool{
		NodeKindSession: true,
		NodeKindWindow:  true,
		NodeKindPane:    true,
	})
	model.invalidateCache()
	model.visibleNodes = model.computeVisibleNodes()
	return model
}

func setExpansionState(root *Node, expanded map[NodeKind]bool) {
	var walk func(node *Node)
	walk = func(node *Node) {
		if node == nil {
			return
		}

		if node.Kind == NodeKindSession || node.Kind == NodeKindWindow || node.Kind == NodeKindPane {
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
