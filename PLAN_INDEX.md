# Status Format Extension Plan - Master Index

**Epic**: Extend `bd status` with `--format` flag for custom template formatting

**Plan Status**: ✅ COMPLETE & READY FOR EXECUTION

---

## 📚 Documentation Files

### 1. PARALLEL_WORKSTREAMS.md (PRIMARY REFERENCE)
**Location**: `/Users/cristianoliveira/other/tmux-intray/PARALLEL_WORKSTREAMS.md`  
**Size**: 29 KB, 1000+ lines  
**Audience**: All team members (engineering, QA, tech writing)  

**Contains**:
- Executive summary of 5 parallel deliverables
- Detailed breakdown of each deliverable (D1-D5)
- Acceptance criteria per deliverable
- File impact analysis (files to create/modify)
- Integration points and contracts
- Risk assessment and mitigations
- Timeline and parallelization strategy
- Verification checklist
- Success metrics

**Start Here**: Everyone should read this once before starting work

### 2. PLAN_INDEX.md (THIS FILE)
**Location**: `/Users/cristianoliveira/other/tmux-intray/PLAN_INDEX.md`  
**Audience**: Quick navigation reference  

**Contains**:
- Links to all plan documents
- BD task IDs and quick access
- Quick start guide
- FAQ and common questions

---

## 🎯 BD Task Tracking

All deliverables tracked in the BD (Beads) system:

| Task ID | Title | Duration | Role | Status |
|---------|-------|----------|------|--------|
| `tmux-intray-4dbn` | D1: Template Engine & Variables Package | 2 days | Backend Engineer | 🟢 Ready |
| `tmux-intray-oet5` | D2: Format Command Integration | 1.5 days | CLI Engineer | 🟡 Blocked by D1 |
| `tmux-intray-93h8` | D3: Backward Compatibility Layer | 1 day | Backend Engineer | 🟡 Blocked by D1, D2 |
| `tmux-intray-yulc` | D4: Integration & End-to-End Tests | 1.5 days | QA Engineer | 🟡 Blocked by D1, D2, D3 |
| `tmux-intray-gqqa` | D5: Documentation & Help Text | 1-2 days | Tech Writer | 🟡 Blocked by D2, D4 |

### View Tasks
```bash
# View single task with all details
bd show tmux-intray-4dbn

# View dependency tree
bd dep tree tmux-intray-gqqa

# List all status-format tasks
bd list --label status-format
```

---

## 🚀 Quick Start Guide

### For Team Lead
1. Read PARALLEL_WORKSTREAMS.md (20-30 min)
2. Review this index and task list
3. Assign backend engineer to D1 (critical path)
4. Schedule team kickoff

### For D1 Engineer (Backend)
1. Read PARALLEL_WORKSTREAMS.md "D1: Template Engine..." section
2. Review acceptance criteria for D1
3. Create `internal/formatter/` directory
4. Start with `template.go` (template parser)
5. Write unit tests alongside (aim for 100% coverage)

### For D2 Engineer (CLI) - Day 2-3
1. Read PARALLEL_WORKSTREAMS.md "D2: Format Command..." section
2. Watch D1 PR for interface stabilization
3. Prepare development environment
4. Start when D1 interfaces are ready

### For D3 Engineer (Backend) - Day 3-4
1. Read PARALLEL_WORKSTREAMS.md "D3: Backward Compatibility..." section
2. Ensure D1 & D2 complete first
3. Refactor `status-panel-cmd.go` to use formatter
4. Run regression tests to verify pixel-perfect match

### For QA Engineer - Day 4-5
1. Read PARALLEL_WORKSTREAMS.md "D4: Integration & E2E..." section
2. Prepare test fixtures during D1-D3
3. Write comprehensive test suite once code stable
4. Run coverage analysis (target >80%)

### For Tech Writer - Day 5-6
1. Read PARALLEL_WORKSTREAMS.md "D5: Documentation..." section
2. Wait for D2 (feature implementation) and D4 (tests prove behavior)
3. Create user guide with examples
4. Update CLI references and README

---

## 📊 Timeline Overview

### Sequential Path (6.5 days)
```
Day 1-2:   D1 ════════════════════
Day 2-3:            D2 ══════════════
Day 3-4:                D3 ════════
Day 4-5:                   D4 ════════════
Day 5-6:                      D5 ══════════
```

### Parallel Optimized (5-6 days with 2-3 engineers)
```
Day 1-2:   D1 ════════════════════
Day 2-3:   D1 ════════ D2 ══════════════
Day 3-4:             D2 ═════ D3 ═════
Day 4-5:                  D4 ════════════════
Day 5-6:                     D5 ══════════════
```

---

## ✅ Acceptance Criteria Summary

### D1: Template Engine & Variables Package
- [ ] Template parser correctly identifies `${variable}` syntax
- [ ] All 13 template variables implemented
- [ ] All 6 presets registered (compact, detailed, json, count-only, levels, panes)
- [ ] VariableContext struct exported
- [ ] Interfaces exported and documented
- [ ] >80% unit test coverage

### D2: Format Command Integration  
- [ ] `--format` flag accepted and parsed
- [ ] Presets work: compact, detailed, json, count-only, levels, panes
- [ ] Custom templates with `${variables}` work
- [ ] Help text includes examples
- [ ] Invalid variables return helpful error
- [ ] Backward compatible with existing format names

### D3: Backward Compatibility Layer
- [ ] status-panel output pixel-perfect identical
- [ ] No breaking changes to existing configs
- [ ] All legacy tests pass (100%)
- [ ] Data extraction not duplicated

### D4: Integration & End-to-End Tests
- [ ] All 6 preset formats tested
- [ ] All 13 variables tested
- [ ] All error cases tested
- [ ] Backward compatibility verified
- [ ] >80% coverage across all new code
- [ ] No flaky tests

### D5: Documentation & Help Text
- [ ] `bd status --help` comprehensive
- [ ] `docs/status-command-guide.md` created
- [ ] All examples tested and working
- [ ] No broken links
- [ ] Migration guide from status-panel included

---

## 🔗 Integration Points

### D1 → D2 (Template Engine Contract)
Exported from `internal/formatter/`:
- `TemplateEngine` interface
- `VariableContext` struct
- `PresetRegistry` interface

### D2 → D3 (Command Output)
Status command uses formatter, data extraction reusable

### D1-3 → D4 (Testing Interface)
All code must be testable with deterministic behavior

### D1-4 → D5 (Documentation Contract)
Features documented must match actual behavior

---

## ⚠️ Key Risks & Mitigations

| Risk | Level | Mitigation |
|------|-------|-----------|
| D1 complexity | MEDIUM | Start simple (regex parser), iterate |
| Data model misalignment | LOW | Define VariableContext in D1 first |
| Backward compat breaks | HIGH | D3 + D4 pixel-perfect regression tests |
| Test coverage gaps | MEDIUM | D1: 100% unit, D4: integration focus |
| Doc staleness | LOW | D5 happens last, examples tested |

---

## 📁 File Organization Summary

### New Files (D1, D4, D5)
```
internal/formatter/           # NEW (D1)
├── template.go
├── template_test.go
├── variables.go
├── variables_test.go
├── presets.go
└── presets_test.go

docs/
└── status-command-guide.md   # NEW (D5)

tests/integration/
└── formatter_e2e.bats        # NEW (D4, optional)
```

### Modified Files (D2, D3, D5)
```
cmd/tmux-intray/
├── status.go                 # D2 core, D5 help text
├── status_test.go            # D2 basic, D4 integration
└── status-panel-cmd.go       # D3 refactor

internal/format/
└── status.go                 # D3 adds helpers

docs/
├── cli/CLI_REFERENCE.md      # D5 updates
└── README.md                 # D5 updates
```

**Total Code**: ~1,350 lines (50% tests)

---

## 🎯 How to Track Progress

### In BD System
```bash
# Update task status
bd update <task-id> --status in_progress
bd update <task-id> --status blocked  # if blocked
bd close <task-id>  # when complete

# Check blockers
bd dep tree tmux-intray-gqqa  # shows full dependency chain

# List my tasks
bd ready  # shows unblocked work available
```

### In Git
- Create feature branch per deliverable
- Include task ID in PR title: `D1: Template engine parser implementation`
- Link BD task in PR description

### Communication
- Async: GitHub PR comments for implementation details
- Sync: Weekly standup on progress and blockers
- Escalation: Slack only for critical blockers

---

## ❓ FAQ

**Q: When can I start D2?**  
A: When D1 basic interfaces are ready (day 1-2). Watch D1 PR for interface stabilization signal.

**Q: Do I need to understand all 5 deliverables?**  
A: No. Read PARALLEL_WORKSTREAMS.md for your deliverable. Skim others for context. Tech lead reviews overall plan.

**Q: What if D1 overruns by 1 day?**  
A: D2 starts day 2-3 instead of 1-2. D3 can still start once D1 complete. Final timeline shifts to 7 days sequential, 6 parallel.

**Q: How do we handle merge conflicts if two people touch the same file?**  
A: D2 modifies `status.go` first (core flag), D5 only touches help text. D3 modifies `status-panel-cmd.go` after D2 done. Clear sequencing.

**Q: What if a deliverable finds a blocker?**  
A: Update BD task status to "blocked" with reason. Tech lead triages and adjusts plan. Communicate in standup.

**Q: Can we do D4 in parallel with D3?**  
A: Not recommended - D4 needs D3 complete for comprehensive regression testing. D4 can start prep work during D3.

**Q: What coverage target is critical?**  
A: >80% minimum, but D1 should aim for 100% (simple, focused). D4 focuses on integration/edge cases.

**Q: How do we verify backward compatibility?**  
A: D3 includes pixel-perfect output comparison tests. D4 runs all legacy tests. This is critical - don't skip.

---

## 📖 Document Cross-References

**From PARALLEL_WORKSTREAMS.md**:
- See "D1: Template Engine & Variables Package" section (page ~)
- See "Integration Points & Contracts" section (page ~)
- See "File Overlap Analysis" section (page ~)
- See "Risk Assessment & Mitigation" section (page ~)

**From original epic documentation**:
- `docs/plans/status-format-extension.md` - Original comprehensive plan
- Command spec: See "Command Specification" section in original plan
- Template variables: See "Data Model" section in original plan

---

## 🎬 Next Action Items

### Immediate (Today)
- [ ] Tech lead reviews PARALLEL_WORKSTREAMS.md
- [ ] Assign backend engineer to D1
- [ ] D1 engineer reads D1 section

### This Week
- [ ] Team kickoff to review plan (30 min)
- [ ] D1 engineer creates internal/formatter/ and starts template.go
- [ ] Prepare D2 engineer (read section, review requirements)

### By Day 3
- [ ] D1 interfaces stabilized
- [ ] D2 engineer starts implementation
- [ ] Prepare D3 engineer

### By Day 5
- [ ] D1 complete
- [ ] D2 & D3 in progress
- [ ] D4 engineer (QA) starts test suite
- [ ] D5 engineer (Tech Writer) starts documentation

---

**Document Status**: Ready for team execution  
**Created**: February 26, 2026  
**Plan Verified**: ✅ All 5 deliverables scoped and tracked  

Questions? Start with PARALLEL_WORKSTREAMS.md for your deliverable.
