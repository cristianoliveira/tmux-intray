#!/bin/bash

# This script updates the test files to use the new UIState struct

# Replace viewport field references with uiState
sed -i '' 's/viewport: viewport\.New([^)]*)/uiState: NewUIState()/g' internal/tui/state/model_test.go
sed -i '' 's/viewport: viewport\.New([^,]*),/uiState: NewUIState(),/g' internal/tui/state/model_test.go

# Replace width field references with uiState.SetWidth
sed -i '' 's/width: \([0-9]*\)/uiState: NewUIState()\
		m.uiState.SetWidth(\1)/g' internal/tui/state/model_test.go

# Replace model.width with model.uiState.GetWidth()
sed -i '' 's/model\.width/model.uiState.GetWidth()/g' internal/tui/state/model_test.go

# Replace model.height with model.uiState.GetHeight()
sed -i '' 's/model\.height/model.uiState.GetHeight()/g' internal/tui/state/model_test.go

# Replace model.cursor with model.uiState.GetCursor()
sed -i '' 's/model\.cursor/model.uiState.GetCursor()/g' internal/tui/state/model_test.go

# Replace model.searchMode with model.uiState.IsSearchMode()
sed -i '' 's/model\.searchMode/model.uiState.IsSearchMode()/g' internal/tui/state/model_test.go

# Replace model.searchQuery with model.uiState.GetSearchQuery()
sed -i '' 's/model\.searchQuery/model.uiState.GetSearchQuery()/g' internal/tui/state/model_test.go

# Update model_bench_test.go as well
sed -i '' 's/viewport: viewport\.New([^)]*)/uiState: NewUIState()/g' internal/tui/state/model_bench_test.go
sed -i '' 's/viewport: viewport\.New([^,]*),/uiState: NewUIState(),/g' internal/tui/state/model_bench_test.go

# Replace width field references with uiState.SetWidth in bench tests
sed -i '' 's/width: \([0-9]*\)/uiState: NewUIState()\
		m.uiState.SetWidth(\1)/g' internal/tui/state/model_bench_test.go

# Replace model.cursor with model.uiState.GetCursor() in bench tests
sed -i '' 's/model\.cursor/model.uiState.GetCursor()/g' internal/tui/state/model_bench_test.go

# Replace model.searchQuery with model.uiState.GetSearchQuery() in bench tests
sed -i '' 's/model\.searchQuery/model.uiState.GetSearchQuery()/g' internal/tui/state/model_bench_test.go

echo "Updated test files to use UIState"
