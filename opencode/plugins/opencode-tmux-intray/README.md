# OpenCode Tmux Intray Plugin

A simplified plugin for OpenCode that integrates with tmux-intray to display notifications in tmux when OpenCode sessions complete, error, or require permissions.

## Overview

The `opencode-tmux-intray` plugin connects OpenCode events to tmux-intray notifications. When OpenCode sessions emit events (like session completion, errors, or permission requests), the plugin sends notifications to tmux-intray, which displays them in your tmux status bar.

Key features:
- **Simple event-driven notifications**: Notifications for common OpenCode session events
- **Tmux context capture**: Automatically detects tmux session/window/pane for context
- **Plugin hooks**: Integrates seamlessly with OpenCode's plugin system

## Installation

### Prerequisites

1. **OpenCode**: The plugin runs within OpenCode's plugin ecosystem
2. **tmux-intray CLI**: Must be installed and accessible in your PATH
3. **tmux session**: For notifications to appear, you should be running inside tmux

### Install the Plugin

#### Quick Installation (recommended)

```bash
# Clone the repository
git clone https://github.com/cristianoliveira/tmux-intray.git
cd tmux-intray/opencode/plugins/opencode-tmux-intray

# Run the installer
./install.sh
```

The script will:
1. Create the global plugin directory: `~/.config/opencode/plugins/`
2. Copy the plugin files to that directory
3. OpenCode will automatically detect the plugin

#### Manual Installation

```bash
# Clone the repository
git clone https://github.com/cristianoliveira/tmux-intray.git
cd tmux-intray

# Create plugin directory and copy plugin
mkdir -p ~/.config/opencode/plugins
cp opencode/plugins/opencode-tmux-intray.js ~/.config/opencode/plugins/
cp -r opencode/plugins/opencode-tmux-intray ~/.config/opencode/plugins/
```

#### Configure OpenCode

OpenCode automatically loads plugins from:
- Global: `~/.config/opencode/plugins/`
- Local: `$PWD/.opencode/plugins/`

If you prefer to specify the plugin path explicitly, add it to your OpenCode configuration file:

```json
{
  "plugins": [
    "/path/to/tmux-intray/opencode/plugins/opencode-tmux-intray.js"
  ]
}
```

### Uninstallation

To remove the plugin:

```bash
# Using the uninstall script
cd tmux-intray/opencode/plugins/opencode-tmux-intray
./uninstall.sh

# Or manually remove
rm -rf ~/.config/opencode/plugins/opencode-tmux-intray.js
rm -rf ~/.config/opencode/plugins/opencode-tmux-intray/
```

## Usage

Once installed, the plugin works automatically. OpenCode will load the plugin and start listening for events.

### Event Types

The plugin handles the following OpenCode events:

| Event Type | Status | Description |
|------------|--------|-------------|
| `session.idle` | `success` | Session completed successfully |
| `session.error` | `error` | Session encountered an error |
| `session.status` | `pending` | Session status changed to pending |
| `permission.updated` | `pending` | AI waiting for user input/permission |

### Notification Levels Mapping

The plugin maps event statuses to tmux-intray levels:

| Event Status | tmux-intray Level | Description |
|--------------|-------------------|-------------|
| `success`    | `info`            | Normal completion |
| `error`      | `error`           | Error occurred |
| `pending`    | `warning`         | Awaiting user input |

### Default Messages

- `session.idle`: "Task completed"
- `session.error`: "Session error"
- `session.status`: "Session status: pending" (only when status is pending)
- `permission.updated`: "Permission needed"

## Testing the Plugin

You can test the plugin using Vitest:

```bash
cd /path/to/tmux-intray/opencode/plugins/opencode-tmux-intray
npm run test:plugin
```

The test suite includes:
- `test-plugin.js`: Basic plugin functionality
- `integration.test.js`: Integration tests with tmux context capture
- `unit-context-capture.test.js`: Unit tests for context capture functions
- `test-real.js`: Real-world integration with tmux-intray (requires tmux-intray installed)

## Directory Structure

```
opencode/plugins/
├── opencode-tmux-intray.js          # Main plugin entry point
└── opencode-tmux-intray/            # Supporting files
    ├── README.md                    # This file
    ├── tests/                       # Test files
    │   ├── test-plugin.js
    │   ├── integration.test.js
    │   ├── unit-context-capture.test.js
    │   └── test-real.js
    ├── install.sh                   # Installation script
    └── uninstall.sh                 # Uninstallation script
```

## Running Tests

The plugin includes comprehensive tests using Vitest:

```bash
cd /path/to/tmux-intray/opencode/plugins/opencode-tmux-intray

# Install dependencies (first time only)
npm install

# Run all tests
npm test

# Run only the plugin tests
npm run test:plugin

# Run tests in watch mode
npm run test:watch

# Run tests with UI
npm run test:ui

# Generate coverage report
npm run test:coverage
```

Tests verify:
- Plugin loading and initialization
- Event handling and notification sending
- Tmux context capture (session/window/pane)
- Error handling

## Troubleshooting

### No notifications appear

1. **Check plugin loading**:
   - Verify OpenCode loads the plugin (check OpenCode logs)
   - Ensure the plugin path is correct in OpenCode configuration

2. **Check tmux-intray installation**:
   ```bash
   which tmux-intray
   tmux-intray --version
   ```

3. **Check tmux session**:
   - Ensure you're inside a tmux session
   - Verify `tmux display-message -p "#{session_name}"` returns a session name

4. **Enable debug logging**:
   - Plugin logs to `/tmp/opencode-tmux-intray.log`
   - Check this file for errors and debugging information

### Notifications have wrong content

Events use hardcoded messages:
- Only the events listed above are handled
- Messages cannot be customized (simplified design)

### Session detection issues

The plugin captures tmux context (session/window/pane) on each notification:
1. Checks tmux session name via `tmux display-message -p "#{session_name}"`
2. Checks tmux window ID via `tmux display-message -p "#{window_id}"`
3. Checks tmux pane ID via `tmux display-message -p "#{pane_id}"`

If you're not in a tmux session, the plugin will still work but won't attach context.

## Plugin API and Event Handling

The plugin implements the OpenCode plugin interface:

```javascript
export async function opencodeTmuxIntrayPlugin({ client }) {
  return {
    event: async ({ event }) => {
      // Handle OpenCode events
      // Calls tmux-intray add with appropriate level and message
    }
  };
}
```

### Event Flow

1. OpenCode emits an event
2. Plugin checks if event type is known
3. If known, determines message and status based on event type
4. Maps event status to tmux-intray level
5. Captures current tmux context (session/window/pane)
6. Calls `tmux-intray add --level=<level> --session=<id> --window=<id> --pane=<id> "<message>"`
7. Notification appears in tmux-intray

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This plugin is part of the tmux-intray project and is distributed under the same license. See the main project LICENSE file for details.

## Support

- **Issues**: Report bugs on the [GitHub issue tracker](https://github.com/cristianoliveira/tmux-intray/issues)
- **Documentation**: See the main [tmux-intray README](../../README.md)
- **Questions**: Open a discussion on GitHub
