# tmux-intray CLI Migration to Go: Roadmap & Checklists

## Overview
**Project**: Migrate tmux-intray CLI from Bash to Go with code freeze  
**Strategy**: Direct replacement, no wrapper, no feature flags  
**Timeline**: 12-14 weeks  
**Goal**: Pure Go implementation with full backward compatibility

## Current Progress

**Last Updated**: February 4, 2026

### Accomplishments

- **Go scaffold created**: Stub implementations in internal packages (`core`, `storage`, `config`, `colors`, `hooks`, `tmuxintray`)
- **Build issues resolved**: Package conflicts resolved, dependencies configured (`go.mod`)
- **Wrapper implementation**: `cmd/wrapper/main.go` exists (embeds Bash script for transition)
- **Documentation updated**: Migration roadmap and testing strategy documents moved to `docs/`
- **Verification report updated**: Package structure verified and deviations documented
- **Package structure decision**: Simple flat cmd/ structure documented (`docs/design/package-structure-decision.md`)
- **Command implementation**: All 11 commands are implemented directly in `cmd/*.go` files, combining CLI definitions and business logic

### Current Status

- **Phase 1 (Infrastructure Setup)**: Partially complete. Go module and wrapper exist; linting, pre‑commit hooks, CI steps remain.
- **Phase 2 (Design & Core Libraries)**: In progress. Stub implementations exist; core packages need actual implementation.
- **Phases 3–6**: Not yet started (pending completion of earlier phases).

### Next Immediate Steps

1. Implement core storage layer (TSV file I/O, locking)
2. Set up Go‑specific linting (`golangci‑lint`) and pre‑commit hooks
3. Create CI workflow for Go builds and tests

## Migration Phases

### Phase 1: Infrastructure Setup (Weeks 1-3)
**Goal**: Establish Go development environment and testing infrastructure

#### ✅ **Checklist: Infrastructure Setup**

**Development Environment**
- [x] Set up Go module structure (`go.mod`)
- [ ] Configure `golangci-lint` for code quality
- [ ] Set up `gofmt` and `goimports` formatting
- [ ] Configure pre-commit hooks for Go
- [ ] Set up Go development toolchain (IDE, debuggers)

**Testing Infrastructure**
- [ ] Create Go unit test framework with `testing` package
- [ ] Set up `testify/assert` for assertions
- [ ] Configure test coverage reporting
- [ ] Create test helpers and utilities
- [ ] Set up golden test framework

**CI/CD Pipeline**
- [ ] Create GitHub Actions workflow for Go
- [ ] Configure automated testing on push/PR
- [ ] Set up linting in CI pipeline
- [ ] Configure test coverage reporting
- [ ] Set up performance benchmarking in CI

**Test Mocks**
- [ ] Define interfaces for tmux operations
- [ ] Create mock for tmux client
- [ ] Create mock for storage operations
- [ ] Create mock for file system operations
- [ ] Create mock for hook execution

**Documentation**
- [ ] Create comprehensive testing strategy document
- [ ] Document Go coding standards
- [ ] Create contributor guidelines for Go development
- [ ] Document golden test creation process

---

### Phase 2: Design & Core Libraries (Weeks 4-7)
**Goal**: Design Go package structure and implement core functionality

#### ✅ **Checklist: Design & Core Libraries**

**Status**: In progress. Stub implementations exist for core packages; need actual implementation.

**Package Structure Design**
- [x] Analyze existing Bash library architecture
- [x] Design Go package structure mirroring Bash modularity
- [x] Document package organization and responsibilities
- [-] Define public APIs for each package (stub functions exist)
- [x] Review and approve design (package structure decision accepted)

**Storage Layer Implementation**
- [ ] Implement TSV file reader/writer with Go
- [ ] Implement file locking mechanism (directory-based `mkdir`)
- [ ] Ensure backward compatibility with existing TSV format
- [ ] Implement storage operations: add, list, dismiss, clear, cleanup
- [ ] Add unit tests for storage layer (≥90% coverage)

**Configuration Management**
- [ ] Implement configuration loading from XDG directories
- [ ] Support existing config file format
- [ ] Implement environment variable overrides
- [ ] Add unit tests for config management

**Hook System Compatibility**
- [ ] Design interface for executing existing Bash hooks
- [ ] Implement hook execution with environment variables
- [ ] Support pre/post notification hooks
- [ ] Add unit tests for hook execution

**Tmux Interface Package**
- [ ] Create `TmuxClient` interface
- [ ] Implement with `exec.Command("tmux", ...)` calls
- [ ] Support session/window/pane detection
- [ ] Implement tmux command execution
- [ ] Add unit tests with mocks

**Core Utilities**
- [ ] Implement color output and logging
- [ ] Implement core utilities from `core.sh`
- [ ] Ensure compatibility with existing behavior
- [ ] Add comprehensive unit tests

**Integration Testing**
- [ ] Create integration tests with existing Bash commands
- [ ] Validate storage format compatibility
- [ ] Validate hook execution compatibility
- [ ] Run existing Bats tests as integration validation

---

### Phase 3: Command Migration Planning (Week 8)
**Goal**: Plan optimal command migration sequence

#### ✅ **Checklist: Migration Planning**

**Command Analysis**
- [ ] Analyze all 12 CLI commands and their dependencies
- [ ] Map command dependencies on core libraries
- [ ] Identify command complexity and risk levels
- [ ] Document command-specific requirements

**Migration Sequence**
- [ ] Create optimal migration order based on dependencies
- [ ] Define command groups for parallel development
- [ ] Estimate effort for each command
- [ ] Create detailed timeline with milestones

**Resource Allocation**
- [ ] Assign team members to command groups
- [ ] Define success criteria for each command
- [ ] Set up tracking for command migration progress
- [ ] Create communication plan for coordination

**Risk Assessment**
- [ ] Identify technical risks for each command
- [ ] Create mitigation strategies for high-risk commands
- [ ] Define fallback plan if issues arise
- [ ] Document assumptions and constraints

---

### Phase 4: Command Implementation (Weeks 8-12)
**Goal**: Implement all 12 CLI commands in Go

#### ✅ **Checklist: Command Implementation**

**Group A: Simple Commands (Week 8)**
- [ ] **version command**: Show version information
  - [ ] Implement version display logic
  - [ ] Match exact output format of Bash version
  - [ ] Add unit tests
  - [ ] Validate with golden tests

- [ ] **help command**: Display help text
  - [ ] Implement help text generation
  - [ ] Match help formatting and content
  - [ ] Add unit tests
  - [ ] Validate with golden tests

- [ ] **clear command**: Remove all notifications
  - [ ] Implement storage interaction for clearing
  - [ ] Execute pre/post hooks
  - [ ] Add unit tests
  - [ ] Validate with golden tests

- [ ] **toggle command**: Toggle tray visibility
  - [ ] Implement toggle logic
  - [ ] Add unit tests
  - [ ] Validate with golden tests

- [ ] **cleanup command**: Remove old dismissed notifications
  - [ ] Implement age-based cleanup logic
  - [ ] Support `--days` and `--dry-run` options
  - [ ] Add unit tests
  - [ ] Validate with golden tests

**Group B: Complex Commands (Weeks 9-10)**
- [ ] **add command**: Add new notification to tray
  - [ ] Implement message validation
  - [ ] Capture tmux context (session/window/pane)
  - [ ] Support level and association options
  - [ ] Execute pre/post hooks
  - [ ] Add comprehensive unit tests
  - [ ] Validate with golden tests

- [ ] **list command**: List notifications with filters
  - [ ] Implement filtering (active, dismissed, level, pane)
  - [ ] Support multiple output formats (table, compact, JSON)
  - [ ] Add unit tests for each filter combination
  - [ ] Validate with golden tests

- [ ] **status command**: Show notification summary
  - [ ] Implement status calculation
  - [ ] Support multiple output formats (summary, levels, panes, JSON)
  - [ ] Add unit tests
  - [ ] Validate with golden tests

- [ ] **dismiss command**: Dismiss notifications
  - [ ] Support single notification dismissal
  - [ ] Support `--all` option
  - [ ] Execute pre/post hooks
  - [ ] Add unit tests
  - [ ] Validate with golden tests

**Group C: Tmux-Dependent Commands (Week 11)**
- [ ] **jump command**: Jump to pane of notification
  - [ ] Implement tmux navigation
  - [ ] Validate tmux session/window/pane exists
  - [ ] Add unit tests with tmux mock
  - [ ] Validate with golden tests

- [ ] **status-panel command**: Generate status bar indicator
  - [ ] Implement status calculation for tmux status-right
  - [ ] Support multiple output formats (compact, detailed, count-only)
  - [ ] Add unit tests
  - [ ] Validate with golden tests

- [ ] **follow command**: Monitor notifications in real-time
  - [ ] Implement real-time monitoring
  - [ ] Support filtering options
  - [ ] Support `--interval` option
  - [ ] Add unit tests
  - [ ] Validate with golden tests

**Command Integration**
- [ ] Integrate all commands into main CLI entry point
- [ ] Ensure consistent command-line argument parsing
- [ ] Validate command help text matches Bash
- [ ] Run comprehensive integration tests

---

### Phase 5: Validation (Week 13)
**Goal**: Comprehensive testing and validation

#### ✅ **Checklist: Validation**

**Golden Test Execution**
- [ ] Run all golden tests comparing Go vs Bash outputs
- [ ] Validate stdout matches exactly
- [ ] Validate stderr matches exactly  
- [ ] Validate exit codes match
- [ ] Document any discrepancies and resolutions

**Performance Benchmarking**
- [ ] Benchmark critical paths in Go vs Bash
- [ ] Measure execution time for each command
- [ ] Measure memory usage
- [ ] Document performance improvements/regressions
- [ ] Optimize any performance bottlenecks

**Integration Testing**
- [ ] Test with real tmux sessions
- [ ] Validate tmux integration commands work correctly
- [ ] Test with different tmux versions (2.x, 3.x)
- [ ] Validate hook execution with sample hooks
- [ ] Test edge cases and error conditions

**Security Testing**
- [ ] Validate file permission handling
- [ ] Test concurrent access scenarios
- [ ] Validate input sanitization
- [ ] Test with malformed input data
- [ ] Validate error handling and reporting

**Cross-Platform Testing**
- [ ] Test on Linux (x86-64, ARM64)
- [ ] Test on macOS (Intel, Apple Silicon)
- [ ] Validate path handling differences
- [ ] Test with different shell environments

**Test Coverage Validation**
- [ ] Ensure ≥90% test coverage for all packages
- [ ] Identify and add tests for uncovered code paths
- [ ] Validate edge case coverage
- [ ] Document test coverage results

---

### Phase 6: Switchover (Week 14)
**Goal**: Replace Bash CLI with Go implementation

#### ✅ **Checklist: Switchover**

**Build System Updates**
- [ ] Update Makefile for Go build targets
- [ ] Remove Bash-specific build targets
- [ ] Update `go.mod` with final dependencies
- [ ] Configure release build process
- [ ] Validate cross-compilation for all platforms

**Installation Methods**
- [ ] Update Homebrew formula for Go binary
- [ ] Update npm package distribution
- [ ] Update GitHub Releases automation
- [ ] Update Docker container build
- [ ] Update any other distribution channels

**Documentation Updates**
- [ ] Update CLI reference documentation
- [ ] Update installation instructions
- [ ] Update contributor guidelines
- [ ] Update API/interface documentation
- [ ] Archive Bash implementation documentation

**Final Validation**
- [ ] Run full test suite one final time
- [ ] Validate all 12 commands in production-like environment
- [ ] Test upgrade path from Bash to Go version
- [ ] Validate data migration (if needed)
- [ ] Create rollback plan

**Release Preparation**
- [ ] Create release notes highlighting changes
- [ ] Update version number
- [ ] Tag release in version control
- [ ] Notify any existing users (if applicable)
- [ ] Deploy to all distribution channels

**Post-Migration Cleanup**
- [ ] Remove Bash scripts from repository
- [ ] Clean up any migration utilities
- [ ] Update CI/CD pipelines to remove Bash tests
- [ ] Archive old Bash implementation
- [ ] Document lessons learned

---

## Success Metrics

### Functional Requirements
- [ ] All 12 CLI commands work identically to Bash version
- [ ] TSV storage format fully compatible
- [ ] Hook system executes existing Bash hooks unchanged
- [ ] Configuration files load correctly from XDG locations
- [ ] Tmux integration works with tmux 2.x and 3.x

### Quality Requirements
- [ ] ≥90% test coverage for all Go packages
- [ ] Zero critical or high-severity bugs found in validation
- [ ] Performance equal or better than Bash implementation
- [ ] All existing Bats tests pass (as integration tests)

### User Experience Requirements  
- [ ] Same CLI interface (commands, options, help text)
- [ ] Same output formats and styling
- [ ] Same error messages and exit codes
- [ ] Zero configuration changes required for users
- [ ] Smooth upgrade path from Bash version

### Development Requirements
- [ ] Clean, maintainable Go codebase
- [ ] Comprehensive documentation
- [ ] Automated CI/CD pipeline
- [ ] Performance benchmarking suite
- [ ] Golden test framework for future regression testing

## Risk Management

### Identified Risks
1. **Storage format incompatibility** - Mitigated by golden tests
2. **Hook execution differences** - Mitigated by comprehensive hook testing
3. **Tmux version compatibility** - Mitigated by testing with multiple tmux versions
4. **Performance regression** - Mitigated by performance benchmarking
5. **Missing edge cases** - Mitigated by comprehensive test suite

### Mitigation Strategies
- Daily integration testing during development
- Regular golden test validation
- Performance benchmarking at each phase
- Peer review of critical components
- Rollback plan in case of critical issues

## Team Coordination

### Recommended Team Structure
- **Team Lead (1)**: Overall coordination, architecture decisions
- **Infrastructure Team (2)**: Phase 1 setup, CI/CD, testing frameworks
- **Core Libraries Team (2)**: Phase 2 implementation of storage, config, hooks, tmux
- **Command Migration Team (3)**: Phase 4 command implementation
- **Testing/Validation (1)**: Phase 5 validation, golden tests, benchmarking

### Communication Plan
- **Daily Standups**: 15 minutes per workstream
- **Weekly Sync**: Full team progress review and planning
- **Integration Checkpoints**: Every Friday for cross-team integration
- **Documentation Updates**: Continuous documentation of decisions and progress

## Timeline Summary

| Phase | Duration | Start Date | End Date | Key Milestones |
|-------|----------|------------|----------|----------------|
| **1. Infrastructure** | 3 weeks | T+0 | T+3 | Go dev environment ready |
| **2. Core Libraries** | 4 weeks | T+3 | T+7 | All core libraries implemented |
| **3. Planning** | 1 week | T+7 | T+8 | Migration sequence finalized |
| **4. Commands** | 4 weeks | T+8 | T+12 | All 12 commands implemented |
| **5. Validation** | 1 week | T+12 | T+13 | Comprehensive testing complete |
| **6. Switchover** | 1 week | T+13 | T+14 | Production deployment |

**Total Duration**: 14 weeks (with 1 week buffer)

## Progress Tracking

### Weekly Status Report Template
```
Week [X]: [Phase Name]
----------------------------
Completed This Week:
1. 
2. 
3. 

Planned for Next Week:
1. 
2. 
3. 

Blockers/Issues:
1. 
2. 

Key Decisions/Updates:
1. 
2. 

Test Coverage: [X]%
Performance: [Status]
Golden Tests: [X]/[Y] passing
```

### Beads Task Status
- Track progress using beads task management system
- Update task status daily
- Use dependencies to track blocked tasks
- Generate weekly progress reports from beads

## Appendix

### Command Dependency Graph
```
Core Libraries (Storage, Config, Hooks, Tmux)
    ├── Simple Commands (version, help, clear, toggle, cleanup)
    ├── Complex Commands (add, list, status, dismiss)
    └── Tmux-Dependent Commands (jump, status-panel, follow)
```

### Test Environment Setup
- Use `TMUX_SOCKET_NAME` for isolated tmux testing
- Create temporary directories for storage testing
- Mock external dependencies for unit tests
- Use golden test fixtures for behavioral validation

### Performance Benchmark Targets
- Command execution time < Bash equivalent
- Memory usage < 50MB for all operations
- Storage operations scale linearly with data size
- No memory leaks in long-running operations

---

*Last Updated: February 4, 2026*  
*Document Version: 1.0*  
*Owner: CLI Migration Team*