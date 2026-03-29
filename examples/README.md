# Examples

This directory contains example scripts and configurations demonstrating how to use and integrate tmux-intray.

## Quick Start

All examples assume `tmux-intray` is installed and available in your PATH:

```bash
# Check if installed
tmux-intray --version

# If not installed:
go install github.com/cristianoliveira/tmux-intray@latest
```

## Examples

### Basic Usage

**File**: `basic-usage.sh`

Demonstrates adding notifications and listing them from a script.

```bash
./basic-usage.sh
```

**What it shows**:
- Checking if tmux is running
- Adding a simple notification
- Listing notifications in table format

### Process Notifications

**File**: `process-notifications.sh`

Shows how to use tmux-intray in a long-running process with status updates.

```bash
./process-notifications.sh
```

**What it shows**:
- Notify process start
- Send progress updates
- Notify completion
- List final tray contents

### CI/CD Pipeline Integration

**File**: `ci-pipeline.sh`

Demonstrates integrating tmux-intray into a CI/CD pipeline for build/test notifications.

```bash
./ci-pipeline.sh "MyProject"
```

**What it shows**:
- Build stage notifications
- Test stage notifications
- Failure handling with notifications
- Final status reporting

### Hooks

**Directory**: `hooks/`

Contains example hook scripts for extending tmux-intray with custom automation.

**Hook scripts include**:
- `01-log.sh` - Log events to file
- `02-macos-notification.sh` - macOS desktop notifications
- `03-tmux-status-bar.sh` - Tmux status bar alerts
- `04-linux-notification.sh` - Linux desktop notifications
- `05-macos-sound.sh` - macOS sound alerts
- `06-linux-sound.sh` - Linux sound alerts

See [hooks/README.md](hooks/README.md) for detailed hook documentation.

**Installation**:
```bash
mkdir -p ~/.config/tmux-intray/hooks
cp -r hooks/* ~/.config/tmux-intray/hooks/
chmod +x ~/.config/tmux-intray/hooks/*/*.sh
```

### Advanced Filtering

**File**: `advanced-filtering.sh`

Comprehensive examples of filtering and searching notifications.

```bash
# View examples (not executable, just documentation)
./advanced-filtering.sh
```

**What it shows**:
- Filter by session, level, time
- Search for patterns
- Regex searches
- Group notifications
- Complex filter combinations

## Running Examples

### Prerequisites

1. **tmux is running**:
   ```bash
   tmux new-session -d -s example
   ```

2. **tmux-intray is installed**:
   ```bash
   go install github.com/cristianoliveira/tmux-intray@latest
   ```

### Executing Examples

```bash
# Basic usage
./basic-usage.sh

# Process notifications
./process-notifications.sh

# CI pipeline
./ci-pipeline.sh "MyProject"

# View advanced filtering examples
cat advanced-filtering.sh
```

## Customization

All examples are designed to be modified for your needs:

- **Change messages**: Edit notification text to match your use case
- **Add filters**: Use `--level`, `--session`, `--pane` flags
- **Custom formats**: Use `--format` flag with templates
- **Integrate**: Combine with other tools (`jq`, `fzf`, `awk`)

## Related Documentation

- [Status Guide](../docs/status-guide.md) - Template variables and formatting
- [CLI Reference](../docs/cli/CLI_REFERENCE.md) - All commands and options
- [Hooks Guide](../docs/hooks.md) - Complete hooks system documentation
- [Configuration Guide](../docs/configuration.md) - Environment variables and settings
- [Troubleshooting](../docs/troubleshooting.md) - Common issues and solutions

## Contributing Examples

Have an example you'd like to share? Please:

1. Create a new script file
2. Add clear comments explaining what it does
3. Add it to this README with a brief description
4. Submit a pull request

We welcome examples that demonstrate:
- Real-world use cases
- Integration with other tools
- Common workflows
- Best practices
