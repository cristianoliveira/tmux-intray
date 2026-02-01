# OpenCode Tmux Intray Plugin

A plugin for OpenCode that integrates with tmux-intray to display notifications in tmux when OpenCode sessions complete, error, require permissions, or ask questions.

## Overview

The `opencode-tmux-intray` plugin connects OpenCode events to tmux-intray notifications. When OpenCode sessions emit events (like session completion, errors, or permission requests), the plugin sends notifications to tmux-intray, which displays them in your tmux status bar.

Key features:
- **Event-driven notifications**: Configurable notifications for OpenCode session events
- **Template messages**: Customizable notification messages with event property substitution
- **Session awareness**: Automatically detects tmux session for context
- **Configurable sound/TTS**: Optional sound and text-to-speech notifications (macOS)
- **Plugin hooks**: Integrates seamlessly with OpenCode's plugin system

## Installation

### Prerequisites

1. **OpenCode**: The plugin runs within OpenCode's plugin ecosystem
2. **tmux-intray CLI**: Must be installed and accessible in your PATH
3. **tmux session**: For notifications to appear, you should be running inside tmux

### Install the Plugin

#### Using the installation script (recommended)

The plugin includes an installation script that handles copying files, installing dependencies, and setting up npm scripts.

```bash
# Clone the repository (if you haven't already)
git clone https://github.com/cristianoliveira/tmux-intray.git
cd tmux-intray

# Navigate to the plugin directory
cd opencode/plugins/opencode-tmux-intray

# Run the installer with your preferred location
# For global installation (recommended):
./install.sh --global

# For local installation (current directory):
./install.sh --local

# Use --force to overwrite existing installation without prompting
# Use --no-deps to skip npm dependency installation
```

The script will:
1. Check for Node.js and npm
2. Copy the plugin files to the selected location:
   - Global: `~/.config/opencode/plugins/`
   - Local: `$PWD/.opencode/plugins/`
3. Install npm dependencies (production only)
4. Add npm scripts (`install-plugin`, `uninstall-plugin`) to package.json

After installation, OpenCode will automatically detect the plugin from the configured plugin directories.

#### Manual Installation

If you prefer to install manually:

```bash
# Clone the repository
git clone https://github.com/cristianoliveira/tmux-intray.git
cd tmux-intray

# The main plugin entry point is at opencode/plugins/opencode-tmux-intray.js
# Supporting modules are in opencode/plugins/opencode-tmux-intray/
# OpenCode will automatically load plugins from its configured plugin directories
```

You'll need to manually copy the plugin files to an OpenCode plugin directory:

```bash
# Copy to global plugin directory
mkdir -p ~/.config/opencode/plugins
cp opencode/plugins/opencode-tmux-intray.js ~/.config/opencode/plugins/
cp -R opencode/plugins/opencode-tmux-intray ~/.config/opencode/plugins/

# Or copy to local project directory
mkdir -p .opencode/plugins
cp opencode/plugins/opencode-tmux-intray.js .opencode/plugins/
cp -R opencode/plugins/opencode-tmux-intray .opencode/plugins/
```

Then install npm dependencies:

```bash
cd ~/.config/opencode/plugins/opencode-tmux-intray
npm install --production
```

#### Configure OpenCode

OpenCode automatically loads plugins from:
- Global: `~/.config/opencode/plugins/`
- Local: `$PWD/.opencode/plugins/`

If you installed using the installation script, no further configuration is needed.

If you prefer to specify the plugin path explicitly, add it to your OpenCode configuration file:

```json
{
  "plugins": [
    "/path/to/tmux-intray/opencode/plugins/opencode-tmux-intray.js"
  ]
}
```

### Uninstallation

To remove the plugin, use the uninstall script:

```bash
# Navigate to the plugin directory
cd /path/to/tmux-intray/opencode/plugins/opencode-tmux-intray

# Uninstall from global location
./uninstall.sh --global

# Or uninstall from local location
./uninstall.sh --local

# Use --force to skip confirmation prompt
```

Alternatively, manually remove the plugin files:

```bash
# Global installation
rm -rf ~/.config/opencode/plugins/opencode-tmux-intray.js ~/.config/opencode/plugins/opencode-tmux-intray/

# Local installation
rm -rf .opencode/plugins/opencode-tmux-intray.js .opencode/plugins/opencode-tmux-intray/
```

## Configuration

The plugin reads configuration from `~/.config/opencode-tmux-intray/opencode-config.json`. You can generate a starter configuration with:

```bash
cd /path/to/tmux-intray/opencode/plugins/opencode-tmux-intray
./generate-config.js
```

Or print it to stdout:

```bash
./generate-config.js --stdout
```

### Configuration Structure

The configuration file has the following structure:

```json
{
  "enabled": true,
  "agentName": "opencode",
  "notifications": {
    "session.idle": {
      "enabled": true,
      "message": "Task completed",
      "status": "success"
    },
    "session.error": {
      "enabled": true,
      "message": "Session error",
      "status": "error"
    },
    "session.status": {
      "enabled": false,
      "message": "Session status: {properties.status}",
      "status": "pending"
    },
    "session.created": {
      "enabled": false,
      "message": "New session created",
      "status": "success"
    },
    "session.updated": {
      "enabled": false,
      "message": "Session updated",
      "status": "success"
    },
    "permission.updated": {
      "enabled": true,
      "message": "Permission needed",
      "status": "pending"
    },
    "permission.replied": {
      "enabled": false,
      "message": "Permission replied",
      "status": "success"
    },
    "question.asked": {
      "enabled": false,
      "message": "Question asked: {question}",
      "status": "pending"
    },
    "permission.asked": {
      "enabled": false,
      "message": "Permission asked: {permission}",
      "status": "pending"
    }
  },
  "sound": {
    "enabled": true,
    "file": "/System/Library/Sounds/Glass.aiff"
  },
  "tts": {
    "enabled": false,
    "message": "Agent {agentName} completed with status {status}",
    "voice": "Alex"
  }
}
```

### Configuration Details

#### Global Settings
- `enabled`: Master toggle for the plugin (default: `true`)
- `agentName`: Name used in notifications (default: `"opencode"`)

#### Event Configuration
Each event type has:
- `enabled`: Whether notifications are sent for this event
- `message`: Template string with placeholders like `{property.path}`
- `status`: Maps to tmux-intray levels: `"success"` → `info`, `"error"` → `error`, `"pending"` → `warning`

#### Sound and TTS
- `sound`: Sound notification settings (macOS only)
- `tts`: Text-to-speech settings (macOS only)

### Template Substitution

Message templates support placeholders that reference properties from the OpenCode event object. Use curly braces with dot notation:

```json
{
  "message": "Session {sessionId} completed with status {properties.status}"
}
```

Available properties depend on the event type. Common properties include:
- `sessionId`: The OpenCode session ID
- `properties.*`: Event-specific properties
- `agentName`: From global configuration

## Usage

Once installed and configured, the plugin works automatically. OpenCode will load the plugin and start listening for events.

### Event Types

The plugin supports the following OpenCode events:

| Event Type | Default Status | Description |
|------------|----------------|-------------|
| `session.idle` | `success` | Session completed successfully |
| `session.error` | `error` | Session encountered an error |
| `session.status` | `pending` | Session status changed (e.g., waiting for input) |
| `session.created` | `success` | New session created |
| `session.updated` | `success` | Session updated with new messages |
| `permission.updated` | `pending` | AI waiting for user input/permission |
| `permission.replied` | `success` | Permission response received |
| `question.asked` | `pending` | AI asked a question |
| `permission.asked` | `pending` | AI asked for permission |

### Notification Levels Mapping

The plugin maps event statuses to tmux-intray levels:

| Event Status | tmux-intray Level | Description |
|--------------|-------------------|-------------|
| `success`    | `info`            | Normal completion |
| `error`      | `error`           | Error occurred |
| `pending`    | `warning`         | Awaiting user input |

### Testing the Plugin

You can test the plugin using Vitest. Note that `npm test` runs both the main project tests (via `make tests`) and the plugin tests (via Vitest). For plugin-only tests, use `npm run test:plugin`.

```bash
cd /path/to/tmux-intray/opencode/plugins/opencode-tmux-intray
npm run test:plugin
```

The test suite includes:
- `test-plugin.js`: Basic plugin functionality and session detection
- `test-config-loader.js`: Configuration loading and validation
- `test-integration.js`: Full integration tests with configuration scenarios
- `test-real.js`: Real-world integration with tmux-intray (requires tmux-intray installed)

## Directory Structure

The plugin consists of two parts:
1. **Main plugin entry point**: `opencode/plugins/opencode-tmux-intray.js` (loaded by OpenCode)
2. **Supporting modules**: Located in the `opencode/plugins/opencode-tmux-intray/` directory

```
opencode/plugins/
├── opencode-tmux-intray.js          # Main plugin entry point
└── opencode-tmux-intray/            # Supporting modules directory
    ├── README.md                    # This file
    ├── config-loader.js             # Configuration loading and validation
    ├── generate-config.js           # Configuration file generator
    ├── example-config.json          # Example configuration with comments
    └── tests/
        ├── test-plugin.js           # Basic plugin functionality tests
        ├── test-config-loader.js    # Configuration loading tests
        ├── test-integration.js      # Full integration tests
        └── test-real.js             # Real-world scenario tests
```

## Running Tests

The plugin includes comprehensive tests using Vitest. The test suite runs both the main project tests (via `make tests`) and the plugin tests (via Vitest). Run them with:

```bash
cd /path/to/tmux-intray/opencode/plugins/opencode-tmux-intray

# Install dependencies (first time only)
npm install

# Run all tests (main project tests + plugin tests)
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

You can also run plugin tests directly with Vitest:

```bash
npx vitest run
```

Tests verify:
- Plugin loading and initialization
- Configuration loading and validation
- Event handling and notification sending
- Session detection (tmux integration)
- Template substitution

## Migration Notes from agentmux-notify

If you're migrating from the older `agentmux-notify` plugin:

### Key Differences

1. **Configuration Location**: 
   - `agentmux-notify`: Used OpenCode's main configuration
   - `opencode-tmux-intray`: Separate config file at `~/.config/opencode-tmux-intray/opencode-config.json`

2. **Event Handling**:
   - `agentmux-notify`: Limited event types
   - `opencode-tmux-intray`: Comprehensive event support with template substitution

3. **tmux Integration**:
   - Both plugins use tmux-intray, but `opencode-tmux-intray` has better session detection and caching

### Migration Steps

1. **Generate new configuration**:
   ```bash
   ./generate-config.js
   ```

2. **Review and customize** the generated configuration to match your previous settings.

3. **Update OpenCode configuration** to use the new plugin path.

4. **Test** with a simple OpenCode session to ensure notifications work as expected.

### Configuration Mapping

If you had custom messages in `agentmux-notify`, you'll need to manually recreate them in the new configuration format. The new template system is more powerful, allowing property substitution.

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
   - Verify `tmux display-message -p "#S"` returns a session name

4. **Check configuration**:
   - Ensure `enabled: true` in config
   - Verify specific events are enabled

5. **Enable debug logging**:
   - Plugin logs errors to `.tmp/debug.log` in the current directory
   - Notifications are logged to `/tmp/opencode-tmux-intray.log`

### Notifications have wrong content

1. **Check template syntax**:
   - Placeholders use `{property.path}` syntax
   - Properties must exist in the event object

2. **Check event properties**:
   - Different events have different properties
   - Use the example config as reference

### Sound/TTS not working

1. **macOS only**: Sound and TTS features are macOS-specific
2. **Check file paths**: Sound files must exist at the specified path
3. **Check permissions**: The plugin needs permission to play sounds

### Session detection issues

The plugin caches the tmux session at initialization. If you change tmux sessions after OpenCode starts, notifications may use the wrong session. Restart OpenCode to refresh the session cache.

## Plugin API and Event Handling

The plugin implements the OpenCode plugin interface:

```javascript
export async function opencodeTmuxIntrayPlugin({ client }) {
  return {
    event: async ({ event }) => {
      // Handle OpenCode events
    }
  };
}
```

### Event Flow

1. OpenCode emits an event
2. Plugin checks if event is enabled in configuration
3. If enabled, builds message using template substitution
4. Maps event status to tmux-intray level
5. Calls `tmux-intray add --level=<level> "<message>"`
6. Notification appears in tmux-intray

### Session Detection

The plugin detects tmux sessions by:
1. Checking `TMUX` environment variable
2. Falling back to `tmux display-message -p "#S"` command
3. Caching the session at plugin initialization for performance

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