# tmux-intray CLI Feature Implementation Plan

## Overview
This document outlines the phased implementation plan for the tmux-intray notification system upgrade. Each phase is designed to be independently testable, allowing for incremental delivery and validation.

## Phase 1: Core Storage & Basic Commands

### Objective
Replace tmux environment variable storage with file-based TSV storage in standard Linux locations.

### Testable Deliverables
1. **Storage Directory Structure Created**
   - `~/.local/state/tmux-intray/` directory exists
   - `~/.config/tmux-intray/` directory exists
   - Proper permissions set (user read/write)

2. **TSV Storage Library**
   - `lib/storage.sh` with functions:
     - `storage_init()`: Initialize storage directories
     - `storage_add_notification()`: Add notification to TSV file
     - `storage_list_notifications()`: List notifications with filtering
     - `storage_dismiss_notification()`: Mark notification as dismissed
     - `storage_get_active_count()`: Get count of active notifications
   - File locking using `flock`
   - TSV format: `id\ttimestamp\tstate\tsession\twindow\tpane\tmessage`

3. **Updated `add` Command**
   - Uses new storage system instead of environment variables
   - Captures timestamp automatically
   - Returns notification ID
   - Maintains backward compatibility during migration

4. **New `list` Command**
   - Shows notifications in human-readable format
   - Supports `--active` and `--dismissed` filters
   - Displays ID, timestamp, source, and message
   - Color-coded output based on age

5. **New `dismiss` Command**
   - Dismiss individual notification by ID
   - Updates state in notifications.tsv
   - Adds entry to dismissed.tsv
   - Updates active count

6. **Configuration System**
   - `~/.config/tmux-intray/config.sh` template
   - Configurable storage paths
   - Default settings loaded

7. **Migration Script**
   - Detect existing `TMUX_INTRAY_ITEMS` environment variable
   - Convert to TSV format
   - Preserve existing items
   - Clear environment variables after migration

### Testing Criteria
- ✅ Notifications persist after tmux detach/attach
- ✅ Concurrent `add` operations handled safely
- ✅ `list` shows correct information
- ✅ `dismiss` removes notifications from active list
- ✅ Active count updates correctly
- ✅ All existing tests pass
- ✅ Migration from environment variables works

### Exit Criteria for Phase 1
- All Phase 1 tests pass
- Storage library complete and tested
- Commands working with new storage
- Migration path validated
- Documentation updated

## Phase 2: Pane Association & Jump-to-Source

### Objective
Implement origin tracking and navigation to source panes.

### Testable Deliverables
1. **Context Capture Library**
   - `lib/context.sh` with functions:
     - `context_capture()`: Capture current session/window/pane
     - `context_to_string()`: Convert context to display format
     - `context_validate()`: Check if context still exists
     - `context_jump()`: Navigate to captured context

2. **Enhanced `add` Command**
   - Automatically captures current context
   - Stores session/window/pane metadata
   - Validates context is capturable

3. **Enhanced `list` Command**
   - Shows source information (session/window/pane)
   - Indicates if source still exists
   - Color-codes based on source validity

4. **New `jump` Command**
   - `tmux-intray jump <id>`
   - Navigates to notification source
   - Handles missing sources gracefully
   - Provides feedback on success/failure

5. **Context Validation**
   - Check if captured panes still exist
   - Update display to show stale sources
   - Provide warning when jumping to stale sources

### Testing Criteria
- ✅ Context captured correctly for new notifications
- ✅ `list` shows accurate source information
- ✅ `jump` successfully navigates to valid sources
- ✅ `jump` handles missing sources gracefully
- ✅ Context validation works correctly
- ✅ Edge cases handled (panes/windows closed)

### Exit Criteria for Phase 2
- All Phase 2 tests pass
- Context capture working reliably
- Jump functionality complete
- Source display accurate
- Error handling robust

## Phase 3: Status Indicator Integration

### Objective
Add non-intrusive status bar indicator showing notification count.

### Testable Deliverables
1. **Status Library**
   - `lib/status.sh` with functions:
     - `status_update()`: Update tmux options with current count
     - `status_get_format()`: Get formatted status string
     - `status_refresh()`: Force status bar update

2. **Status Format Integration**
   - Tmux format string: `#{tmux_intray_active_count}`
   - Configurable format in config.sh
   - Automatic updates on notification changes

3. **Status Bar Configuration**
   - Update `tmux-intray.tmux` plugin
   - Add status-right component
   - Make status optional/configurable

4. **Performance Optimization**
   - O(1) access to active count via tmux options
   - Batch updates for multiple operations
   - Minimal impact on status bar refresh

5. **Visual Styles**
   - Configurable colors and formats
   - Different styles for zero/non-zero counts
   - Optional emoji indicators

### Testing Criteria
- ✅ Status bar shows correct notification count
- ✅ Updates in real-time when notifications change
- ✅ Configurable format works as expected
- ✅ No performance degradation in status updates
- ✅ Works with different tmux status bar configurations

### Exit Criteria for Phase 3
- All Phase 3 tests pass
- Status indicator working correctly
- Performance requirements met
- Configuration options validated
- Documentation complete

## Phase 4: Advanced Features & Polish

### Objective
Add advanced functionality and refinement features.

### Testable Deliverables
1. **Notification Levels**
   - Support for info/warn/error levels
   - Different colors and prefixes
   - Filtering by level in `list` command

2. **Filtering & Grouping**
   - Filter by source (session/window/pane)
   - Filter by time range
   - Group related notifications
   - Search within messages

3. **Garbage Collection**
   - Automatic cleanup of old notifications
   - Configurable retention period
   - Manual cleanup command

4. **Hooks & Automation**
   - Pre/post notification hooks
   - Integration with other tools
   - Custom script execution

5. **Performance Optimizations**
   - Indexed access for large datasets
   - Batch operations
   - Memory usage improvements

6. **Documentation & Examples**
   - Complete user guide
   - Example configurations
   - Integration examples
   - Troubleshooting guide

### Testing Criteria
- ✅ Notification levels displayed correctly
- ✅ Filtering works as specified
- ✅ Garbage collection removes old entries
- ✅ Hooks trigger expected actions
- ✅ Performance meets requirements with large datasets
- ✅ All features documented with examples

### Exit Criteria for Phase 4
- All Phase 4 tests pass
- Advanced features complete
- Performance optimized
- Documentation comprehensive
- Ready for production use

## Testing Strategy

### Unit Tests
- **Storage Tests**: Verify TSV file operations, locking, parsing
- **Command Tests**: Test each CLI command with various inputs
- **Context Tests**: Validate context capture and navigation
- **Status Tests**: Verify status bar integration

### Integration Tests
- **End-to-End Tests**: Full workflow from add to dismiss to jump
- **Migration Tests**: Environment variable to TSV migration
- **Concurrency Tests**: Multiple tmux instances accessing storage
- **Persistence Tests**: Data survives tmux restart

### Performance Tests
- **Scalability**: Handling thousands of notifications
- **Response Time**: Command execution times
- **Memory Usage**: Storage and processing overhead
- **Concurrent Access**: Multiple simultaneous operations

## Development Workflow

### Phase 1 Development Steps
1. Create storage directory structure
2. Implement storage library with tests
3. Update `add` command to use storage
4. Implement `list` command
5. Implement `dismiss` command
6. Add configuration system
7. Implement migration script
8. Update documentation

### Continuous Integration
- Run existing test suite after each change
- Add new tests for each feature
- Validate backward compatibility
- Check performance benchmarks

### Quality Gates
- All tests pass
- ShellCheck passes without errors
- Documentation updated
- Migration path tested
- Performance requirements met

## Risk Mitigation

### Technical Risks
1. **File Locking Issues**: Use well-tested `flock` utility, implement retry logic
2. **Performance Degradation**: Profile and optimize, use tmux options for O(1) access
3. **Data Corruption**: Append-only log design, atomic operations where possible
4. **Backward Compatibility**: Clear migration path, deprecation warnings

### Project Risks
1. **Scope Creep**: Stick to phased approach, defer non-essential features
2. **Testing Complexity**: Incremental testing, focus on testable deliverables
3. **User Adoption**: Clear documentation, migration tools, backward compatibility

## Success Metrics

### Phase 1 Success Metrics
- All existing functionality preserved
- Notifications persist across sessions
- Storage system handles concurrent access
- Migration works transparently

### Overall Success Metrics
- Users can track and navigate to notification sources
- Status bar provides useful at-a-glance information
- System performs well with typical workloads
- Users report positive experience

## Timeline & Milestones

### Phase 1 (Core Storage): 2-3 weeks
- Week 1: Storage library and basic commands
- Week 2: Migration and testing
- Week 3: Polish and documentation

### Phase 2 (Pane Association): 1-2 weeks
- Week 1: Context capture and jump functionality
- Week 2: Testing and refinement

### Phase 3 (Status Indicator): 1 week
- Status bar integration and testing

### Phase 4 (Advanced Features): 2-3 weeks
- Advanced functionality and polish

## Next Steps

### Immediate Actions
1. Review and approve implementation plan
2. Set up development environment
3. Begin Phase 1 implementation
4. Establish testing infrastructure

### Ongoing Actions
- Regular progress reviews
- User feedback collection
- Performance monitoring
- Documentation updates

## Conclusion

This phased implementation plan provides a clear path to upgrading tmux-intray from a simple tray system to a full-featured notification system. Each phase delivers testable functionality, allowing for incremental validation and user feedback. The plan emphasizes backward compatibility, performance, and user experience throughout the implementation process.