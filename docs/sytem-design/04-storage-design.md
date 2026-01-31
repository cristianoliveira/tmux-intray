# Storage Design

## Design constraints
- tmux state is string-based
- bash-first implementation
- minimal dependencies
- must scale to long-lived sessions

## Evaluated options

### tmux options / environment
- Good for small state
- Poor for growing datasets

### Flat files
- Append-only
- Simple concurrency with flock
- Fast enough for thousands of entries

### SQLite
- Powerful querying and indexing
- Adds external dependency
- Overkill for v0.1

## Chosen approach: Hybrid

- Append-only TSV log for notifications
- Separate log for dismissed IDs
- tmux options for:
  - active notification count
  - next notification ID

This keeps:
- status bar fast
- writes cheap
- reads manageable

## File layout

- notifications.tsv
- dismissed.tsv
- notifications.lock
