# Testing Strategy for Go CLI Migration

## Purpose & Scope

**Testing Goals for Go Migration**:
- Ensure backward compatibility with existing Bash CLI behavior
- Validate TSV storage format compatibility across Bash and Go implementations
- Verify hook execution consistency between implementations
- Maintain existing Bats test suite while adding Go-specific tests
- Provide confidence in gradual migration via dual-runner strategy

**Scope**:
- Core library packages (storage, config, hooks)
- CLI commands parity (add, list, dismiss, jump, etc.)
- TSV file format round-trip compatibility
- Hook execution environment
- Performance benchmarks comparing Bash vs Go
- Integration with existing tmux plugin and status bar

## Testing Pyramid & Coverage Targets

| Level | Target | Description |
|-------|--------|-------------|
| **Unit** | ≥80% for core Go packages | Storage, config, hooks packages; focus on isolated logic |
| **Integration** | TSV/hook/config compatibility | Bash↔Go interoperability, TSV round-trip, hook execution |
| **End-to-End / Shadow** | Compare Bash vs Go outputs | Golden outputs from Bats; diff outputs side-by-side |
| **Performance** | Benchmarks Go vs Bash | Latency, throughput, memory usage; establish thresholds |
| **CI Checks** | 100% pass | Linting, formatting, security checks, existing Bats suite |

## Test Types & Owners

| Type | Scope | Owner / Responsibility |
|------|-------|------------------------|
| **Unit** | Storage, config, hooks packages | Go development team |
| **Integration** | Bash↔Go, TSV round-trip, hooks execution | Integration lead |
| **CLI Command Tests** | Golden outputs from Bats | Test maintainer |
| **Performance** | Benchmarks, thresholds | Performance lead |
| **Security/Lint** | ShellCheck, Go vet, static analysis | Security reviewer |
| **Formatting** | shfmt, gofmt | CI pipeline |

**Detailed breakdown**:
- **Unit (Go)**: Storage library, configuration parsing, hook runner, tmux context capture
- **Unit (Bash)**: Existing Bats tests remain; new tests for migration helpers
- **Integration**: Dual-runner tests (Bash and Go binaries), TSV file compatibility, hook script execution
- **CLI**: Each command (`add`, `list`, `dismiss`, `jump`, etc.) tested with golden outputs
- **Performance**: Benchmarks for command latency, storage operations, concurrent access
- **Security**: ShellCheck for Bash scripts, Go vet/staticcheck for Go code, dependency scanning

## Environments & Data

**Test Data & Fixtures**:
- TSV sample files (valid, malformed, edge cases)
- Golden fixture location: `tests/fixtures/golden/` (planned) *(Note: this directory is planned but not yet created.)*
- Hooks test scripts in sandboxed temporary directories

**Dual-Runner Strategy**:
- Run Bash and Go binaries side-by-side in CI
- Same input → compare outputs and exit codes
- TSV storage directory isolated per test run

**Hook Sandboxing**:
- Temporary hook directory with minimal environment
- Capture hook stdout/stderr and exit codes
- Clean up after each test

## Tooling & Commands

| Tool / Target | Purpose |
|---------------|---------|
| `make tests` | Runs existing Bats test suite |
| `make lint` | Runs ShellCheck on all Bash scripts |
| `make security-check` | (Future) runs Go security scanning |
| `make check-fmt` | Checks shell script formatting |
| `go test ./...` | Runs Go unit and integration tests |
| `bats tests/` | Runs Bats integration tests |
| `benchmark runner` | (Placeholder) runs performance benchmarks |
| **CI Workflows** | GitHub Actions: `ci.yml`, `release.yml` |

**Existing CI Jobs**:
- `test` (macOS, Ubuntu): runs `make tests`
- `lint` (macOS, Ubuntu): runs `make lint`
- `format` (macOS, Ubuntu): runs `make check-fmt`
- `install` (macOS): audits Homebrew formula, tests npm/Go/source install
- `install-linux` (Ubuntu): tests npm/Go/Docker/source install

## Shadow / Canary Strategy

**Dual-Execution Model**:
1. Run Bash command, capture output and exit code
2. Run Go command with same input, capture output and exit code
3. Diff outputs (allow for known differences, e.g., version strings)
4. Track regressions over time

**Implementation**:
- Shadow test runner script (Bash) that invokes both binaries
- Diff tool that ignores permissible differences (timestamps, temporary paths)
- Report generation for any unexpected deviations

**Canary Deployment**:
- Gradually shift traffic from Bash to Go binary
- Monitor error rates and performance metrics
- Rollback on any regression

## Performance Benchmarks

**Scope**:
- Command-level latency (mean, p95, p99)
- Storage operations (add, list, dismiss) with varying dataset sizes
- Concurrent access scaling (multiple tmux instances)
- Memory footprint (RSS) of long-running processes

**Metrics to Capture**:
- Time to add 1 / 100 / 1000 notifications
- Time to list filtered notifications
- Time to dismiss a notification by ID
- Time to jump to source pane
- File I/O overhead (TSV reads/writes)

**Trigger Cadence**:
- On every PR that touches performance-sensitive code
- Weekly scheduled run on main branch
- Before each major release

**Thresholds**:
- Go implementation should not degrade performance by more than 10% compared to the Bash baseline
- Memory usage increase should not exceed 20% compared to the Bash baseline

## CI Integration

**Checks run on PR / main**:

| Check | Trigger | Gating Policy |
|-------|---------|---------------|
| **Bats tests** | Every PR | Required |
| **Go unit tests** | PR with Go changes | Required |
| **ShellCheck lint** | Every PR | Required |
| **Format check** | Every PR | Required |
| **Shadow diff** | PR with Go changes | Required (no unexpected diffs) |
| **Performance benchmarks** | PR with perf-sensitive changes | Advisory (failures do not block) |
| **Installation tests** | Every PR | Required (macOS & Linux) |


**Definitions**:
- **Go changes**: Modifications to any Go module (including `cmd/`, `internal/`, `pkg/` directories).
- **Perf-sensitive changes**: Modifications to benchmarks, core libraries, or any code that could affect performance (storage operations, hook execution, etc.).
**Artifacts**:
- Coverage reports (Go)
- Benchmark results (historical comparison)
- Shadow diff reports
- Lint reports

**Gating**:
- All required checks must pass before merge
- Shadow diff must show zero unexpected differences
- Existing Bats suite must remain green

## Risk & Gaps

**Known Risks**:
- **Flaky tests**: Hook sandboxing may have environment dependencies; use strict isolation
- **Performance measurement noise**: Use statistical significance and multiple runs
- **Missing tooling**: Go benchmark runner not yet implemented; mark as placeholder
- **Golden test maintenance**: Golden outputs may need updating as behavior evolves

**Gaps**:
- **Markdown lint**: Optional; could be added later
- **Go coverage enforcement**: Not yet integrated into CI
- **Security scanning for Go**: Need to add `go vet`, `staticcheck`, `gosec`
- **End-to-end tmux integration**: Requires full tmux environment; currently tested via Bats

**Mitigations**:
- Implement benchmark runner as part of migration tasks
- Add Go coverage threshold enforcement once baseline established
- Integrate security scanning as separate CI job
- Use Docker-based tmux environment for E2E tests (future)

**Assumptions**:
- Bash implementation remains stable during migration
- TSV storage format is final and will not change
- Hook interface (environment variables, arguments) is stable
- Go binary will eventually replace Bash binary completely

## References
- [tmux-intray CLI Feature Implementation Plan](../implementation/tmux-intray-implementation-plan.md)
- [Go Migration Task Report](../implementation/go-migration-task-report.md)
- [CI Workflow](../../.github/workflows/ci.yml)
- [Makefile](../../Makefile)