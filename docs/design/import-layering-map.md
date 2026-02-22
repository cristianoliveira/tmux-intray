# Import Layering Map (Target)

This document defines the target package layering for `tmux-intray` and the
explicit dependency edges that are allowed and denied.

## Layer Definitions

The target layering is defined by package groups (prefixes):

1. `cli`
   - `github.com/cristianoliveira/tmux-intray/cmd`
   - `github.com/cristianoliveira/tmux-intray/cmd/tmux-intray`
2. `presentation`
   - `github.com/cristianoliveira/tmux-intray/internal/tui/*`
   - `github.com/cristianoliveira/tmux-intray/internal/format`
   - `github.com/cristianoliveira/tmux-intray/internal/status`
3. `application`
   - `github.com/cristianoliveira/tmux-intray/internal/core`
   - `github.com/cristianoliveira/tmux-intray/internal/tmuxintray`
4. `domain`
   - `github.com/cristianoliveira/tmux-intray/internal/domain`
   - `github.com/cristianoliveira/tmux-intray/internal/notification`
   - `github.com/cristianoliveira/tmux-intray/internal/search`
   - `github.com/cristianoliveira/tmux-intray/internal/dedup`
5. `infrastructure`
   - `github.com/cristianoliveira/tmux-intray/internal/storage*`
   - `github.com/cristianoliveira/tmux-intray/internal/tmux`
   - `github.com/cristianoliveira/tmux-intray/internal/config`
   - `github.com/cristianoliveira/tmux-intray/internal/dedupconfig`
   - `github.com/cristianoliveira/tmux-intray/internal/settings`
   - `github.com/cristianoliveira/tmux-intray/internal/hooks`
   - `github.com/cristianoliveira/tmux-intray/internal/colors`
   - `github.com/cristianoliveira/tmux-intray/internal/errors`
   - `github.com/cristianoliveira/tmux-intray/internal/logging`
   - `github.com/cristianoliveira/tmux-intray/internal/version`

## Allowed Edges

Explicitly allowed inter-layer imports:

- `cli -> presentation`
- `cli -> application`
- `cli -> domain`
- `cli -> infrastructure`
- `presentation -> application`
- `presentation -> domain`
- `presentation -> infrastructure`
- `application -> domain`
- `application -> infrastructure`
- `domain -> domain`
- `infrastructure -> domain`
- `infrastructure -> infrastructure`

## Denied Edges

The following inter-layer imports are denied in the target architecture:

- `presentation -> cli`
- `application -> cli`
- `application -> presentation`
- `domain -> cli`
- `domain -> presentation`
- `domain -> application`
- `domain -> infrastructure`
- `infrastructure -> cli`
- `infrastructure -> presentation`
- `infrastructure -> application`

## Baseline Artifact and Regeneration

Current baseline snapshot is committed at:

- `docs/design/import-graph-baseline.tsv`

Regenerate with either command:

- `./scripts/generate-import-graph.sh`
- `make import-graph`

Validate the snapshot is up to date:

- `./scripts/generate-import-graph.sh && git diff --exit-code docs/design/import-graph-baseline.tsv`
