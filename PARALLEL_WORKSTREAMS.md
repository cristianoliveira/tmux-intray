# Parallel Workstreams: Extend `bd status` with `--format` Flag

**Epic**: Extend `bd status` with `--format` flag for custom template formatting  
**Status**: Planning Phase  
**Document Version**: 1.0  
**Date**: February 26, 2026

---

## Executive Summary

This document breaks down the status format extension epic into **5 parallel, independently executable deliverables** with minimal file overlap and clear integration points.

**Key Philosophy**:
- Separate implementers should rarely touch the same files
- Clear contracts (interfaces) allow parallel development
- Integration testing happens at boundaries
- Shared dependencies (template engine) must complete first

---

## Deliverables Overview

| # | Deliverable | Owner | Duration | Dependencies | Status |
|---|-------------|-------|----------|--------------|--------|
| **1** | **Template Engine & Variables** | Backend | 2 days | None | Blocking |
| **2** | **Format Command Integration** | CLI Engineer | 1.5 days | D1 | Blocking |
| **3** | **Backward Compatibility Layer** | Backend | 1 day | D1, D2 | Sequential |
| **4** | **Integration & End-to-End Tests** | QA/Test | 1.5 days | D1, D2, D3 | Sequential |
| **5** | **Documentation & Help Text** | Tech Writer | 1-2 days | D2, D4 | Sequential |

**Total Effort**: ~6.5 days (sequential path: D1 → D2 → D3 → D4 → D5)  
**Parallel Tracks**: D1 is blocking; D2-D3 can overlap with D1's end; D4-D5 follow in sequence

---

## Detailed Deliverables

### Deliverable 1: Template Engine & Variables (Shared Dependency)

**Owner**: Backend Engineer  
**Duration**: 2 days  
**Files Touched**:
- `internal/formatter/` (NEW directory)
  - `template.go` - Template parser and renderer
  - `template_test.go` - Unit tests for parser
  - `variables.go` - Variable resolver with contract
  - `variables_test.go` - Unit tests for resolver
  - `presets.go` - Preset registry
  - `presets_test.go` - Preset tests

**Files Read (Dependencies)**:
- `internal/format/status.go` - For data structure patterns
- `internal/storage/` - To understand data models

**Acceptance Criteria**:

✅ **AC1.1: Template Parser**
- [ ] Parser correctly identifies variables using `${variable-name}` syntax
- [ ] Parser validates variable names against a whitelist
- [ ] Parser rejects invalid variable names with clear error messages
- [ ] Parser handles edge cases: empty templates, no variables, special characters
- [ ] 100% of parser code has unit test coverage

✅ **AC1.2: Variable Resolver**
- [ ] `VariableResolver` interface defined and implemented
- [ ] Resolves all 13 template variables correctly:
  - `${unread-count}`, `${total-count}` (alias)
  - `${info-count}`, `${warning-count}`, `${error-count}`, `${critical-count}`
  - `${has-critical}`, `${has-error}`, `${has-warning}`, `${has-info}`
  - `${highest-severity}`, `${pane-count}`, `${last-updated}`
- [ ] Type conversions correct: integers → strings, booleans → "true"/"false"
- [ ] Handles missing variables gracefully (returns error with list of valid variables)
- [ ] >80% unit test coverage

✅ **AC1.3: Preset Registry**
- [ ] Presets registered: `compact`, `detailed`, `json`, `count-only`, `levels`, `panes`
- [ ] `IsPreset(name)` returns true/false correctly
- [ ] `RenderPreset(name, context)` returns correct output for each preset
- [ ] Output formats match documented specifications
- [ ] >80% unit test coverage

✅ **AC1.4: Data Contract**
- [ ] `VariableContext` struct defined with all needed fields (11 fields)
- [ ] All fields properly typed and documented
- [ ] Struct is exported and usable from other packages
- [ ] Interfaces exported: `TemplateEngine`, `VariableResolver`, `PresetRegistry`

**Verification Commands**:
```bash
cd /Users/cristianoliveira/other/tmux-intray

# Unit tests for formatter
go test ./internal/formatter/... -v -cover

# Check coverage
go test ./internal/formatter/... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Verify no errors
go build ./internal/formatter/...
```

**Success Criteria**:
- All unit tests pass
- >80% code coverage for formatter package
- No linter warnings
- Interfaces are clean and exportable

**Known Gotchas**:
- Template variable names must support hyphens (not just underscores) - use string parsing carefully
- `highest-severity` requires mapping info/warning/error/critical to ordinal values
- `${total-count}` is an alias for `${unread-count}` - handle in resolver
- Boolean variables must return literal "true"/"false" strings, not Go bools

---

### Deliverable 2: Format Command Integration

**Owner**: CLI Engineer  
**Duration**: 1.5 days  
**Dependencies**: D1 (Template Engine & Variables must be ready)  
**Files Touched**:
- `cmd/tmux-intray/status.go` (MODIFY)
  - Add `--format` flag parsing
  - Integrate template engine
  - Update RunE handler
  - Add validation

- `cmd/tmux-intray/status_test.go` (MODIFY)
  - Add tests for `--format` flag
  - Add tests for preset formats
  - Add tests for custom templates
  - Verify backward compatibility with existing format names

**Files Read (Dependencies)**:
- `internal/formatter/` (from D1)
- `internal/format/status.go` - Data extraction patterns
- `cmd/tmux-intray/status.go` - Current implementation

**Acceptance Criteria**:

✅ **AC2.1: Flag Parsing**
- [ ] `--format` flag accepted and parsed correctly
- [ ] Default value is "compact" (backward compatible)
- [ ] Preset names recognized: compact, detailed, json, count-only, levels, panes
- [ ] Custom templates parsed as templates when not a preset name
- [ ] Help text updated with flag description and examples

✅ **AC2.2: Template Rendering**
- [ ] Custom template `--format='${unread-count} notifications'` renders correctly
- [ ] Custom template with multiple variables works
- [ ] Tmux color codes pass through unchanged: `--format='#[fg=red]${critical-count}#[default]'`
- [ ] Invalid variable names return error with helpful message
- [ ] Exit code is 0 on success, 1 on template error

✅ **AC2.3: Preset Formats**
- [ ] `--format=compact` produces expected output
- [ ] `--format=detailed` produces level breakdown
- [ ] `--format=json` produces valid JSON
- [ ] `--format=count-only` produces only the count
- [ ] `--format=levels` produces key:value format
- [ ] `--format=panes` produces pane breakdown
- [ ] All outputs match format/ package specifications

✅ **AC2.4: Integration with Status Query**
- [ ] Data flows correctly from client.ListNotifications() to template context
- [ ] Count aggregation matches existing behavior
- [ ] Severity levels computed correctly
- [ ] Last-updated timestamp extracted properly

✅ **AC2.5: Backward Compatibility**
- [ ] Existing "summary", "levels", "panes", "json" format names still work
- [ ] Default behavior (no --format flag) unchanged
- [ ] Output of `bd status` without flags matches previous behavior

**Verification Commands**:
```bash
cd /Users/cristianoliveira/other/tmux-intray

# Unit tests
go test ./cmd/tmux-intray/ -v -run TestStatus -cover

# Integration test - format flag parsing
go run ./cmd/tmux-intray status --format=compact
go run ./cmd/tmux-intray status --format='${critical-count}'
go run ./cmd/tmux-intray status --format=json

# Help text
go run ./cmd/tmux-intray status --help

# Backward compat
go run ./cmd/tmux-intray status  # Should work like before
```

**Success Criteria**:
- All status command tests pass
- Presets produce correct output
- Custom templates work
- Error messages are clear
- Help text is informative

**Known Gotchas**:
- Must not break existing `status-panel` functionality
- Need to handle both preset names and custom templates in same flag
- May need to normalize preset names (case-insensitive?)
- Exit code behavior must align with error handling

---

### Deliverable 3: Backward Compatibility Layer

**Owner**: Backend Engineer  
**Duration**: 1 day  
**Dependencies**: D1 (Template Engine), D2 (Command Integration)  
**Files Touched**:
- `cmd/tmux-intray/status-panel-cmd.go` (MODIFY)
  - Refactor to delegate to new formatter
  - Map old format names to new templates
  - Maintain identical output

- `internal/format/status.go` (MODIFY)
  - Add helper function to create VariableContext from query results
  - Export any needed data extraction utilities

- `cmd/tmux-intray/status-panel-core.go` (MODIFY if needed)
  - Ensure data flows through new formatter

**Files Read**:
- `internal/formatter/` (from D1)
- Current `status-panel` implementation

**Acceptance Criteria**:

✅ **AC3.1: Delegation Pattern**
- [ ] `status-panel` command internally uses new `--format` with appropriate template
- [ ] Output of `status-panel` is pixel-perfect identical to before
- [ ] No external behavior change - users see no difference
- [ ] Configuration keys still work unchanged

✅ **AC3.2: Format Compatibility Mapping**
- [ ] Old internal formats mapped to new templates:
  - "summary" → compact preset
  - "detailed" → detailed preset
  - Custom formats delegated appropriately
- [ ] Mapping is documented in code comments

✅ **AC3.3: Data Flow**
- [ ] Data extraction helpers in `internal/format/status.go` used by both paths
- [ ] No duplication of data aggregation logic
- [ ] Shared code paths ensure consistency

✅ **AC3.4: Regression Testing**
- [ ] All existing `status-panel` tests pass unchanged
- [ ] Output comparison tests verify pixel-perfect matching
- [ ] No deprecation warnings appear (per design decision)

**Verification Commands**:
```bash
cd /Users/cristianoliveira/other/tmux-intray

# Run existing tests
bats tests/commands/status-panel.bats

# Verify output identity
OLD_OUTPUT=$(go run ./cmd/tmux-intray status-panel)
NEW_OUTPUT=$(go run ./cmd/tmux-intray status --format=compact)
diff <(echo "$OLD_OUTPUT") <(echo "$NEW_OUTPUT")

# Verify no data loss
go test ./internal/format/... -v
```

**Success Criteria**:
- All legacy tests pass
- No output regressions
- Clear code comments on delegation
- Data flow is understandable

**Known Gotchas**:
- Must preserve exact spacing and formatting of old output
- May need to handle environment variables (TMUX_INTRAY_STATUS_FORMAT)
- Configuration compatibility must be preserved
- Any color codes must pass through correctly

---

### Deliverable 4: Integration & End-to-End Tests

**Owner**: QA Engineer / Test Specialist  
**Duration**: 1.5 days  
**Dependencies**: D1, D2, D3 (all core implementation done)  
**Files Touched**:
- `tests/commands/status.bats` (MODIFY/EXPAND)
  - Add format flag tests
  - Add template variable tests
  - Add preset format tests
  - Add error cases

- `cmd/tmux-intray/status_test.go` (MODIFY)
  - Add integration tests with real data
  - Add end-to-end format tests

- Create: `tests/integration/formatter_e2e.bats` (NEW, optional)
  - End-to-end formatter tests with actual notification data

**Files Read**:
- `cmd/tmux-intray/` (all status-related code)
- `internal/formatter/` (to understand behavior)
- `internal/format/status.go` (test data patterns)

**Acceptance Criteria**:

✅ **AC4.1: Format Flag Acceptance Tests**
- [ ] `--format=compact` works end-to-end
- [ ] `--format='${unread-count}'` outputs just the count
- [ ] `--format='${critical-count} critical'` includes literal text
- [ ] Multiple variables in one template work correctly
- [ ] All 6 presets work: compact, detailed, json, count-only, levels, panes

✅ **AC4.2: Template Variable Tests**
- [ ] Each of 13 variables resolves correctly with test data
- [ ] Boolean variables output "true"/"false" not Go boolean
- [ ] Large numbers format correctly (no truncation)
- [ ] Timestamp format is ISO 8601
- [ ] Empty data sets handled (all counts → 0)

✅ **AC4.3: Error Handling Tests**
- [ ] Invalid variable name → error message listing valid ones
- [ ] Malformed template → helpful error
- [ ] Unknown preset → error message listing valid presets
- [ ] Exit codes correct: 0 success, 1 error, 2 system error

✅ **AC4.4: Backward Compatibility Tests**
- [ ] `status-panel` output unchanged (regression)
- [ ] Old environment variable TMUX_INTRAY_STATUS_FORMAT still works
- [ ] Default `bd status` (no flags) produces compact output
- [ ] All existing status tests pass

✅ **AC4.5: Data Accuracy Tests**
- [ ] Counts match actual notification data
- [ ] Severity levels correct in all presets
- [ ] Pane counts accurate and sorted
- [ ] Last-updated timestamp current

✅ **AC4.6: Coverage**
- [ ] All new code paths tested
- [ ] Positive and negative cases covered
- [ ] Overall coverage >80% including all code added in D1-D3

**Verification Commands**:
```bash
cd /Users/cristianoliveira/other/tmux-intray

# Run all status tests
bats tests/commands/status.bats -v

# Run all formatter tests
go test ./internal/formatter/... -v

# Run status command tests
go test ./cmd/tmux-intray -v -run TestStatus -cover

# Full coverage report
go test ./internal/formatter/... ./cmd/tmux-intray/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Backward compat regression check
bats tests/commands/status-panel.bats
```

**Success Criteria**:
- 100% of bats tests pass
- 100% of Go unit tests pass
- Coverage report shows >80% for all new code
- No regressions in existing functionality
- Clear test names describing what's being verified

**Known Gotchas**:
- Mock data setup can be complex - use fixtures
- Timestamp comparisons need to handle "now" values
- Bats tests need proper tmux session setup
- JSON output must be exactly comparable (no whitespace diffs)
- Pane counting requires understanding tmux structure

---

### Deliverable 5: Documentation & Help Text

**Owner**: Technical Writer / Documentation Specialist  
**Duration**: 1-2 days  
**Dependencies**: D2, D4 (command is stable, all tests passing)  
**Files Touched**:
- `cmd/tmux-intray/status.go` (MODIFY)
  - Update Long help text with detailed examples
  - Add variable list to help
  - Document preset formats
  - Add usage patterns

- Create: `docs/status-command-guide.md` (NEW)
  - Comprehensive user guide
  - Real-world examples
  - Integration guides
  - Troubleshooting
  - Migration path from status-panel

- `docs/cli/CLI_REFERENCE.md` (MODIFY)
  - Add status command section
  - Document all variables and presets
  - Link to detailed guide

- `README.md` (MODIFY)
  - Add mention of template capabilities
  - Link to status guide
  - Highlight composability with Unix tools

**Files Read**:
- `docs/plans/status-format-extension.md` - Existing comprehensive plan
- `internal/formatter/presets.go` - Preset specifications
- `cmd/tmux-intray/status.go` - Current implementation

**Acceptance Criteria**:

✅ **AC5.1: CLI Help Text**
- [ ] `bd status --help` shows:
  - Clear one-line description
  - Usage synopsis
  - Options explained (--format, --quiet if added)
  - List of all 13 variables with brief description
  - List of 6 preset names
  - 3-5 real-world examples
  - Link to docs/status-command-guide.md
- [ ] Help text is under 80 chars per line (terminal friendly)
- [ ] Examples are copy-paste ready

✅ **AC5.2: User Guide (docs/status-command-guide.md)**
- [ ] Introduction: What is status command, what it solves
- [ ] Quick start: 2-3 simplest examples
- [ ] Template syntax explained clearly
- [ ] All 13 variables documented:
  - Variable name
  - Type (number, string, boolean)
  - Example value
  - Use case / when to use it
- [ ] All 6 presets documented:
  - Preset name
  - Use case
  - Example output
  - When to use vs templates
- [ ] Real-world use cases section:
  - Tmux status bar with colors
  - Shell script conditional
  - Monitoring/Prometheus integration
  - Desktop notification trigger
  - Custom dashboard
- [ ] Troubleshooting section:
  - "My variable isn't working" → list valid variables
  - "Template output is wrong" → common mistakes
  - "How do I pipe to jq?" → example
- [ ] Migration guide: status-panel → bd status
  - Side-by-side comparison
  - Configuration changes needed (if any)
  - Equivalent templates for old formats

✅ **AC5.3: CLI Reference**
- [ ] docs/cli/CLI_REFERENCE.md has status command entry
- [ ] Links to detailed guide
- [ ] Quick reference table of variables
- [ ] Quick reference table of presets
- [ ] Example commands

✅ **AC5.4: README Updates**
- [ ] Status command mentioned in features
- [ ] Link to guide added
- [ ] Composability philosophy highlighted
- [ ] One simple example showing pipe to jq

✅ **AC5.5: Documentation Quality**
- [ ] No typos or grammatical errors
- [ ] Consistent terminology (template, preset, variable)
- [ ] Clear and jargon-free language
- [ ] Examples actually work (tested mentally)
- [ ] Markdown renders cleanly

**Verification Commands**:
```bash
cd /Users/cristianoliveira/other/tmux-intray

# Check help text
go run ./cmd/tmux-intray status --help

# Verify documentation files exist and are readable
ls -l docs/status-command-guide.md
cat docs/status-command-guide.md | head -50

# Check for broken links in markdown
grep -r "](.*)" docs/status-command-guide.md

# Verify examples in docs work (manual testing)
# Mentioned in wiki section
```

**Success Criteria**:
- Help text is clear and comprehensive
- User guide is self-contained and accessible to beginners
- All examples are tested and work
- Cross-references are correct
- Documentation is discoverable from README

**Known Gotchas**:
- Help text has character limits - be concise but complete
- Examples in docs must actually work or be clearly marked as pseudo-code
- Links to other docs must use correct relative paths
- Terminology must be consistent (don't switch between "template" and "format string")
- Some users may not know what ISO 8601 is - explain
- Tmux color syntax may confuse users - add examples

---

## Parallelization Strategy

### Critical Path
```
D1 (Template Engine: 2d)
  ↓
D2 (Command Integration: 1.5d) ────→ D4 (Testing: 1.5d) ────→ D5 (Docs: 1-2d)
  ↓
D3 (Backward Compat: 1d)
  ↓
D4 (Testing: 1.5d)
```

**Sequential bottleneck**: D1 must complete before D2 and D3 can start meaningfully.

### Parallel Opportunity
- **D1 (Days 1-2)**: Template engine development
- **D2 (Days 2-3)**: Command integration (starts overlapping with D1's end)
- **D3 (Days 3-4)**: Backward compatibility (parallel with D2)
- **D4 (Days 4-5)**: Integration tests (waits for D3)
- **D5 (Days 5-6)**: Documentation (waits for D4)

### Optimal Execution
1. **Day 1-2**: Engineer A works on D1 (template engine)
2. **Day 2-3**: 
   - Engineer A continues D1 completion
   - Engineer B starts D2 (command integration)
3. **Day 3-4**:
   - Engineer A starts D3 (backward compatibility)
   - Engineer B finishes D2
   - QA prepares test fixtures
4. **Day 4-5**:
   - QA Engineer works on D4 (integration tests with D1-D3 complete)
   - Tech Writer can start D5 with D2 knowledge
5. **Day 5-6**:
   - Final documentation polish
   - Integration test results inform final docs

---

## Integration Points & Contracts

### Integration Point 1: Template Engine → Command Integration

**Contract**: `internal/formatter` package exports:

```go
// Template engine interface
type TemplateEngine interface {
    Parse(template string) (*Template, error)
    Render(ctx *VariableContext) (string, error)
}

// Variable context - data passed to renderer
type VariableContext struct {
    UnreadCount    int
    TotalCount     int      // Alias
    InfoCount      int
    WarningCount   int
    ErrorCount     int
    CriticalCount  int
    HasCritical    bool
    HasError       bool
    HasWarning     bool
    HasInfo        bool
    HighestSeverity string
    PaneCount      int
    LastUpdated    string   // ISO 8601
}

// Preset registry
type PresetRegistry interface {
    IsPreset(name string) bool
    RenderPreset(name string, ctx *VariableContext) (string, error)
}
```

**Used by**: `cmd/tmux-intray/status.go` in RunE handler

**Sync point**: D2 waits for D1's interfaces to stabilize

---

### Integration Point 2: Command Integration → Backward Compatibility

**Contract**: `status-panel-cmd.go` calls refactored data extraction, then uses `TemplateEngine`

**Data flow**:
```
status-panel invocation
  ↓
Query notifications (existing code)
  ↓
Build VariableContext (new helper in internal/format)
  ↓
Select preset based on old format name
  ↓
Render via TemplateEngine
  ↓
Output
```

**Used by**: `cmd/tmux-intray/status-panel-cmd.go`

**Sync point**: D3 refactors D2's output logic to use shared template engine

---

### Integration Point 3: All Code → Testing

**Contract**: All D1, D2, D3 code is instrumented for testability

**Requirements**:
- Exported interfaces in formatter package (for mocking)
- Status command accepts injected formatter
- Data structures are deterministic

**Used by**: `tests/commands/status.bats` and `cmd/tmux-intray/status_test.go`

**Sync point**: D4 needs all code stable and callable

---

### Integration Point 4: Everything → Documentation

**Contract**: Features documented match actual behavior

**Requirements**:
- Help text matches CLI implementation
- Variables in docs match variables in code
- Presets documented match actual presets
- Examples actually work

**Used by**: User-facing docs in `docs/status-command-guide.md`

**Sync point**: D5 waits for D4 (tests prove behavior)

---

## File Overlap Analysis

### Zero Overlap Files (Can work independently)
- `internal/formatter/template.go` (D1 only)
- `internal/formatter/variables.go` (D1 only)
- `internal/formatter/presets.go` (D1 only)
- `internal/formatter/*_test.go` (D1 only)
- `tests/integration/formatter_e2e.bats` (D4 only)
- `docs/status-command-guide.md` (D5 only)

### Single-Touch Files (Modified once)
- `cmd/tmux-intray/status.go` (D2 main, D5 help text)
  - Strategy: D2 implements core, D5 only updates help/Long field
- `cmd/tmux-intray/status_test.go` (D2 main, D4 additional tests)
  - Strategy: D2 adds basic tests, D4 adds integration tests
- `docs/cli/CLI_REFERENCE.md` (D5 only)
- `README.md` (D5 only)

### Potential Conflict Points (needs sequencing)
1. **`cmd/tmux-intray/status.go` help text**:
   - D2 writes basic help
   - D5 enhances with examples
   - **Solution**: D2 uses placeholder help, D5 fills in details

2. **`internal/format/status.go`**:
   - D1 reads for patterns
   - D3 potentially adds helpers
   - **Solution**: D3 adds non-breaking helper functions

3. **`cmd/tmux-intray/status-panel-cmd.go`**:
   - D3 refactors completely
   - **Solution**: D3 is sequential after D2 complete

---

## Risk Assessment & Mitigation

### Risk 1: Template Engine Complexity
**Risk Level**: MEDIUM  
**Impact**: If D1 overruns, delays all downstream work  
**Mitigation**:
- Start with regex-based simple parser (not full templating language)
- Focus on substitution, not conditionals (per design decision)
- Have fallback: hardcoded parsing if reflection becomes complex

### Risk 2: Data Model Misalignment
**Risk Level**: LOW  
**Impact**: D2-D3 integration issues  
**Mitigation**:
- Define `VariableContext` struct in D1 first
- D2-D3 only consume this struct, don't modify
- Clear interface contract in code

### Risk 3: Backward Compatibility Breaks
**Risk Level**: HIGH (if not careful)  
**Impact**: User configurations fail  
**Mitigation**:
- D3 must include pixel-perfect output comparison tests
- D4 explicitly tests old behavior unchanged
- Run both old and new paths in parallel during D3, compare outputs

### Risk 4: Test Coverage Gaps
**Risk Level**: MEDIUM  
**Impact**: Production bugs in edge cases  
**Mitigation**:
- D1 hits 100% unit test coverage (simple code)
- D4 focuses on integration and edge cases
- Use table-driven tests for variable combinations

### Risk 5: Documentation Staleness
**Risk Level**: LOW  
**Impact**: Users can't use feature  
**Mitigation**:
- D5 happens last, closest to code
- Examples tested before publication
- Links point to stable code

### Risk 6: File Merge Conflicts
**Risk Level**: MEDIUM  
**Impact**: Integration friction  
**Mitigation**:
- Clear file ownership per deliverable
- Single-touch files for most work
- Help text marked as "expandable placeholder" for D5

---

## Deliverable Checklist & Handoff Criteria

### D1 Completion Checklist (unblocks D2, D3)
- [ ] `internal/formatter/` package created with 3 modules
- [ ] All 13 template variables implemented and tested
- [ ] All 6 presets registered
- [ ] Interfaces exported and documented
- [ ] >80% unit test coverage
- [ ] No linter warnings
- [ ] Code review approved
- **Handoff**: Push to branch, notify D2 and D3 owners

### D2 Completion Checklist (unblocks D3, D4)
- [ ] `--format` flag added to status command
- [ ] Command integrates with formatter package
- [ ] All presets work end-to-end
- [ ] Custom templates work
- [ ] Help text drafted (may be expanded by D5)
- [ ] Basic command tests pass
- [ ] Backward compatible with existing format names
- **Handoff**: Code compiles, basic tests pass, ready for D3 integration

### D3 Completion Checklist (unblocks D4)
- [ ] status-panel delegates to new formatter
- [ ] Output pixel-perfect identical to original
- [ ] No deprecation warnings
- [ ] Old config keys still work
- [ ] All legacy tests pass
- [ ] Data extraction helpers created
- **Handoff**: Legacy tests 100% pass, no regressions

### D4 Completion Checklist (unblocks D5)
- [ ] All format tests passing
- [ ] All template variable tests passing
- [ ] All error case tests passing
- [ ] Backward compatibility tests passing
- [ ] End-to-end integration tests passing
- [ ] Coverage report >80%
- [ ] No flaky tests
- **Handoff**: Test suite stable, all CI green

### D5 Completion Checklist (final delivery)
- [ ] Help text comprehensive with examples
- [ ] User guide complete and clear
- [ ] CLI reference updated
- [ ] README updated
- [ ] No broken links
- [ ] Examples tested/validated
- [ ] Spelling and grammar checked
- **Handoff**: Documentation PR ready

---

## Success Metrics

### Phase 1 Delivery Success
- [ ] All 5 deliverables completed on schedule
- [ ] No regressions to existing status-panel functionality
- [ ] `bd status --format` works with 5+ different format strings
- [ ] Coverage >80% across all new code
- [ ] Code review approval from tech lead
- [ ] Zero critical bugs in initial testing

### User Success
- [ ] Users can create custom templates without trial-and-error (test with 3 beta users)
- [ ] Help text answers 80% of FAQ questions
- [ ] Migration from status-panel is frictionless
- [ ] Examples in docs are accurate and useful

### Code Quality
- [ ] No new linter warnings introduced
- [ ] New code follows project style guide
- [ ] Interfaces are clean and stable
- [ ] Test code is readable and maintainable

---

## Communication & Sync Points

### Weekly Sync
- **Monday morning**: Review D1 progress, unblock D2/D3 if needed
- **Wednesday**: D2/D3 integration point check
- **Friday**: D4 test results, D5 draft review

### Async Communication
- Use GitHub PR comments for implementation details
- Slack for blockers only
- Code review as primary feedback mechanism

### Documentation
- Keep this file updated as timelines shift
- Comment code with rationale (helps other lanes)
- Log decisions in commit messages

---

## Appendix: Detailed File Reference

### Files to Create (9 total)
```
internal/formatter/                 # NEW PACKAGE
├── template.go                      # Template parser (150 lines)
├── template_test.go                 # Parser tests (200 lines)
├── variables.go                     # Variable resolver (100 lines)
├── variables_test.go                # Resolver tests (150 lines)
├── presets.go                       # Preset registry (80 lines)
├── presets_test.go                  # Preset tests (100 lines)

docs/
└── status-command-guide.md          # User guide (200-300 lines)

tests/
└── integration/
    └── formatter_e2e.bats           # Optional E2E tests (100 lines)
```

### Files to Modify (6 total)
```
cmd/tmux-intray/
├── status.go                        # Add --format flag (50 lines)
├── status_test.go                   # Add format tests (100 lines)
└── status-panel-cmd.go              # Refactor to use formatter (30 lines changes)

internal/
└── format/
    └── status.go                    # Add VariableContext builder (50 lines)

docs/
├── cli/CLI_REFERENCE.md             # Add status section (30 lines)
└── README.md                        # Add mention (10 lines)
```

### Estimated Total New Code
- **Core logic**: ~450 lines (D1 formatter)
- **Command integration**: ~150 lines (D2, D3)
- **Tests**: ~500 lines (D1, D4)
- **Documentation**: ~250 lines (D5)
- **Total**: ~1,350 lines

---

**Document Status**: Ready for Implementation  
**Last Updated**: February 26, 2026  
**Next Step**: Create `bd` tasks for each deliverable
