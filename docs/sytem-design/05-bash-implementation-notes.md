# Bash Implementation Notes

## Data model (TSV)
id \t timestamp \t state \t session \t window \t pane \t message

## Emit flow
- Acquire lock
- Allocate ID
- Append to notifications.tsv
- Increment tmux active_count

## List flow
- Load dismissed IDs
- Filter active notifications via awk
- Display in popup or scratch pane

## Jump flow
- Resolve record by ID
- switch-client
- select-window
- select-pane

## Dismiss flow
- Append ID to dismissed.tsv
- Decrement active_count

## Performance strategy
- Never scan logs in status bar
- Cap history size or age
- Explicit GC command if needed
