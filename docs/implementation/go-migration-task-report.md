# Go Migration Task Report

## Task Inventory by Priority

| Task ID | Priority | Status | Dependencies | Testing Requirements | Complexity | Estimate |
|---------|----------|--------|--------------|----------------------|------------|----------|
| tmux-intray-z445 | P0 | IN_PROGRESS | none | report completion | Low | Missing estimate |
| tmux-intray-549 | P1 | IN_PROGRESS | none | n/a | High | Missing estimate |
| tmux-intray-w9e | P2 | IN_PROGRESS | tmux-intray-549 | design review | High | Missing estimate |
| tmux-intray-8xh | P2 | OPEN | tmux-intray-549 | strategy doc unit/integration/shadow/perf/CI | Med | Missing estimate |
| tmux-intray-jrf | P2 | OPEN | tmux-intray-549 | perf benchmarks Go vs Bash | Med | Missing estimate |
| tmux-intray-37r | P2 | OPEN | tmux-intray-549 | golden fixtures from Bats | Med | Missing estimate |
| tmux-intray-b6i | P2 | OPEN | tmux-intray-549 | dual-runner Bash/Go integration | Med | Missing estimate |
| tmux-intray-b8i | P2 | OPEN | none | docs | Low | Missing estimate |
| tmux-intray-bgd | P2 | OPEN | none | docs | Low | Missing estimate |
| tmux-intray-szo | P2 | OPEN | none | docs | Low | Missing estimate |
| tmux-intray-77n | P2 | OPEN | tmux-intray-549, tmux-intray-lyc, tmux-intray-m6c, tmux-intray-niz, tmux-intray-5bw (done) | parent | High | Missing estimate |
| tmux-intray-lyc | P2 | IN_PROGRESS | tmux-intray-549, tmux-intray-w9e | unit config env/default | High | Missing estimate |
| tmux-intray-m6c | P2 | IN_PROGRESS | tmux-intray-549, tmux-intray-w9e | unit/integration round-trip | High | Missing estimate |
| tmux-intray-niz | P2 | OPEN | tmux-intray-549, tmux-intray-w9e | unit hooks/integration scripts | High | Missing estimate |
| tmux-intray-w9m | P2 | OPEN | tmux-intray-549, tmux-intray-lyc, tmux-intray-m6c, tmux-intray-niz | ≥80% coverage core | Med | Missing estimate |
| tmux-intray-ec3 | P2 | OPEN | tmux-intray-549, tmux-intray-w9m | TSV/hook/compat | Med | Missing estimate |
| tmux-intray-34k | P2 | IN_PROGRESS | tmux-intray-77n | version output | Low | Missing estimate |
| tmux-intray-046 | P2 | OPEN | tmux-intray-77n | notification summary | Med | Missing estimate |
| tmux-intray-12l | P2 | OPEN | tmux-intray-77n | tmux bar | Med | Missing estimate |
| tmux-intray-31x | P2 | OPEN | tmux-intray-77n | help text | Low | Missing estimate |
| tmux-intray-5sk | P2 | OPEN | tmux-intray-77n | dismiss | Med | Missing estimate |
| tmux-intray-6al | P2 | OPEN | tmux-intray-77n | realtime follow | High | Missing estimate |
| tmux-intray-cgg | P2 | OPEN | tmux-intray-77n | cleanup | Low | Missing estimate |
| tmux-intray-eas | P2 | OPEN | tmux-intray-77n | validation/hooks | Med | Missing estimate |
| tmux-intray-h6y | P2 | OPEN | tmux-intray-77n | filters/formats | Med | Missing estimate |
| tmux-intray-npa | P2 | OPEN | tmux-intray-77n | toggle | Low | Missing estimate |
| tmux-intray-ple | P2 | OPEN | tmux-intray-77n | clear | Low | Missing estimate |
| tmux-intray-x4z | P2 | OPEN | tmux-intray-77n | tmux jump | High | Missing estimate |
| tmux-intray-bra | P2 | OPEN | tmux-intray-549, tmux-intray-77n, tmux-intray-o2p (done) | final switchover | High | Missing estimate |

## Dependency Edges

- **tmux-intray-549** → tmux-intray-w9e, tmux-intray-8xh, tmux-intray-jrf, tmux-intray-37r, tmux-intray-b6i, tmux-intray-77n, tmux-intray-lyc, tmux-intray-m6c, tmux-intray-niz, tmux-intray-w9m, tmux-intray-ec3, tmux-intray-bra
- **tmux-intray-w9e** → tmux-intray-lyc, tmux-intray-m6c, tmux-intray-niz
- **tmux-intray-lyc** → tmux-intray-77n, tmux-intray-w9m
- **tmux-intray-m6c** → tmux-intray-77n, tmux-intray-w9m
- **tmux-intray-niz** → tmux-intray-77n, tmux-intray-w9m
- **tmux-intray-w9m** → tmux-intray-ec3
- **tmux-intray-77n** → tmux-intray-34k, tmux-intray-046, tmux-intray-12l, tmux-intray-31x, tmux-intray-5sk, tmux-intray-6al, tmux-intray-cgg, tmux-intray-eas, tmux-intray-h6y, tmux-intray-npa, tmux-intray-ple, tmux-intray-x4z, tmux-intray-bra
- **tmux-intray-5bw (done)** → tmux-intray-77n
- **tmux-intray-o2p (done)** → tmux-intray-bra

## Testing Requirements

Each task includes specific testing criteria as noted in the table above.

## Summary Statistics

- **Total tasks**: 29
- **Priority breakdown**: P0: 1, P1: 1, P2: 27, P3/P4: 0
- **Status breakdown**: IN_PROGRESS: 6, OPEN: 23
- **Complexity tally**: High: 9, Med: 12, Low: 8, Missing: 0
- **Estimate status**: All tasks marked "Missing estimate"

## Prioritized Execution Order

**Phase 1**: w9e, 8xh, jrf, 37r, b6i  
**Phase 2**: lyc, m6c, niz, w9m, ec3  
**Phase 3**: 34k, 31x, npa, ple, cgg  
**Phase 4**: 046, 5sk, eas, h6y  
**Phase 5**: 12l, 6al, x4z  
**Phase 6**: bra (switchover)

## Assumptions

1. Dependencies marked "(done)" are already completed and do not require implementation.
2. Task IDs are unique and refer to specific work items.
3. Complexity ratings (High/Med/Low) are based on initial assessment and may change during implementation.
4. All estimates are currently missing and will need to be populated as part of planning.
5. The execution order is a suggested sequencing based on dependencies; actual implementation may adjust based on resource availability.
6. Testing requirements are as specified per task; additional testing may be required during development.
7. The report captures the state after rebase loss; dependencies have been corrected to use full task IDs.