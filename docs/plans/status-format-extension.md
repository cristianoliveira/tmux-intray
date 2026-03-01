# BD Status Command Format Extension Plan

**Document Version:** 1.0
**Date Created:** February 2026
**Status:** Planning Phase
**Author:** Implementation Team

---

## Executive Summary

This document outlines the plan to extend the `bd status` command with a `--format` flag that supports custom template-based formatting. The goal is to decouple the command from tmux-specific output formatting, enabling it to work seamlessly with other CLI tools via pipes, jq, xargs, and other composable utilities.

Currently, the status display command is tightly coupled to tmux output format. This plan enables creating a general-purpose status query API that any tool can use.

---

## Table of Contents

1. [Problem Statement](#problem-statement)
2. [Solution & Design](#solution--design)
3. [Implementation Plan](#implementation-plan)
4. [Command Specification](#command-specification)
5. [Acceptance Criteria](#acceptance-criteria)
6. [Success Metrics](#success-metrics)
7. [Timeline & Dependencies](#timeline--dependencies)
8. [Open Decisions](#open-decisions)
9. [Code Organization](#code-organization)

---

## Problem Statement

### Current Situation

The tmux-intray project currently provides a `status` command designed specifically for tmux's status bar. While functional, this implementation has several limitations:

1. **Tight Tmux Coupling**: Output format is hardcoded for tmux (uses `#[fg=...]` color syntax and tmux-specific formatting)
2. **Limited Flexibility**: Cannot easily use notification data in other contexts (monitoring systems, shell scripts, dashboards, HTTP APIs)
3. **Template Inflexibility**: To change output format, users must modify code or create wrapper scripts
4. **Tool Composition Issues**: Cannot pipe output directly to `jq`, `xargs`, or other Unix tools for further processing
5. **Maintenance Overhead**: Each new format requirement requires code changes
6. **User Pain Points**:
   - Integrators who want to use notification data in dashboards can't access structured data
   - Tool builders need custom scripts to wrap the command
   - Monitoring systems can't query notification status without shell scripting

### Impact

- **Users**: Can't customize notification display without editing code
- **Integrators**: Creating custom dashboards requires wrapper scripts and field parsing
- **Tool Builders**: Building tools that depend on notification status is cumbersome
- **Project Maintainers**: Each format request requires code review and maintenance

### Why Now

1. **Growing Integration Demand**: Users are increasingly integrating tmux-intray with other tools (monitoring dashboards, custom status bars, automation scripts)
2. **Composability Philosophy**: The project follows Unix philosophy of composable tools
3. **Foundation Ready**: The codebase already has query and formatting infrastructure that can be generalized

---

## Solution & Design

### Core Insight

**Shift from "tmux status bar formatter" to "notification data query API".**

Instead of a single hardcoded format, provide a flexible template-based system where users can specify exactly what data they want, in what format.

### Data Model

The status command will expose the following queryable variables:

```
${unread-count}        # Total number of active notifications
${total-count}         # Same as unread-count (alias)
${info-count}          # Number of info-level notifications
${warning-count}       # Number of warning-level notifications
${error-count}         # Number of error-level notifications
${critical-count}      # Number of critical-level notifications
${has-critical}        # Boolean: "true" if any critical notifications exist
${has-error}           # Boolean: "true" if any error notifications exist
${has-warning}         # Boolean: "true" if any warning notifications exist
${has-info}            # Boolean: "true" if any info notifications exist
${highest-severity}    # Highest severity level present (critical, error, warning, info)
${pane-count}          # Number of panes with active notifications
${last-updated}        # ISO 8601 timestamp of last update
```

### Output Format Types

The `--format` flag will support several modes:

1. **Template Mode** (default flexible approach)
   - Custom templates using `${variable}` syntax
   - Example: `--format='${unread-count} notifications'`
   - Example: `--format='#[fg=red]${critical-count}#[default] critical'`

2. **Preset Formats** (convenience shortcuts)
   - `compact` - Minimal output (default for backward compatibility)
   - `detailed` - Full breakdown by level
   - `json` - Structured JSON output
   - `count-only` - Just the number
   - `panes` - Breakdown by pane location
   - `levels` - Breakdown by severity level

3. **No Data Output** (when useful)
   - Exit code indicates status (0 = has notifications, 1 = no notifications)
   - Useful for shell conditionals: `if bd status --quiet; then ...`

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     bd status command                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Input:  --format="<template or preset>"                         │
│          [various filters: --level, --session, etc]              │
│                                                                   │
├─────────────────────────────────────────────────────────────────┤
│                    Notification Data Layer                        │
│  (queries storage for active notifications)                      │
├─────────────────────────────────────────────────────────────────┤
│              StatusFormatter Package (new)                        │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ • Template Parser (parse ${variable} syntax)             │   │
│  │ • Variable Resolver (map variable names to values)       │   │
│  │ • Preset Registry (compact, detailed, json, etc)         │   │
│  │ • Format Renderer (substitute variables in template)     │   │
│  └──────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────┤
│                          Output                                   │
│  (rendered template, preset output, or exit code)                │
└─────────────────────────────────────────────────────────────────┘
```

### Real-World Usage Examples

#### Example 1: Simple Count in Shell Script
**Use Case**: Script needs to check if there are notifications

```bash
# Get count and use in logic
count=$(bd status --format='${unread-count}')
if [ "$count" -gt 0 ]; then
  notify-send "You have $count notifications"
fi
```

#### Example 2: Tmux Status Bar (Compact)
**Use Case**: Display in tmux status bar with colors

```bash
# Use template that includes tmux color codes
bd status --format='#[fg=red]${critical-count}#[default] #[fg=yellow]${warning-count}#[default]'

# Output: 0 2 (if 0 critical and 2 warnings)
```

#### Example 3: Monitoring Dashboard Integration
**Use Case**: Ingest into monitoring system (Prometheus, Grafana, etc)

```bash
# Get structured data for metrics export
bd status --format=json | jq '.critical, .error' | awk '{ print "notifications{level=\"critical\"} " $1 }'

# Output: notifications{level="critical"} 5
```

#### Example 4: Desktop Notification
**Use Case**: Trigger desktop notifications with summary

```bash
# Check if there are critical items, show custom message
if bd status --format='${critical-count}' | grep -v '^0$' > /dev/null; then
  severity=$(bd status --format='${highest-severity}')
  notify-send "Alert" "Highest severity: $severity"
fi
```

#### Example 5: Conditional tmux Binding
**Use Case**: Color status bar based on notification severity

```tmux
# In tmux.conf
set -g status-right "#(bd status --format='#[fg=#{?#{!#{ $(bd status --format='${has-critical}'),0},green,red}}]●#[default]') %H:%M"

# Or simpler with preset:
set -g status-right "#(bd status) %H:%M"
```

#### Example 6: Complex Template for Custom Display
**Use Case**: Detailed custom output for status line

```bash
bd status --format='📋 ${unread-count} [i:${info-count} w:${warning-count} e:${error-count} c:${critical-count}]'

# Output: 📋 5 [i:1 w:2 e:2 c:0]
```

---

## Implementation Plan

### Phase 1: Core Implementation (Effort: 2-3 days)

Build the foundation for template-based formatting.

**Files to Create:**
- `internal/formatter/template.go` - Template parsing and variable substitution engine
- `internal/formatter/template_test.go` - Unit tests for template engine
- `cmd/tmux-intray/status.go` - Extended with --format flag support
- `cmd/tmux-intray/status_test.go` - Tests for status command

**Files to Modify:**
- `internal/format/status.go` - Add helper functions to extract data needed for templates

**Key Implementation Tasks:**

1. **Create StatusFormatter Package**
   - Parse templates with `${variable}` syntax
   - Validate variable names
   - Resolve variables to values from notification data
   - Handle edge cases (missing variables, empty data)
   - Return error messages for invalid templates

2. **Extend bd status Command**
   - Add `--format` flag accepting template strings or preset names
   - Add validation for format strings
   - Add help text with variable list and examples
   - Support both templates and presets

3. **Data Collection**
   - Gather all necessary data (unread count, level breakdown, panes, severity)
   - Create a clean data structure to pass to formatter
   - Ensure data is accurate and consistent

#### Acceptance Criteria for Phase 1

**AC1.1: Template Parser**
```gherkin
Feature: Template Parsing
  Scenario: Parse valid template
    Given a template "${unread-count} notifications"
    When parsed
    Then it identifies one variable "unread-count"
    And no parse errors occur

  Scenario: Reject invalid variable names
    Given a template "${invalid-var}"
    When parsed and validated
    Then an error message appears listing valid variables
    And command exits with code 1

  Scenario: Handle escaped braces
    Given a template with literal "$${unread-count}"
    When parsed
    Then the output contains literal "${unread-count}"
```

**AC1.2: Variable Substitution**
```gherkin
Feature: Variable Substitution
  Scenario: Substitute multiple variables
    Given a template "${critical-count} critical, ${warning-count} warnings"
    And 2 critical and 3 warning notifications
    When substituted
    Then output is "2 critical, 3 warnings"

  Scenario: Handle missing variables gracefully
    Given a template "${unread-count}"
    And no notifications
    When substituted
    Then output is "0"
```

**AC1.3: Preset Formats**
```gherkin
Feature: Preset Formats
  Scenario: Compact format output
    Given 2 critical, 1 warning, 0 error, 0 info
    When --format=compact
    Then output contains icon and total count
    And output is single-line

  Scenario: JSON format output
    Given 2 critical, 1 warning
    When --format=json
    Then output is valid JSON
    And JSON contains "critical": 2, "warning": 1

  Scenario: Count-only format
    Given 5 total notifications
    When --format=count-only
    Then output is exactly "5"
```

**AC1.4: Command Integration**
```gherkin
Feature: Status Command with Format Flag
  Scenario: Custom template format
    Given template "--format='[${critical-count}c ${warning-count}w]'"
    When bd status is run
    Then output matches template
    And exit code is 0

  Scenario: Invalid format string
    Given invalid template "--format='${nosuchvar}'"
    When bd status is run
    Then error message lists valid variables
    And exit code is 1 (failure)

  Scenario: Help shows examples
    When bd status --help
    Then output includes template examples
    And output includes preset names
```

---

### Phase 2: Backward Compatibility (Effort: 1 day)

Ensure `status-panel` continues to work without changes to user configurations.

**Files to Modify:**
- `cmd/tmux-intray/status-panel-cmd.go` - Delegate to new formatter
- `internal/status/status.go` - Update to use new formatter

**Implementation Tasks:**

1. **Refactor status-panel to use new formatter**
   - Create internal function that maps old format names to new templates
   - Ensure output is pixel-perfect identical to current behavior
   - Add deprecation path (optional warning, no removal yet)

2. **Maintain configuration compatibility**
   - Old config keys still work
   - New `status_format` key can be used for presets
   - Environment variable `TMUX_INTRAY_STATUS_FORMAT` still works

3. **Document migration path**
   - Create migration guide for users wanting to switch to `bd status`
   - Show equivalent templates for each old format

#### Acceptance Criteria for Phase 2

**AC2.1: status-panel Compatibility**
```gherkin
Feature: Backward Compatibility
  Scenario: Compact format unchanged
    Given existing tmux setup using status-panel with compact format
    When tmux status bar is refreshed
    Then output is identical to current behavior
    And no configuration changes needed

  Scenario: Legacy config still works
    Given config.toml with status_format = "detailed"
    When status-panel runs
    Then output uses detailed format
    And no deprecation warning appears (Phase 2 choice)
```

**AC2.2: Configuration Migration**
```gherkin
Feature: Configuration Continuity
  Scenario: Using bd status instead of status-panel
    Given user reads migration guide
    When user updates tmux.conf to use "bd status --format='compact'"
    Then tmux status bar works identically
    And no manual format adjustment needed
```

---
### Phase 3: Documentation (Effort: 1-2 days)

Make the feature discoverable and understandable.

**Files to Create:**
- `docs/status-command-guide.md` - User guide for status command and templates
- Update `docs/cli/CLI_REFERENCE.md` - Add status command documentation
- Update `README.md` - Mention template capabilities

**Documentation Tasks:**

1. **CLI Help Text**
   - List all available variables
   - Show 3-5 real-world examples
   - Document preset formats
   - Include troubleshooting

2. **User Guide**
   - Explain template syntax
   - Document all variables with examples
- Show common use cases (tmux, monitoring, scripts)
   - Provide migration path from status-panel

3. **Integration Guides**
   - How to integrate with popular monitoring tools
   - How to use with custom shell prompts
   - How to pipe to jq for JSON processing

4. **README Updates**
   - Mention composability with other tools
   - Add status command to quick reference

#### Acceptance Criteria for Phase 3

**AC3.1: CLI Help Quality**
```gherkin
Feature: Help Text
  Scenario: Status command help is complete
    When bd status --help
    Then output includes:
      | Item                              |
      | List of available variables       |
      | Preset format names               |
      | At least 2 real-world examples    |
      | Link to detailed docs             |

  Scenario: Variables are clearly explained
    When reading --help output
    Then each variable has 1-2 line description
    And examples show expected values
```

**AC3.2: Documentation Completeness**
```gherkin
Feature: User Guide
  Scenario: New user can get started
    Given a user wants to use custom template
    When they read docs/status-command-guide.md
    Then they can:
      | Action                                |
      | Understand template syntax            |
      | Find list of available variables      |
      | Copy-paste working example            |
      | Understand what output means          |

  Scenario: Integration guide exists
    Given user wants to use with Prometheus
    When they search for integration guide
    Then they find docs/status-command-guide.md#integration
    And it includes working example
```

---

### Phase 4: Validation (Optional, Future)

**Timeline:** Post-release (if needed)

- QA testing with diverse template formats
- User testing with integrators
- Performance benchmarks
- Release management

---

## Command Specification

### Syntax

```
bd status [OPTIONS]
```

### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `--format` | string | `compact` | Output format: template string or preset name |
| `--quiet` | bool | false | Exit silently; return code only (0=has notifications, 1=none) |
| `--help` | - | - | Show help message |

### Preset Formats

| Preset | Output | Example |
|--------|--------|---------|
| `compact` | Single line with icon and total | 🔔 5 |
| `detailed` | Level breakdown | i:1 w:2 e:2 c:0 |
| `count-only` | Just the number | 5 |
| `json` | Valid JSON structure | `{"active":5,"info":1,...}` |
| `levels` | Key:value by level | `info:1\nwarning:2\nerror:2\ncritical:0` |
| `panes` | Breakdown by pane | `session:window:pane:5` |

### Template Variables

| Variable | Type | Example | Notes |
|----------|------|---------|-------|
| `${unread-count}` | number | 5 | Total active notifications |
| `${total-count}` | number | 5 | Alias for unread-count |
| `${info-count}` | number | 1 | Info-level only |
| `${warning-count}` | number | 2 | Warning-level only |
| `${error-count}` | number | 2 | Error-level only |
| `${critical-count}` | number | 0 | Critical-level only |
| `${has-critical}` | bool | false | true/false string |
| `${has-error}` | bool | true | true/false string |
| `${has-warning}` | bool | true | true/false string |
| `${has-info}` | bool | true | true/false string |
| `${highest-severity}` | string | error | critical/error/warning/info |
| `${pane-count}` | number | 3 | Panes with notifications |
| `${last-updated}` | timestamp | 2026-02-26T... | ISO 8601 format |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success; if `--quiet` flag: has notifications |
| 1 | Syntax error in template or format invalid; if `--quiet` flag: no notifications |
| 2 | System error (tmux not running, storage error) |

### Examples

```bash
# Show total count
$ bd status --format='${unread-count}'
5

# Show detailed breakdown
$ bd status --format='[i:${info-count} w:${warning-count} e:${error-count} c:${critical-count}]'
[i:1 w:2 e:2 c:0]

# Tmux color output
$ bd status --format='#[fg=red]${critical-count}#[default] critical'
0 critical

# JSON output
$ bd status --format=json
{
  "active": 5,
  "info": 1,
  "warning": 2,
  "error": 2,
  "critical": 0,
  "panes": {
    "session:window:pane": 5
  }
}

# Preset format (compact)
$ bd status
🔔 5

# Use in shell script
$ if [ "$(bd status --format='${critical-count}')" -gt 0 ]; then
>   echo "Critical notifications found!"
> fi
Critical notifications found!

# Pipe to jq for processing
$ bd status --format=json | jq '.critical + .error'
2

# Exit code only (for conditionals)
$ bd status --quiet && echo "You have notifications"
You have notifications
```

---

## Acceptance Criteria

### Phase 1 Acceptance Criteria

✅ **Template Engine Works**
- [ ] Parser correctly identifies variables in templates
- [ ] Parser rejects invalid variable names with clear error
- [ ] Variable substitution works for all documented variables
- [ ] Handles edge cases: empty data, special characters, large numbers

✅ **Format Flag Integration**
- [ ] `--format` flag accepted by `bd status`
- [ ] Preset formats work: compact, detailed, json, count-only, levels, panes
- [ ] Custom templates work with `${variable}` syntax
- [ ] Invalid format strings produce helpful error messages

✅ **Data Accuracy**
- [ ] Counts match actual notification data
- [ ] Severity levels correct (info, warning, error, critical)
- [ ] Timestamps accurate and in ISO 8601 format

✅ **Test Coverage**
- [ ] Unit tests for template parser (positive and negative cases)
- [ ] Unit tests for variable resolver
- [ ] Integration tests for `bd status --format`
- [ ] At least 80% code coverage for new formatter package

### Phase 2 Acceptance Criteria

✅ **Backward Compatibility**
- [ ] `status-panel` command works unchanged
- [ ] All old format names produce identical output
- [ ] Old configuration keys still work
- [ ] No deprecation warnings (Phase 2 choice to be confirmed)

✅ **Migration Path**
- [ ] Existing tmux.conf configurations work without modification
- [ ] Users can optionally switch to `bd status` by updating tmux.conf
- [ ] No data loss or behavior change in migration
### Phase 3 Acceptance Criteria

✅ **Documentation Quality**
- [ ] `bd status --help` shows variable list
- [ ] `bd status --help` includes 3+ real-world examples
- [ ] `docs/status-command-guide.md` exists and is comprehensive
- [ ] Integration guide includes Prometheus/monitoring example
- [ ] README mentions template capabilities

✅ **User Experience**
- [ ] New user can create custom template from docs without trial-and-error
- [ ] Examples are copy-paste ready
- [ ] Help text is clear and jargon-free

---

## Success Metrics

### Primary Metrics

| Metric | Target | Verification |
|--------|--------|--------------|
| `bd status --format` is usable | Any custom template works | Manual testing with 5+ unique templates |
| No regressions | `status-panel` output unchanged | Diff current vs Phase 2 output |
| Help text quality | Users can understand usage | Have 2-3 beta users try without docs |

### Secondary Metrics

| Metric | Target | Verification |
|--------|--------|--------------|
| Backward compatibility | Zero configuration changes needed | Configuration file compatibility test |
| Documentation completeness | New user can use feature | User reads docs and creates 1 template |
| Code quality | No technical debt | Code review, test coverage >80% |

### Validation Checklist

Before release, verify:

- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] `status-panel` output pixel-perfect identical
- [ ] Documentation compiles and renders correctly
- [ ] Help text is clear and complete
- [ ] No deprecation warnings appear (Phase 2 confirmation)
- [ ] Examples are tested and work
- [ ] Code follows project style guidelines
- [ ] No new linter warnings

---

## Timeline & Dependencies

### Phase Timeline

| Phase | Duration | Start | End | Dependencies |
|-------|----------|-------|-----|--------------|
| **Phase 1: Core Implementation** | 2-3 days | Mon | Wed | None |
| **Phase 2: Backward Compatibility** | 1 day | Thu | Thu | Phase 1 complete |
| **Phase 3: Documentation** | 1-2 days | Fri | Fri-Mon | Phase 1 + 2 complete |
| **Phase 4: Validation** | TBD | Post-release | TBD | All phases complete |

### Critical Path

```
Phase 1 (Core) ──→ Phase 2 (Compat) ──→ Phase 3 (Docs) ──→ Phase 4 (QA)
    3 days              1 day              2 days           TBD
```

**Total Effort:** ~6 days for Phases 1-3 (release-ready)

### Dependencies

**External Dependencies:**
- None (uses existing Go stdlib and project dependencies)

**Internal Dependencies:**
- `internal/storage` - For notification queries
- `internal/format` - For existing format utilities
- `cmd/tmux-intray` - Where status command lives

**Team Dependencies:**
- Code review (1-2 hours)
- Documentation review (30 minutes)
- Testing with users (optional, Phase 4)

### Risk Matrix

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|-----------|
| Template parser complexity | Medium | Low | Start with simple regex-based parser; enhance if needed |
| Breaking `status-panel` | High | Low | Comprehensive backward compat tests before merge |
| Users confused by templates | Medium | Medium | Provide 5+ real-world examples in docs |
| Performance regression | Medium | Low | Benchmark template rendering vs hardcoded |
| Scope creep (color support) | Medium | Medium | Document as future work; don't block release |

---

## Open Decisions

### Decision 1: Deprecation Warning for status-panel
**Question:** Should the `status-panel` command emit a deprecation warning?

**Options:**
- **A) No warning** (Recommended for Phase 2)
  - Pro: Minimal disruption to existing users
  - Con: Unclear long-term direction
  - Timeline: Supports immediate release without user friction

- **B) Optional warning** via environment variable
  - Pro: Users can opt-in to see path forward
  - Con: Additional complexity
  - Timeline: Same-release feasible

- **C) Deprecation warning from day 1**
  - Pro: Clear signal to users
  - Con: May confuse existing users who are happy with status-panel
  - Timeline: Better long-term, may require communication plan

**Recommendation:** Choose **A (No warning)** for Phase 2. Revisit in Phase 4 with user feedback.

**Decision Made:** [TO BE DECIDED BY TEAM]

---
### Decision 2: Default Format (preset vs template syntax)
**Question:** What should the default `--format` be if the flag is omitted?

**Options:**
- **A) Preset "compact"** (Recommended)
  - Pro: Backward compatible with status-panel behavior
  - Con: Doesn't leverage new template power
  - Default: `bd status` → same as `bd status --format=compact`

- **B) Accept environment variable override**
  - Pro: Users can customize without CLI flag
  - Con: Added complexity
  - Timeline: Feasible, follows project philosophy

- **C) Always require explicit --format flag**
  - Pro: Forces deliberate choice
  - Con: Breaking change, friction for simple use case
  - Timeline: Not recommended for Phase 1

**Recommendation:** Choose **A (Preset "compact")** with optional **B** support.

**Decision Made:** [TO BE DECIDED BY TEAM]

---### Decision 4: Template Syntax for Special Formatting
**Question:** Should templates support conditionals or filters (e.g., color output based on level)?

**Examples of what we're asking about:**
```bash
# Conditional formatting (complex)
bd status --format='${if ${has-critical}|#[fg=red]${critical-count}#[default]|0}'

# vs just letting user handle it (simpler)
# User would pipe to jq or use shell conditionals
```

**Options:**
- **A) Simple template substitution only** (Recommended for Phase 1)
  - Pro: Simple to implement, easy to understand
  - Con: Less powerful, requires external tools for complex logic
  - Example: `--format='${critical-count} critical'`

- **B) Add conditional support** (Phase 1.5)
  - Pro: More powerful
  - Con: Complexity increases significantly
  - Timeline: Post-release enhancement

- **C) Add color filter support** (Phase 1.5)
  - Pro: Useful for tmux users
  - Con: Scope creep
  - Timeline: Possibly Phase 3

**Recommendation:** Choose **A** for Phase 1 (simple syntax). Revisit in Phase 4 based on user feedback.

**Decision Made:** [TO BE DECIDED BY TEAM]

---

### Decision 5: Should --format Accept File Paths?
**Question:** Should `--format=@file.txt` read template from a file?

**Options:**
- **A) CLI-only templates** (Recommended)
  - Pro: Simple, predictable behavior
  - Con: Long templates clunky on command line
  - Timeline: Phase 1 friendly

- **B) Support file paths** (Phase 1.5)
  - Pro: Useful for complex templates
  - Con: File I/O adds complexity
  - Timeline: Post-release feature

**Recommendation:** Choose **A** for Phase 1. Users can create shell aliases or functions for complex templates.

**Decision Made:** [TO BE DECIDED BY TEAM]

---

## Code Organization

### Package Structure

```
internal/
├── formatter/                      # NEW: Template formatting
│   ├── template.go                 # Template parser and renderer
│   ├── template_test.go            # Unit tests
│   ├── variables.go                # Variable resolver
│   ├── variables_test.go           # Unit tests
│   └── presets.go                  # Preset format definitions
│
├── format/                         # EXISTING: Format utilities
│   ├── status.go                   # [MODIFY] Add helper functions
│   └── status_test.go
│
└── status/                         # EXISTING: Status business logic
    ├── status.go                   # [MODIFY] Update if needed
    └── status_test.go

cmd/tmux-intray/
├── status.go                       # [MODIFY] Add --format flag support
├── status_test.go                  # [MODIFY] Add format tests
├── status-panel-cmd.go             # [MODIFY] Delegate to formatter
└── status-panel-core.go            # [MODIFY] Update if needed

docs/
└── plans/
    ├── status-format-extension.md  # THIS FILE
    └── (other planning docs)

docs/
├── status-command-guide.md         # NEW: User guide
└── cli/
    └── CLI_REFERENCE.md            # [MODIFY] Add status documentation
```

### Module Design

#### formatter/template.go

**Responsibilities:**
- Parse template strings for variables
- Validate variable names
- Render templates with variable substitution

**Public API:**
```go
type TemplateEngine interface {
    Parse(template string) (*Template, error)
    Render(ctx *VariableContext) (string, error)
}

type VariableContext struct {
    UnreadCount    int
    InfoCount      int
    WarningCount   int
    ErrorCount     int
    CriticalCount  int
    HighestSeverity string
    // ... more fields
}
```

#### formatter/variables.go

**Responsibilities:**
- Map variable names to context values
- Handle type conversions (int to string, bool to "true"/"false")
- Validate known variables

**Public API:**
```go
type VariableResolver interface {
    Resolve(varName string, ctx *VariableContext) (string, error)
    ValidVariables() []string
}
```

#### formatter/presets.go

**Responsibilities:**
- Define built-in formats (compact, detailed, json, etc.)
- Map preset names to templates or custom renderers

**Public API:**
```go
type PresetRegistry interface {
    IsPreset(name string) bool
    GetTemplate(name string) string
    RenderPreset(name string, ctx *VariableContext) (string, error)
}
```

#### cmd/tmux-intray/status.go

**Changes:**
- Add `--format` flag
- Parse format value as preset or template
- Use formatter to render output
- Keep backward compatibility with existing tests

**Example Flow:**
```
flag: --format='${critical-count} critical'
  ↓
Parse as template (not a preset)
  ↓
Parse template, get variables needed
  ↓
Collect notification data
  ↓
Build VariableContext
  ↓
Render template with context
  ↓
Output: "2 critical"
```

### File Modification Summary

| File | Type | Change | Reason |
|------|------|--------|--------|
| `internal/formatter/template.go` | Create | New package | Core formatting logic |
| `internal/formatter/variables.go` | Create | New module | Variable resolution |
| `internal/formatter/presets.go` | Create | New module | Preset management |
| `internal/formatter/*_test.go` | Create | Test files | Test coverage |
| `cmd/tmux-intray/status.go` | Modify | Add `--format` support | Feature implementation |
| `cmd/tmux-intray/status-panel-cmd.go` | Modify | Use new formatter | Backward compatibility |
| `internal/format/status.go` | Modify | Add helpers | Support new features |
| `docs/status-command-guide.md` | Create | User guide | Documentation |
| `docs/cli/CLI_REFERENCE.md` | Modify | Add status docs | API documentation |
| `README.md` | Modify | Add mention | Feature visibility |

### Testing Strategy

**Unit Tests** (in `internal/formatter/`):
- Template parsing (valid and invalid)
- Variable substitution
- Preset lookups
- Error handling

**Integration Tests** (in `cmd/tmux-intray/` tests):
- `bd status --format=<preset>`
- `bd status --format='<template>'`
- Exit codes
- Output accuracy with various notification states

**Regression Tests**:
- `status-panel` output unchanged
- Existing functionality unaffected

**Test Organization:**
```
tests/
├── commands/
│   └── status.bats              # Bats integration tests
└── unit/
    └── formatter/
        ├── template_test.go
        ├── variables_test.go
        └── presets_test.go
```

**Coverage Goal:** >80% for new code

---

## Appendix: Related Documentation

- [Project Philosophy](../philosophy.md) - Design principles
- [CLI Reference](../cli/CLI_REFERENCE.md) - Current command docs
- [Design: Go Package Structure](../design/go-package-structure.md) - Code organization
- [Development Guide](../../DEVELOPMENT.md) - Contributing guidelines

---

## Document Change History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-02-26 | Team | Initial comprehensive plan |

---

## Sign-Off

**Document Review:**
- [ ] Technical Lead: _____________ Date: _______
- [ ] Product Owner: _____________ Date: _______
- [ ] QA Lead: _____________ Date: _______

**Ready to Implement:** [YES / NO]

---

**For questions or clarifications:** See "Open Decisions" section above.
