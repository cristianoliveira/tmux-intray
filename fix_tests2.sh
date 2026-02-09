#!/bin/bash

# Replace viewport field references with uiState
sed -i '' 's/viewport: viewport\.New([^)]*)/uiState: NewUIState()/g' internal/tui/state/model_test.go
sed -i '' 's/width: \([0-9]*\)/uiState: NewUIState()/g' internal/tui/state/model_test.go

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

echo "Updated model_test.go"
