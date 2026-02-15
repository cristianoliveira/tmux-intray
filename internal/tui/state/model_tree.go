package state

import (
	"fmt"
	"strings"

	"github.com/cristianoliveira/tmux-intray/internal/notification"
	"github.com/cristianoliveira/tmux-intray/internal/settings"
	"github.com/cristianoliveira/tmux-intray/internal/tui/model"
)

func (m *Model) isGroupedView() bool {
	return m.uiState.IsGroupedView()
}

// IsGroupedView is the public version of isGroupedView.
func (m *Model) IsGroupedView() bool {
	return m.isGroupedView()
}

// cycleViewMode cycles through available view modes (compact → detailed → grouped).
func (m *Model) cycleViewMode() {
	m.uiState.CycleViewMode()
	m.applySearchFilter()
	m.resetCursor()

	if err := m.saveSettings(); err != nil {
		m.errorHandler.Warning(fmt.Sprintf("Failed to save settings: %v", err))
	}
}

func (m *Model) computeVisibleNodes() []*model.TreeNode {
	return m.treeService.GetVisibleNodes()
}

func (m *Model) invalidateCache() {
	m.ensureTreeService().InvalidateCache()
}

func isGroupNode(node *model.TreeNode) bool {
	if node == nil {
		return false
	}
	return node.Kind != model.NodeKindNotification && node.Kind != model.NodeKindRoot
}

// isGroupNode checks if a model.TreeNode is a group node.
func (m *Model) isGroupNode(node *model.TreeNode) bool {
	return node.Kind != model.NodeKindNotification && node.Kind != model.NodeKindRoot
}

func getTreeLevel(node *model.TreeNode) int {
	if node == nil {
		return 0
	}
	switch node.Kind {
	case model.NodeKindSession:
		return 0
	case model.NodeKindWindow:
		return 1
	case model.NodeKindPane:
		return 2
	case model.NodeKindMessage:
		return 3
	default:
		return 0
	}
}
func (m *Model) currentListLen() int {
	if m.isGroupedView() {
		return len(m.treeService.GetVisibleNodes())
	}
	return len(m.filtered)
}

func (m *Model) selectedVisibleNode() *model.TreeNode {
	if !m.isGroupedView() {
		return nil
	}
	cursor := m.uiState.GetCursor()
	visibleNodes := m.treeService.GetVisibleNodes()
	if cursor < 0 || cursor >= len(visibleNodes) {
		return nil
	}
	return visibleNodes[cursor]
}

func (m *Model) toggleNodeExpansion() bool {
	node := m.selectedVisibleNode()
	if node == nil || node.Kind == model.NodeKindNotification {
		return false
	}
	if node.Expanded {
		m.treeService.CollapseNode(node)
	} else {
		m.treeService.ExpandNode(node)
	}
	m.invalidateCache()
	return true
}

func (m *Model) toggleFold() {
	if !m.isGroupedView() {
		return
	}
	node := m.selectedVisibleNode()
	if node == nil || node.Kind == model.NodeKindNotification {
		return
	}
	if m.allGroupsCollapsed() {
		m.applyDefaultExpansion()
		return
	}
	if node.Expanded {
		m.treeService.CollapseNode(node)
		m.invalidateCache()
		m.updateViewportContent()
		return
	}
	m.treeService.ExpandNode(node)
	m.invalidateCache()
	m.updateViewportContent()
}

func (m *Model) allGroupsCollapsed() bool {
	treeRoot := m.treeService.GetTreeRoot()
	if treeRoot == nil {
		return false
	}
	collapsed := true
	seen := false
	var walk func(node *model.TreeNode)
	walk = func(node *model.TreeNode) {
		if node == nil || !collapsed {
			return
		}
		if m.isGroupNode(node) {
			seen = true
			if node.Expanded {
				collapsed = false
				return
			}
		}
		for _, child := range node.Children {
			walk(child)
			if !collapsed {
				return
			}
		}
	}
	walk(treeRoot)
	return seen && collapsed
}

func (m *Model) applyDefaultExpansion() {
	treeRoot := m.treeService.GetTreeRoot()
	if treeRoot == nil {
		return
	}

	// Save selected node identifier before modifying tree
	selectedID := ""
	if selected := m.selectedVisibleNode(); selected != nil {
		selectedID = m.treeService.GetNodeIdentifier(selected)
	}

	level := m.uiState.GetExpandLevel()
	if level < settings.MinExpandLevel {
		level = settings.MinExpandLevel
	}
	if level > settings.MaxExpandLevel {
		level = settings.MaxExpandLevel
	}

	var walk func(node *model.TreeNode)
	walk = func(node *model.TreeNode) {
		if node == nil {
			return
		}
		if m.isGroupNode(node) {
			nodeLevel := m.treeService.GetTreeLevel(node) + 1
			expanded := nodeLevel <= level
			node.Expanded = expanded
			m.updateExpansionState(node, expanded)
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(treeRoot)

	m.invalidateCache()

	// Restore cursor to the selected node using identifier
	if selectedID != "" {
		m.restoreCursor(selectedID)
	}

	// Ensure cursor is within bounds
	visibleNodes := m.treeService.GetVisibleNodes()
	if m.uiState.GetCursor() >= len(visibleNodes) {
		m.uiState.SetCursor(len(visibleNodes) - 1)
	}
	if m.uiState.GetCursor() < 0 {
		m.uiState.SetCursor(0)
	}
	m.updateViewportContent()
	m.ensureCursorVisible()
}

// ApplyDefaultExpansion is the public version of applyDefaultExpansion.
func (m *Model) ApplyDefaultExpansion() {
	m.applyDefaultExpansion()
}

// GetViewMode returns the current view mode.
func (m *Model) GetViewMode() string {
	return string(m.uiState.GetViewMode())
}

// ToggleViewMode toggles between view modes.
func (m *Model) ToggleViewMode() error {
	m.cycleViewMode()
	return nil
}

func (m *Model) expandNode(node *model.TreeNode) {
	if !m.isGroupedView() {
		return
	}
	if node == nil || node.Kind == model.NodeKindNotification {
		return
	}
	if node.Expanded {
		return
	}

	// Save node identifier before modifying tree to avoid using stale references
	nodeID := m.treeService.GetNodeIdentifier(node)

	m.treeService.ExpandNode(node)
	m.updateExpansionState(node, true)

	// Restore cursor to the same node using identifier
	m.restoreCursor(nodeID)

	m.updateViewportContent()
	m.ensureCursorVisible()
}

func (m *Model) collapseNode(node *model.TreeNode) {
	if !m.isGroupedView() {
		return
	}
	if node == nil || node.Kind == model.NodeKindNotification {
		return
	}
	if !node.Expanded {
		return
	}

	// Save node identifiers before modifying tree to avoid using stale references
	selectedID := ""
	if selected := m.selectedVisibleNode(); selected != nil {
		selectedID = m.treeService.GetNodeIdentifier(selected)
	}
	nodeID := m.treeService.GetNodeIdentifier(node)

	m.treeService.CollapseNode(node)
	m.updateExpansionState(node, false)
	visibleNodes := m.treeService.GetVisibleNodes()

	// If selected node was inside the collapsed node, move cursor to the collapsed node
	m.moveCursorToCollapsedNodeIfNeeded(selectedID, nodeID, visibleNodes)

	// Ensure cursor is within bounds
	m.clampCursorToBounds(len(visibleNodes))
	m.updateViewportContent()
	m.ensureCursorVisible()
}

// moveCursorToCollapsedNodeIfNeeded moves cursor to collapsed node if selected was inside it.
func (m *Model) moveCursorToCollapsedNodeIfNeeded(selectedID, nodeID string, visibleNodes []*model.TreeNode) {
	if selectedID == "" {
		return
	}

	treeRoot := m.treeService.GetTreeRoot()
	selectedNode := m.treeService.FindNodeByID(treeRoot, selectedID)
	if selectedNode == nil {
		return
	}

	collapsedNode := m.treeService.FindNodeByID(treeRoot, nodeID)
	if collapsedNode == nil {
		return
	}

	if !m.nodeContains(collapsedNode, selectedNode) {
		return
	}

	// Move cursor to the collapsed node
	if index := indexOfTreeNode(visibleNodes, collapsedNode); index >= 0 {
		m.uiState.SetCursor(index)
	}
}

// clampCursorToBounds ensures cursor is within valid bounds.
func (m *Model) clampCursorToBounds(listLen int) {
	if m.uiState.GetCursor() >= listLen {
		m.uiState.SetCursor(listLen - 1)
	}
	if m.uiState.GetCursor() < 0 {
		m.uiState.SetCursor(0)
	}
}

// nodeContains checks if targetNode is contained within root node.
func (m *Model) nodeContains(root, target *model.TreeNode) bool {
	if root == nil || target == nil {
		return false
	}
	if root == target {
		return true
	}
	for _, child := range root.Children {
		if m.nodeContains(child, target) {
			return true
		}
	}
	return false
}

// indexOfTreeNode finds the index of a target node in a slice.
func indexOfTreeNode(nodes []*model.TreeNode, target *model.TreeNode) int {
	for i, node := range nodes {
		if node == target {
			return i
		}
	}
	return -1
}

func (m *Model) updateExpansionState(node *model.TreeNode, expanded bool) {
	key := m.nodeExpansionKey(node)
	if key == "" {
		return
	}
	expansionState := m.uiState.GetExpansionState()
	if expansionState == nil {
		expansionState = map[string]bool{}
		m.uiState.SetExpansionState(expansionState)
	}
	legacyKey := m.nodeExpansionLegacyKey(node)
	if legacyKey != "" && legacyKey != key {
		delete(expansionState, legacyKey)
	}
	m.uiState.UpdateExpansionState(key, expanded)
}

func (m *Model) nodeExpansionKey(node *model.TreeNode) string {
	if node == nil || node.Kind == model.NodeKindNotification || node.Kind == model.NodeKindRoot {
		return ""
	}
	return m.ensureTreeService().GetNodeIdentifier(node)
}

func (m *Model) nodeExpansionLegacyKey(node *model.TreeNode) string {
	if node == nil || node.Kind == model.NodeKindNotification || node.Kind == model.NodeKindRoot {
		return ""
	}
	switch node.Kind {
	case model.NodeKindSession:
		return serializeLegacyNodeExpansionPath(model.NodeKindSession, node.Title)
	case model.NodeKindWindow:
		return serializeLegacyNodeExpansionPath(model.NodeKindWindow, node.Title)
	case model.NodeKindPane:
		return serializeLegacyNodeExpansionPath(model.NodeKindPane, node.Title)
	case model.NodeKindMessage:
		return serializeLegacyNodeExpansionPath(model.NodeKindMessage, node.Title)
	default:
		return ""
	}
}

func serializeNodeExpansionPath(kind model.NodeKind, parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	encoded := make([]string, 0, len(parts))
	for _, part := range parts {
		encoded = append(encoded, escapeExpansionPathSegment(part))
	}
	return fmt.Sprintf("%s:%s", kind, strings.Join(encoded, ":"))
}

func escapeExpansionPathSegment(value string) string {
	replacer := strings.NewReplacer(
		"%", "%25",
		":", "%3A",
	)
	return replacer.Replace(value)
}

func serializeLegacyNodeExpansionPath(kind model.NodeKind, parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%s", kind, strings.Join(parts, ":"))
}

func (m *Model) nodePathSegments(path []*model.TreeNode) (session string, window string, pane string) {
	for _, current := range path {
		switch current.Kind {
		case model.NodeKindSession:
			session = current.Title
		case model.NodeKindWindow:
			window = current.Title
		case model.NodeKindPane:
			pane = current.Title
		}
	}
	return session, window, pane
}

// findNodePath returns the path from root to target node (inclusive).
// Returns nil if target not found.
func (m *Model) findNodePath(root, target *model.TreeNode) []*model.TreeNode {
	if root == nil || target == nil {
		return nil
	}
	if root == target {
		return []*model.TreeNode{root}
	}
	for _, child := range root.Children {
		if path := m.findNodePath(child, target); path != nil {
			return append([]*model.TreeNode{root}, path...)
		}
	}
	return nil
}

// getAncestorTitles returns session, window, pane titles for a node by finding its path from root.
func (m *Model) getAncestorTitles(node *model.TreeNode) (session, window, pane string, ok bool) {
	treeRoot := m.treeService.GetTreeRoot()
	if treeRoot == nil {
		return "", "", "", false
	}
	path := m.findNodePath(treeRoot, node)
	if path == nil {
		return "", "", "", false
	}
	session, window, pane = m.nodePathSegments(path)
	return session, window, pane, true
}

// buildFilteredTree builds a tree from filtered notifications and applies saved expansion state.
// Returns a tree where group counts reflect only matching notifications.
func (m *Model) buildFilteredTree(notifications []notification.Notification) *model.TreeNode {
	m.invalidateCache()

	if len(notifications) == 0 {
		m.treeService.ClearTree()
		return nil
	}

	// Use TreeService to build the tree
	err := m.treeService.BuildTree(notifications, string(m.uiState.GetGroupBy()))
	if err != nil {
		m.treeService.ClearTree()
		return nil
	}

	// Prune empty groups (groups with no matching notifications)
	m.treeService.PruneEmptyGroups()

	// Apply saved expansion state where possible
	expansionState := m.uiState.GetExpansionState()
	if expansionState != nil {
		m.treeService.ApplyExpansionState(expansionState)
	} else {
		// If no saved state, expand all by default
		m.expandTreeRecursive(m.treeService.GetTreeRoot())
	}
	return m.treeService.GetTreeRoot()
}

// expandTreeRecursive is a helper that expands all group nodes.
func (m *Model) expandTreeRecursive(node *model.TreeNode) {
	if node == nil {
		return
	}
	if node.Kind != model.NodeKindNotification {
		node.Expanded = true
	}
	for _, child := range node.Children {
		m.expandTreeRecursive(child)
	}
}

// pruneEmptyGroups removes groups from the tree that have no children or count of 0.
// This ensures that empty groups created by filtering don't appear in the UI.
func (m *Model) pruneEmptyGroups(node *Node) {
	if node == nil {
		return
	}

	// Recursively prune children first
	var filteredChildren []*Node
	for _, child := range node.Children {
		m.pruneEmptyGroups(child)
		// Keep the child if it has children (even if it's a leaf with notifications)
		// or if it's a notification node
		if len(child.Children) > 0 || child.Kind == NodeKindNotification {
			filteredChildren = append(filteredChildren, child)
		}
	}
	node.Children = filteredChildren
}

// applyExpansionState applies the saved expansion state to the tree nodes.
// Only applies state to nodes that still exist in the tree (after pruning).
func (m *Model) applyExpansionState(node *model.TreeNode) {
	if node == nil {
		return
	}

	// Apply expansion state to group nodes
	if m.isGroupNode(node) {
		if expanded, ok := m.expansionStateValue(node); ok {
			node.Expanded = expanded
		} else {
			// Default to expanded for nodes without saved state
			node.Expanded = true
		}

	}

	// Recursively apply to children
	for _, child := range node.Children {
		m.applyExpansionState(child)
	}
}

func (m *Model) expansionStateValue(node *model.TreeNode) (bool, bool) {
	expansionState := m.uiState.GetExpansionState()
	if expansionState == nil {
		return false, false
	}

	key := m.nodeExpansionKey(node)
	if key != "" {
		expanded, ok := expansionState[key]
		if ok {
			return expanded, true
		}
	}

	legacyKey := m.nodeExpansionLegacyKey(node)
	if legacyKey == "" {
		return false, false
	}

	expanded, ok := expansionState[legacyKey]
	if !ok {
		return false, false
	}
	if key != "" {
		m.uiState.UpdateExpansionState(key, expanded)
		delete(expansionState, legacyKey)
	}
	return expanded, true
}

// getNodeIdentifier returns a stable identifier for a node.
// For notification nodes, this is the notification ID.
// For group nodes, this is a combination of the node kind and title.
func (m *Model) getNodeIdentifier(node *model.TreeNode) string {
	return m.treeService.GetNodeIdentifier(node)
}

// findNodeByIdentifier finds a node by its identifier in the visible nodes list.
func (m *Model) findNodeByIdentifier(identifier string) *model.TreeNode {
	for _, node := range m.treeService.GetVisibleNodes() {
		if m.treeService.GetNodeIdentifier(node) == identifier {
			return node
		}
	}
	return nil
}

// restoreCursor restores the cursor to the node with the given identifier.
// If the node is not found, it adjusts the cursor to be within bounds.
func (m *Model) restoreCursor(identifier string) {
	if identifier == "" {
		m.adjustCursorBounds()
		return
	}

	targetNode := m.findNodeByIdentifier(identifier)
	if targetNode != nil {
		visibleNodes := m.ensureTreeService().GetVisibleNodes()
		for i, node := range visibleNodes {
			if node == targetNode {
				m.uiState.SetCursor(i)
				m.uiState.EnsureCursorVisible(len(visibleNodes))
				return
			}
		}
	}

	// If we couldn't find the exact node, adjust to bounds
	m.adjustCursorBounds()
}

// adjustCursorBounds ensures the cursor is within valid bounds.
func (m *Model) adjustCursorBounds() {
	listLen := m.currentListLen()
	m.uiState.AdjustCursorBounds(listLen)
	m.uiState.EnsureCursorVisible(listLen)
}

// getSessionName returns the session name for a session ID.
// Uses RuntimeCoordinator for name resolution.
func (m *Model) getSessionName(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	if m.runtimeCoordinator == nil {
		return sessionID
	}

	name, err := m.runtimeCoordinator.GetSessionName(sessionID)
	if err == nil && name != "" {
		return name
	}

	return m.runtimeCoordinator.ResolveSessionName(sessionID)
}

// getWindowName returns the window name for a window ID.
// Uses RuntimeCoordinator for name resolution.
func (m *Model) getWindowName(windowID string) string {
	if windowID == "" {
		return ""
	}
	if m.runtimeCoordinator == nil {
		return windowID
	}

	name, err := m.runtimeCoordinator.GetWindowName(windowID)
	if err == nil && name != "" {
		return name
	}

	return m.runtimeCoordinator.ResolveWindowName(windowID)
}

// getPaneName returns the pane name for a pane ID.
// Uses RuntimeCoordinator for name resolution.
func (m *Model) getPaneName(paneID string) string {
	if paneID == "" {
		return ""
	}
	if m.runtimeCoordinator == nil {
		return paneID
	}

	name, err := m.runtimeCoordinator.GetPaneName(paneID)
	if err == nil && name != "" {
		return name
	}

	return m.runtimeCoordinator.ResolvePaneName(paneID)
}

// getTreeRootForTest returns the tree root for testing purposes.
func (m *Model) getTreeRootForTest() *model.TreeNode {
	return m.treeService.GetTreeRoot()
}

// getVisibleNodesForTest returns the visible nodes for testing purposes.
func (m *Model) getVisibleNodesForTest() []*model.TreeNode {
	return m.treeService.GetVisibleNodes()
}

// collectNotificationsInGroup collects all notifications under a group node.
// Returns the session, window, pane filters and count of notifications.
func (m *Model) collectNotificationsInGroup(node *model.TreeNode) (session, window, pane string, count int) {
	if node == nil || node.Kind == model.NodeKindNotification {
		return "", "", "", 0
	}

	count = node.Count

	switch node.Kind {
	case model.NodeKindSession:
		return node.Title, "", "", count
	case model.NodeKindWindow, model.NodeKindPane:
		sess, win, pan, ok := m.getAncestorTitles(node)
		if !ok {
			return "", "", "", 0
		}
		return sess, win, pan, count
	default:
		return "", "", "", 0
	}
}

// getGroupTypeLabel returns a human-readable label for the group kind.
func getGroupTypeLabel(kind model.NodeKind) string {
	switch kind {
	case model.NodeKindSession:
		return "session"
	case model.NodeKindWindow:
		return "window"
	case model.NodeKindPane:
		return "pane"
	case model.NodeKindMessage:
		return "message"
	default:
		return "group"
	}
}
