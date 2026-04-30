# List should show human-readable tmux names by default

## Goal
Make `tmux-intray list` render human-readable session/window/pane names by default, while preserving access to internal tmux IDs through explicit flags.

## Problem
Current `list` output shows raw tmux IDs like `$1`, `@4`, `%4`.
That is implementation-facing, not user-facing.

Investigation so far:
- `cmd/tmux-intray/list.go` delegates to `internal/app/list.go`
- `internal/format/notification.go` prints raw `Notification.Session`, `Notification.Window`, `Notification.Pane`
- `defaultListSearchProvider()` in `cmd/tmux-intray/deps.go` already loads session/window/pane name maps, but only for search
- `internal/tmux/lists.go` already exposes `ListSessions`, `ListWindows`, `ListPanes`
- storage filters in `internal/storage/sqlite/queries.sql` use exact ID matches, so docs/examples claiming name-based `--session` filtering are currently misleading

## Constraints
- Keep JSON output stable unless we explicitly decide otherwise
- Prefer live enrichment at render/filter layer, not persisting names in storage
- Fallback to raw IDs when tmux lookup fails or entries no longer exist
- Avoid repeated tmux subprocess calls per row; resolve names in bulk once per command

## Plan

### 1. Lock current and desired behavior with tests first
- [ ] Add CLI/app formatter tests proving default `simple` output shows resolved names instead of raw IDs
- [ ] Add tests for fallback behavior when a name cannot be resolved
- [ ] Add tests for explicit raw-ID mode flag(s)
- [ ] Add tests for `table` / `compact` / `legacy` / `json` expected behavior after decision
- [ ] Add tests for grouped output to confirm group labels use names when grouping by session/window/pane

### 2. Decide output contract
- [ ] Define flag names and semantics
  - candidate: default = names
  - candidate explicit flags: `--ids`, `--show-ids`, or `--internal-ids`
- [ ] Decide whether to support combined display like `work ($1)` or pure name-only by default
- [ ] Decide JSON contract:
  - keep raw stored IDs only
  - or add extra derived fields for names without breaking existing fields
- [ ] Decide whether `legacy` and `compact` remain message-only with no routing metadata changes

### 3. Introduce a reusable name-resolution dependency
- [ ] Create a small resolver abstraction for list rendering/search
- [ ] Reuse bulk tmux lookups from `internal/tmux/lists.go`
- [ ] Move default construction to composition root in `cmd/tmux-intray/deps.go`
- [ ] Ensure resolver returns stable fallbacks when tmux is unavailable

### 4. Enrich list rendering path
- [ ] Add render-time enrichment for session/window/pane display values
- [ ] Keep domain/storage raw fields unchanged
- [ ] Ensure default simple formatter uses human-readable values
- [ ] Ensure grouping by session/window/pane uses resolved display labels
- [ ] Ensure search behavior still matches both IDs and names

### 5. Add explicit raw-ID mode
- [ ] Add CLI flags to force raw tmux IDs in output
- [ ] Make help text explicit about default human-readable behavior
- [ ] Verify raw-ID mode works for normal list and tab views

### 6. Fix filtering inconsistency
- [ ] Decide whether `--session`, `--window`, `--pane` should accept names, IDs, or both
- [ ] If both, resolve names to IDs before storage query
- [ ] If IDs only, fix docs/help to say IDs only
- [ ] Update tests to match real behavior

### 7. Update docs
- [ ] Update `--help` text for `list`
- [ ] Update examples in `examples/advanced-filtering.md`
- [ ] Update CLI docs/man page if generated from source comments
- [ ] Document fallback behavior when tmux metadata is unavailable

### 8. Validate
- [ ] Run targeted tests for `cmd/tmux-intray`, `internal/app`, `internal/format`, `internal/search`, `internal/tmux`
- [ ] Run full `go test ./...`
- [ ] Manually verify:
  - `go run ./cmd/tmux-intray list --tab recents`
  - raw-ID flag variant
  - grouped output
  - search by name and by ID

## Proposed implementation direction
Prefer this shape:
- keep storage and domain model raw
- resolve names once near composition/use-case boundary
- pass display metadata into formatter
- default CLI output = human-readable
- explicit flag = raw IDs

## Open questions
- Should default output be `work`, or `work ($1)`?
- Should JSON include derived name fields?
- Should `--session work` resolve to `$1` before querying, or stay strict and docs-only fix?

## Done when
- default `list` output is human-readable
- users can still request internal tmux IDs explicitly
- grouping/search/filter behavior is consistent and documented
- tests cover resolution, fallback, and raw-ID mode
