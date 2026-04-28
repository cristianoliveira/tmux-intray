# Fix gocognit violations in other packages

## Status
Planned on 2026-04-28.
Linked issue: `tmux-intray-pjec`.
Current issue status: `in_progress`.

## Problem
The backlog issue says there are non-storage `gocognit` violations to fix.

Current verification does **not** reproduce that state:

```bash
golangci-lint run --enable-only gocognit ./...
```

Result:

```text
0 issues.
```

So first task is epistemic: verify whether the issue is stale, environment-dependent, or hidden behind a different lint invocation/config.

## Success criteria
- Reproduce current `gocognit` violations, or prove there are none.
- If violations exist, fix them without behavior changes.
- If violations do not exist, close or rewrite the issue with evidence.
- Run targeted tests for touched packages.
- Run lint before landing changes.

## Plan
1. Reproduce exactly
   - Run the repo-standard lint command.
   - Capture only `gocognit` findings.
   - Exclude storage scope, since this issue is for other packages.

2. Identify true targets
   - Build a concrete list of files/functions over threshold.
   - Group by package.
   - Prefer app/cmd/tui candidates over broad repo churn.

3. Refactor smallest safe unit first
   - Extract helpers.
   - Flatten conditionals.
   - Prefer early returns.
   - Preserve outputs/errors.

4. Verify behavior
   - Run package-level tests for each changed package.
   - Re-run `gocognit` lint.
   - Re-run broader lint if needed.

5. Resolve backlog mismatch
   - If no violations remain, document evidence and close/update the issue.

## Likely commands
```bash
make lint

golangci-lint run --enable-only gocognit ./...

golangci-lint run --enable-only gocognit ./... 2>&1 | rg -v 'internal/storage'

go test ./internal/app ./cmd/tmux-intray ./internal/tui/...
```

## Notes
- `.golangci.yml` currently sets `gocognit.min-complexity: 30`.
- This may explain why older backlog counts no longer reproduce.
- Avoid changing lint thresholds unless evidence shows the issue is config-related rather than code-related.
