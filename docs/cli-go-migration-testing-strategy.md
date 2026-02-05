# Testing Strategy: Using Bats Tests for Go CLI Validation

## Overview
**Primary Integration Testing Approach**: Use the existing Bats (Bash Automated Testing System) test suite as the **gold standard** for validating the Go CLI implementation.

**Rationale**:
1. **Comprehensive Coverage**: Bats tests already exist and thoroughly test CLI behavior
2. **User Perspective**: Tests the actual CLI interface users interact with
3. **Behavioral Validation**: Ensures Go implementation matches Bash exactly
4. **Cost Effective**: No need to rewrite integration test logic

## Implementation Approach

### 1. Bats Test Adapter
The existing Bats tests now run directly against the Go binary:
- **Go implementation**: The `tmux-intray` binary

**Test Execution**:
```bash
# Build and test Go implementation
make go-build
make test
```

### 2. Test Runner Design
```bash
#!/usr/bin/env bash
# tests/run-bats.sh

# Ensure Go binary is built
if [[ ! -f "./tmux-intray" ]]; then
  make go-build
fi

# Export binary path for tests
export TMUX_INTRAY_BINARY="./tmux-intray"

# Run Bats tests
bats "$@"
```

### 3. Test Adaptation in Bats Files
The Bats tests have been updated to use the Go binary directly:

```bash
# Updated tests use:
@test "add command works" {
  run ./tmux-intray add "Test message"
  [ "$status" -eq 0 ]
}
```

### 4. Special Considerations

**Test Implementation**:
- All tests have been updated to work with the Go binary
- Tests now verify behavior rather than implementation details
- Proper test isolation is maintained with Go's storage layer

## Integration with CI/CD

### Test Matrix
```yaml
# GitHub Actions matrix
strategy:
  matrix:
    os: [ubuntu-latest, macos-latest]
  
steps:
  - name: Build Go binary
    run: make go-build
  - name: Run Bats tests
    run: make test
```

### Current Status
All tests have been migrated to use the Go binary directly.

## Benefits

### 1. **Reduced Test Development Effort**
- No need to rewrite integration tests in Go
- Leverage existing test investment
- Faster validation of migrated functionality

### 2. **Behavioral Parity Assurance**
- Same tests validate both implementations
- Ensures identical user experience
- Catches subtle behavioral differences

### 3. **Progressive Validation**
- Test individual commands as they're migrated
- Early feedback on compatibility issues
- Clear pass/fail criteria for each migration step

### 4. **Continuous Integration**
- Run tests against both implementations in CI
- Track progress via test pass rates
- Automated regression detection

## Implementation Tasks

### Phase 1: Infrastructure Setup
1. **Create test adapter** (`tmux-intray-b6i`)
   - Modify Bats test runner to support binary configuration
   - Update all Bats tests to use `$TMUX_INTRAY_BINARY`
   - Handle Bash-specific test cases

2. **Set up CI matrix**
   - Add Go implementation testing to CI pipeline
   - Configure test reporting for both implementations
   - Set up failure notifications

### Phase 2-4: Progressive Testing
1. **Test core libraries**
   - Validate storage, config, hooks with Bats tests
   - Ensure compatibility with existing test expectations

2. **Test migrated commands**
   - Run command-specific Bats tests against Go implementation
   - Fix behavioral discrepancies
   - Update tests if Go behavior is intentionally different

### Phase 5: Comprehensive Validation
1. **Run full test suite**
   - Execute all Bats tests against Go implementation
   - Document and resolve any remaining failures
   - Validate edge cases and error conditions

2. **Performance comparison**
   - Use Bats tests as performance benchmarks
   - Compare execution time between implementations
   - Identify performance regressions

## Risk Mitigation

### Risk: Bash-Specific Test Dependencies
**Mitigation**:
- Audit all Bats tests for Bash-specific assumptions
- Create compatibility layer for common patterns
- Mark truly Bash-specific tests and skip for Go

### Risk: Test Flakiness
**Mitigation**:
- Improve test isolation in Bats tests
- Ensure Go implementation has same deterministic behavior
- Add retry logic for timing-sensitive tests

### Risk: False Positives/Negatives
**Mitigation**:
- Manual validation of test failures
- Create "golden output" comparison for critical tests
- Peer review of test adaptations

## Success Metrics

1. **Test Coverage**: 100% of Bats tests executable against Go implementation
2. **Pass Rate**: 95%+ of Bats tests pass with Go implementation
3. **Performance**: Go implementation passes all timing-based tests
4. **Behavior**: Zero user-visible behavioral differences identified by tests

## Integration with Other Testing Approaches

### Complementary to Golden Tests
- **Bats Tests**: Integration testing from user perspective
- **Golden Tests**: Unit-level validation of specific outputs
- **Combined**: Bats tests validate integration, golden tests validate details

### Complementary to Unit Tests
- **Unit Tests**: Test individual Go components in isolation
- **Bats Tests**: Test complete CLI functionality
- **Combined**: Comprehensive test pyramid

## Example: Test Adaptation Process

### Step 1: Identify Test to Adapt
```bash
# tests/commands/add.bats
@test "add creates notification with message" {
  run bin/tmux-intray add "Test notification"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Added notification" ]]
}
```

### Step 2: Make Binary Configurable
```bash
# tests/commands/add.bats
@test "add creates notification with message" {
  run "$TMUX_INTRAY_BINARY" add "Test notification"
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Added notification" ]]
}
```

### Step 3: Handle Implementation Differences
```bash
# If Go output differs slightly:
@test "add creates notification with message" {
  run "$TMUX_INTRAY_BINARY" add "Test notification"
  [ "$status" -eq 0 ]
  
  # Accept either Bash or Go output format
  if [[ "$TMUX_INTRAY_BINARY" == *"tmux-intray-go"* ]]; then
    [[ "$output" =~ "Notification added" ]]
  else
    [[ "$output" =~ "Added notification" ]]
  fi
}
```

### Step 4: Add to CI Pipeline
```yaml
- name: Test Go Implementation
  run: |
    make build
    export TMUX_INTRAY_BINARY="./tmux-intray-go"
    bats tests/
```

## Conclusion

Using existing Bats tests for Go CLI validation provides:
- **Rapid validation** of migrated functionality
- **High confidence** in behavioral parity
- **Reduced test development** effort
- **Clear progress tracking** via test pass rates

This approach should be implemented in **Phase 1** alongside other testing infrastructure to enable progressive validation throughout the migration.

---
*Part of tmux-intray CLI Migration to Go - Testing Strategy*  
*Document Version: 1.0*  
*Last Updated: February 2, 2026*