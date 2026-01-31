# Research: tmux-intray CLI Feature Implementation Design

## Executive Summary

This research document outlines the design for implementing the full tmux-intray notification system as specified in the project's design documentation. The current implementation uses tmux environment variables for basic tray functionality, while the design documents describe a comprehensive notification system with persistent storage, pane origin tracking, status indicators, and jump-to-source capabilities. The implementation will follow standard Linux filesystem hierarchy with notifications stored in `~/.local/state/tmux-intray/` and configuration in `~/.config/tmux-intray/`. The project will be split into 4 testable phases allowing incremental delivery and validation.

## Research Methodology

- Topics decomposed into 2 parallel investigations: CLI feature requirements and existing design docs
- Research conducted by specialized research-assistant agents
- Synthesis coordinated by Researcher agent
- Analysis of both current implementation and design documentation

## Findings by Subtopic

### Existing System Analysis

The current tmux-intray implementation provides basic functionality:
- **Storage**: Uses tmux environment variables (`TMUX_INTRAY_ITEMS`, `TMUX_INTRAY_VISIBLE`) as colon-separated strings
- **Commands**: `add`, `show`, `clear`, `toggle`, `help`, `version`
- **Architecture**: Modular Bash implementation with command-specific modules
- **Limitations**: No persistence across sessions, no origin tracking, no status indicators, no jump-to-source capability

### Design Documentation Analysis

The system design documents (`docs/sytem-design/`) specify a comprehensive notification system:

**Core Requirements**:
1. **FR-1**: Emit notifications from any pane with automatic session/window/pane capture
2. **FR-2**: Persistence across tmux detach/attach and pane/window changes
3. **FR-3**: Notification states (active/dismissed) with explicit transitions
4. **FR-4**: Status indicator showing presence/count (O(1) access)
5. **FR-5**: List notifications with message, source, and timestamp
6. **FR-6**: Jump to source pane/window on user initiation
7. **FR-7**: Dismiss individual notifications or clear all

**Storage Design**: Hybrid approach using append-only TSV logs for notifications and dismissed IDs, with tmux options for fast status bar access. TSV format: `id\ttimestamp\tstate\tsession\twindow\tpane\tmessage`

**User Stories**: 
- Persistent notifications for long-running processes
- Passive awareness without context switching
- Review capabilities with source context
- Jump-to-source functionality
- Individual notification dismissal

## Integrated Analysis

The gap between current implementation and design specifications is substantial. The current system serves as a minimal viable product (Phase 0), while the design describes Phases 1-2. Key integration points:

1. **Storage Migration**: Move from tmux environment variables to file-based TSV storage
2. **Metadata Enhancement**: Add pane/window/session tracking to notifications
3. **CLI Expansion**: Extend command set with `list`, `dismiss`, `jump` commands
4. **Status Integration**: Add tmux status bar indicator
5. **Configuration System**: Add user-configurable settings

## User Requirement Integration

**Storage Location Requirement**: The user specified using standard Linux locations:
- **Notifications**: `~/.local/state/tmux-intray/` (following XDG Base Directory Specification)
- **Configuration**: `~/.config/tmux-intray/` (following XDG Base Directory Specification)

This aligns with Linux filesystem hierarchy standards and provides proper separation of data, state, and configuration.

## Proposed Architecture

### File Structure
```
~/.local/state/tmux-intray/
├── notifications.tsv    # Append-only notification log
├── dismissed.tsv       # Dismissed notification IDs
└── lock                # File lock for concurrency control

~/.config/tmux-intray/
├── config.sh           # User configuration
└── style.conf          # Display styling preferences
```

### TSV Format Specification
```
id      timestamp               state   session window  pane    message
1       2026-01-31T10:30:00Z    active  %0      0       0       "Build completed"
2       2026-01-31T10:35:00Z    active  %0      1       1       "Tests failed"
```

### Core Components
1. **Storage Manager**: Handles TSV file I/O with flock locking
2. **Context Capturer**: Captures tmux pane/window/session metadata
3. **Status Updater**: Updates tmux options for O(1) status bar access
4. **Command Router**: Dispatches to appropriate command handlers
5. **Configuration Loader**: Loads user settings from config directory

## Implementation Phases (Testable Increments)

### Phase 1: Core Storage & Basic Commands (Testable Unit)
**Objective**: Replace environment variable storage with file-based TSV storage
**Deliverables**:
- Storage library with flock locking
- Updated `add` command using TSV storage
- New `list` command showing notifications with metadata
- New `dismiss` command for individual notifications
- Basic configuration system

**Testing Criteria**:
- Notifications persist across tmux sessions
- Concurrent access handled safely
- Commands produce expected output
- All existing tests pass with new storage

### Phase 2: Pane Association & Jump-to-Source (Testable Unit)
**Objective**: Implement origin tracking and navigation
**Deliverables**:
- Automatic pane/window/session context capture
- `jump` command to navigate to source pane
- Enhanced `list` output showing source information
- Configuration for jump behavior

**Testing Criteria**:
- Context captured correctly for notifications
- `jump` command successfully navigates to source
- Source information displayed accurately
- Edge cases handled (pane/window no longer exists)

### Phase 3: Status Indicator Integration (Testable Unit)
**Objective**: Add non-intrusive status bar indicator
**Deliverables**:
- Status bar component with active count
- Tmux format string: `#{tmux_intray_active_count}`
- Configuration for status format and position
- Automatic count updates

**Testing Criteria**:
- Status bar shows correct notification count
- Updates in real-time as notifications change
- Configurable format works as expected
- No performance impact on status updates

### Phase 4: Advanced Features & Polish (Testable Unit)
**Objective**: Add advanced functionality and refinements
**Deliverables**:
- Notification levels (info/warn/error)
- Filtering and grouping capabilities
- Garbage collection for old notifications
- Hooks and automation system
- Performance optimizations

**Testing Criteria**:
- Notification levels displayed correctly
- Filtering works as specified
- Garbage collection removes old entries appropriately
- Hooks trigger expected actions
- Performance meets requirements

## Technical Specifications

### Storage Implementation Details
- **File Locking**: Use `flock` for concurrent access control
- **TSV Parsing**: Use Bash native IFS for tab-separated parsing
- **Performance**: O(1) status updates via tmux options
- **Scalability**: Designed for thousands of entries
- **Backward Compatibility**: Migration path from environment variable storage

### CLI Command Interface
```
tmux-intray add <message>           # Add notification with current context
tmux-intray list [--active|--dismissed]  # List notifications
tmux-intray dismiss <id>            # Dismiss specific notification
tmux-intray jump <id>               # Jump to notification source
tmux-intray clear                   # Clear all active notifications
tmux-intray status                  # Show system status
tmux-intray help                    # Show help
tmux-intray version                 # Show version
```

### Configuration Options
```bash
# ~/.config/tmux-intray/config.sh
TMUX_INTRAY_STORAGE_DIR="$HOME/.local/state/tmux-intray"
TMUX_INTRAY_MAX_NOTIFICATIONS=1000
TMUX_INTRAY_STATUS_FORMAT="[#{tmux_intray_active_count}]"
TMUX_INTRAY_STATUS_POSITION="right"
TMUX_INTRAY_AUTO_CLEANUP_DAYS=30
```

## Migration Strategy

1. **Phase 0 to Phase 1 Migration**:
   - Detect existing environment variable storage
   - Convert to TSV format on first run
   - Preserve existing notification items
   - Update tmux plugin to use new storage

2. **Configuration Migration**:
   - Support both old and new configuration locations
   - Provide migration script if needed
   - Document changes for users

## Risks & Mitigations

1. **Performance Impact**: File I/O may be slower than environment variables
   - *Mitigation*: Use tmux options for status updates (O(1) access)
   - *Mitigation*: Implement efficient TSV parsing

2. **Concurrency Issues**: Multiple tmux instances accessing same files
   - *Mitigation*: Use `flock` for file locking
   - *Mitigation*: Implement retry logic for contention

3. **Data Corruption**: Power loss during file writes
   - *Mitigation*: Append-only log design
   - *Mitigation*: Atomic file operations where possible

4. **Backward Compatibility**: Existing users with environment variable storage
   - *Mitigation*: Automatic migration on first run
   - *Mitigation*: Clear documentation of changes

## Conclusions & Recommendations

**Immediate Next Steps**:
1. Implement Phase 1 (Core Storage & Basic Commands)
2. Create test suite for new storage system
3. Update documentation for new architecture
4. Begin implementation with focus on testable increments

**Long-term Vision**:
- Fully featured tmux notification system
- Integration with other tmux plugins
- Potential GUI/visual enhancements
- Community plugin ecosystem

**Success Metrics**:
- All existing tests pass with new implementation
- Notifications persist across tmux sessions
- Status bar updates correctly
- Users can jump to notification sources
- Performance meets or exceeds requirements

## Sources & References

- tmux-intray design documentation (`docs/sytem-design/`)
- Current implementation codebase
- XDG Base Directory Specification
- tmux manual pages for pane/window/session identifiers
- Linux filesystem hierarchy standard

## Research Process Details

- **Date**: Sat Jan 31 2026
- **Subagents used**: research-assistant (x2) for parallel investigation
- **Synthesis approach**: Combined analysis of current implementation gaps, design documentation requirements, and user specifications for storage locations
- **Key Insight**: The project needs to transition from simple environment variable storage to a full file-based notification system with proper Linux filesystem hierarchy compliance