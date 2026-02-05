# Golden Test Extraction Design for Go Migration

## Purpose & Scope

**Why golden tests?**
- Ensure backward compatibility of Go CLI with existing Bash implementation
- Capture exact expected outputs (stdout, stderr, exit codes) for each command under various inputs
- Provide deterministic validation during migration and future changes
- Enable table-driven Go tests that are easy to maintain and update

**Scope**
- All CLI commands (`add`, `list`, `show`, `status`, `jump`, `dismiss`, `follow`, `management`, `help`, `version`)
- Storage operations (TSV file reading/writing, locking)
- Hook execution side effects (stdout/stderr capture)
- tmux integration where possible (isolated tmux sessions)
- Edge cases: empty inputs, invalid arguments, long messages, missing tmux context

**Out of scope**
- Performance benchmarks (covered by separate performance testing)
- UI/UX changes (golden tests focus on functional behavior)
- Third‑party tool integration (e.g., external hook scripts) beyond capturing their output

## Sources

**Bats test locations**
- `tests/*.bats` – core functionality (basic, storage, tray, cli, install)
- `tests/commands/*.bats` – per‑command tests (add, list, show, status, jump, dismiss, follow, management, status‑panel, list‑pane)
- Total: ~15 Bats files containing ~60 individual test cases

**Single‑runner approach**
- Each golden fixture will be derived from the Go CLI (`tmux-intray`) as the oracle
- The Go binary is now the single source of truth for all commands

**Parsing Bats tests**
- Extract test names, command invocations, input arguments, environment variables, and expected assertions
- Some tests rely on tmux sessions; these will be isolated or simulated

## Fixture Format & Location

**Directory structure**
```
tests/fixtures/golden/
├── command/
│   ├── add/
│   │   ├── requires-message.json
│   │   ├── item-to-tray.json
│   │   └── empty-message-fails.json
│   ├── list/
│   │   ├── default-empty.json
│   │   └── with-items.json
│   └── ...
├── storage/
│   ├── tsv-roundtrip.json
│   └── lock-acquisition.json
└── hooks/
    ├── pre-add-executes.json
    └── non-zero-exit-warning.json
```

**File format: JSON (primary) or TSV (where appropriate)**
- JSON for command fixtures (structured, easy to extend)
- TSV for storage‑related fixtures (mirroring actual TSV file content)

**JSON fixture fields**
```json
{
  "name": "add requires a message",
  "source": "tests/commands/add.bats:17",
  "description": "Invoking add without arguments shows error and exits with code 1",
  "inputs": {
    "args": [],
    "env": {
      "TMUX_INTRAY_STATE_DIR": "/tmp/tmux-intray-test-12345"
    },
    "cwd": "/tmp/test",
    "stdin": ""
  },
  "expected": {
    "exit_code": 1,
    "stdout": "",
    "stderr": "tmux-intray add: requires a message\n",
    "filesystem": {
      "pre_state": {},
      "post_state": {
        "$TMUX_INTRAY_STATE_DIR/notifications.tsv": null
      }
    },
    "tmux_options": {}
  },
  "metadata": {
    "command": "add",
    "requires_tmux": false,
    "hooks_involved": false,
    "timestamp": "2026-02-04T00:00:00Z",
    "version": "bash-1.0"
  }
}
```

**TSV canonicalization rules**
- Field order: ID, timestamp, state, session, window, pane, message, pane_created, level
- Timestamp format: RFC3339 UTC (`2006-01-02T15:04:05Z`)
- Message escaping: backslashes → `\\`, tabs → `\t`, newlines → `\n`
- Empty fields represented as empty string (no placeholder)
- Trailing newline at end of file (Unix `\n`)

## Extraction Process

**Steps**
1. **Parse Bats files** – identify each `@test` block, extract command line, environment, and assertions
2. **Isolate test environment** – create temporary directories for state and config, set up clean environment variables
3. **Execute Bash CLI** – run the extracted command with the same arguments, capture stdout, stderr, exit code
4. **Capture filesystem state** – record pre‑ and post‑test contents of relevant directories (state dir, config dir)
5. **Normalize outputs** – remove timing‑dependent strings (e.g., “added in 0.12s”), canonicalize paths, strip colors
6. **Write fixture** – serialize as JSON (or TSV) into the appropriate location

**Sample extraction script outline (`scripts/extract-golden-fixtures.sh`)**
```bash
#!/usr/bin/env bash
set -euo pipefail

# Configuration
BATS_DIR="tests"
FIXTURES_DIR="tests/fixtures/golden"
GO_CLI="./tmux-intray"

# For each .bats file
for bats_file in "$BATS_DIR"/*.bats "$BATS_DIR"/commands/*.bats; do
  # Parse test blocks (simplified)
  while read -r test_block; do
    # Extract test name, command, env, assertions
    # ...

    # Create temporary sandbox
    tmpdir=$(mktemp -d)
    export TMUX_INTRAY_STATE_DIR="$tmpdir/state"
    export TMUX_INTRAY_CONFIG_DIR="$tmpdir/config"
    mkdir -p "$TMUX_INTRAY_STATE_DIR" "$TMUX_INTRAY_CONFIG_DIR"

    # Build and run Go CLI
    make go-build
    output=$("$GO_CLI" "${args[@]}" 2>&1)
    exit_code=$?

    # Capture stdout/stderr (separately if possible)
    # ...

    # Write fixture
    jq -n \
      --arg name "$test_name" \
      --arg source "$bats_file:$line" \
      --argjson args "$(printf '%s\n' "${args[@]}" | jq -R . | jq -s .)" \
      --argjson exit_code "$exit_code" \
      --arg stdout "$stdout" \
      --arg stderr "$stderr" \
      '{
         name: $name,
         source: $source,
         inputs: { args: $args, env: {}, cwd: "", stdin: "" },
         expected: { exit_code: $exit_code, stdout: $stdout, stderr: $stderr }
       }' > "$FIXTURES_DIR/$command/$test_name.json"
  done < <(grep -n '@test' "$bats_file")
done
```

**Suggested commands**
- Use `bats --tap` to run individual tests and capture output (but may not give fine‑grained control)
- Direct parsing of `.bats` files with `awk`/`sed` is simpler for extraction
- Use `jq` for JSON serialization
- Use `tmux -L` with isolated socket for tmux‑dependent tests

## Naming & Organization

**Naming conventions**
- Fixture file names: `kebab-case.json` matching test description (e.g., `requires-message.json`)
- Command folder names: same as subcommand (e.g., `add`, `list`, `status`)
- Descriptive test names in fixture `name` field (same as Bats `@test` label)

**Versioning**
- Include `version` field in metadata to track which Bash version generated the fixture
- When behavior intentionally changes, fixtures can be regenerated with a new version tag

**Multiple variants**
- For commands that produce different output depending on flags (e.g., `list --format=json` vs `list --format=tsv`), create separate fixtures: `list/format-json.json`, `list/format-tsv.json`
- For environment‑dependent outputs (e.g., colors enabled/disabled), capture both variants or decide on a canonical form (colors stripped)

## Validation Workflow

**Go test consumption**
- Table‑driven test in Go that loads all fixtures for a given command
- For each fixture, set up the same environment (temp dirs, env vars)
- Invoke Go CLI (`tmux-intray-go`) with the same inputs
- Compare exit code, stdout, stderr against expected values

**Comparison rules**
- **Exact match** for stdout/stderr (after normalization)
- **Allowed differences**: version strings, timestamps, temporary paths – these should be normalized during extraction
- **Tolerance**: none for functional correctness; small differences indicate a regression

**Updating fixtures**
- While Bash is oracle: re‑run extraction script after any Bash behavior change
- After switchover to Go: update fixtures via Go CLI (new extraction script that uses Go binary)
- PR checklist: any change to CLI behavior must be accompanied by fixture updates; diff of fixtures should be reviewed

**Sample Go test structure (`internal/testutil/golden.go`)**
```go
package testutil

import (
    "encoding/json"
    "os"
    "path/filepath"
    "testing"
)

type GoldenFixture struct {
    Name     string          `json:"name"`
    Source   string          `json:"source"`
    Inputs   Inputs          `json:"inputs"`
    Expected Expected        `json:"expected"`
}

type Inputs struct {
    Args []string          `json:"args"`
    Env  map[string]string `json:"env"`
    Cwd  string            `json:"cwd"`
    Stdin string           `json:"stdin"`
}

type Expected struct {
    ExitCode int    `json:"exit_code"`
    Stdout   string `json:"stdout"`
    Stderr   string `json:"stderr"`
}

func LoadGolden(t *testing.T, command string) []GoldenFixture {
    pattern := filepath.Join("tests", "fixtures", "golden", command, "*.json")
    files, err := filepath.Glob(pattern)
    // ... load each file, decode JSON
}

func RunGoldenTest(t *testing.T, fixture GoldenFixture) {
    // Set up environment
    // Run Go CLI
    // Compare results
}
```

**Integration with `go test`**
```go
func TestAddCommandGolden(t *testing.T) {
    for _, fixture := range testutil.LoadGolden(t, "add") {
        t.Run(fixture.Name, func(t *testing.T) {
            testutil.RunGoldenTest(t, fixture)
        })
    }
}
```

## Tooling

**Extraction script (`scripts/extract-golden-fixtures.sh`)**
- Input: Bats test suite
- Output: JSON fixtures in `tests/fixtures/golden/`
- Dependencies: `bash`, `jq`, `tmux` (optional), `coreutils`
- Should be idempotent; can be run locally or in CI

**Go test helper (`internal/testutil/golden.go`)**
- Provides `LoadGolden`, `RunGoldenTest`
- Handles temporary directory creation, environment setup, command execution, comparison
- Can be extended with custom comparators (e.g., regex for timestamps)

**Dependencies**
- `jq` for JSON manipulation (extraction script)
- `tmux` for tmux‑dependent fixtures (optional; can skip those tests)
- `go` 1.24+ for Go test helper
- `bats` not required for extraction (direct parsing)

**CI integration**
- Add a job `generate‑golden` that runs the extraction script and commits changes if any
- Ensure generated fixtures are up‑to‑date before merging PRs that affect CLI behavior

## Limitations & Open Questions

**Flaky tests**
- Tests that depend on timing (e.g., “added in 0.12s”) must have those parts stripped during normalization
- Randomness (e.g., generated IDs) should be replaced with placeholders or ignored in comparison

**Time‑dependent outputs**
- Timestamps in notifications: use RFC3339 format; during comparison, allow any valid timestamp
- Duration strings: remove them or replace with a placeholder

**Hooks side effects**
- Hook execution may have external side effects (e.g., sending notifications)
- Sandboxing: run hooks in a temporary directory with no network access; mock external services

**tmux‑dependent cases**
- Some tests require a live tmux server (`tmux -L` with isolated socket)
- Extraction script can start a temporary tmux server, but adds complexity
- Alternative: skip tmux‑dependent fixtures initially, rely on integration tests later

**Large fixtures**
- Storage tests with thousands of notifications may produce large TSV files
- Consider compressing fixtures or storing only the delta

**Open questions**
1. Should we capture the entire filesystem state or only the notifications TSV file?
2. How to handle tests that involve multiple commands (e.g., add then list)?
3. What to do with tests that rely on external tools (`awk`, `sed`, `jq`)?
4. How to version fixtures when the Bash implementation changes (e.g., bug fix)?
5. Should we include a checksum of the Bash binary to detect version skew?

## Rollout Plan

**Phase 1 – Foundation**
1. Create `tests/fixtures/golden/` directory structure
2. Implement extraction script for simple, non‑tmux commands (`add`, `list`, `show`, `help`, `version`)
3. Implement Go test helper and table‑driven tests for those commands
4. Run extraction, generate initial fixtures, commit

**Phase 2 – Storage & config**
1. Extract fixtures for storage operations (TSV round‑trip, locking)
2. Add config‑related fixtures (environment precedence, config file parsing)
3. Extend Go test helper to handle filesystem state comparison

**Phase 3 – tmux integration**
1. Add isolated tmux server support to extraction script
2. Generate fixtures for `status`, `jump`, `follow`, `management` commands
3. Update Go tests to optionally skip tmux‑dependent tests when tmux not available

**Phase 4 – Hooks**
1. Extract fixtures for hook execution (pre‑add, post‑add, etc.)
2. Sandbox hook scripts to prevent side effects
3. Validate hook stdout/stderr capture

**Phase 5 – Completion**
1. Ensure all Bats tests have corresponding golden fixtures
2. Run Go tests in CI alongside Bats suite
3. Switch oracle from Bash to Go (once Go implementation is fully validated)

**PR checklist for updating fixtures**
- [ ] Run extraction script (`scripts/extract-golden-fixtures.sh`)
- [ ] Review diff of generated fixtures (`git diff tests/fixtures/golden`)
- [ ] Ensure Go tests still pass (`go test ./...`)
- [ ] Update any fixture‑related documentation

**Maintenance**
- Golden fixtures should be updated whenever Bash behavior changes (during migration)
- After migration, Go becomes the source of truth; a similar extraction script can be built for Go