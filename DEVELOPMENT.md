# Development Guide

## Project Structure

```
tmux-intray/
├── bin/
│   └── tmux-intray           # Main CLI entry point
├── commands/                 # Individual command implementations
│   ├── add.sh               # add command (with sub-modules)
│   ├── add/                 # Add command's private modules
│   │   └── modules/
│   │       ├── validators.sh
│   │       └── formatters.sh
│   ├── show.sh              # show command (with sub-modules)
│   ├── show/                # Show command's private modules
│   │   └── modules/
│   │       ├── filters.sh
│   │       └── display.sh
│   ├── clear.sh             # clear command
│   ├── toggle.sh            # toggle command
│   ├── help.sh              # help command
│   └── version.sh           # version command
├── lib/                      # Shared libraries (global)
│   ├── core.sh              # Core functions (tmux interaction)
│   ├── colors.sh            # Color utilities
│   └── tmux-intray.sh       # Legacy compatibility
├── tests/                    # Test suite
│   ├── basic.bats           # Basic CLI tests
│   ├── cli.bats             # CLI interface tests
│   ├── tray.bats            # Tray management tests
│   └── commands/            # Command-specific tests
│       ├── add.bats
│       ├── show.bats
│       └── management.bats
├── scripts/
│   ├── lint.sh              # ShellCheck linter
│   └── security-check.sh    # Security-focused ShellCheck
├── tmux-intray.tmux         # Tmux plugin entry point
├── Makefile                 # Build automation
└── flake.nix                # Nix flake for dev environment
```

## Adding a New Command

### Simple Command (No Sub-modules)

1. Create a new file in `commands/` directory:
   ```bash
   touch commands/mycommand.sh
   chmod +x commands/mycommand.sh
   ```

2. Implement the command with the naming convention `<command>_command`:
   ```bash
   #!/usr/bin/env bash
   # My command - Description of what it does

   mycommand_command() {
       # Your command logic here
       # You can use functions from lib/
       ensure_tmux_running
       success "Command executed"
   }
   ```

3. Add the command to the main CLI in `bin/tmux-intray`:
   ```bash
   case "$command" in
       show|add|clear|toggle|help|version|mycommand)
           source "$COMMANDS_DIR/${command}.sh"
           "${command}_command" "$@"
           ;;
       ...
   esac
   ```

4. Update the help text in `bin/tmux-intray`:
   ```bash
   COMMANDS:
       ...
       mycommand    Description of the command
   ```

5. Create tests in `tests/commands/mycommand.bats`:
   ```bash
   #!/usr/bin/env bats
   # My command tests

   @test "mycommand does something" {
       run ./bin/tmux-intray mycommand
       [ "$status" -eq 0 ]
   }
   ```

### Complex Command with Sub-modules

For complex commands that need their own modules:

1. Create the command directory structure:
   ```bash
   mkdir -p commands/mycommand/modules
   touch commands/mycommand.sh
   chmod +x commands/mycommand.sh
   ```

2. Create sub-modules:
   ```bash
   # commands/mycommand/modules/helper.sh
   #!/usr/bin/env bash

   helper_function() {
       echo "Helper result"
   }
   ```

3. Source modules in the command:
   ```bash
   #!/usr/bin/env bash
   # My command - Complex command with sub-modules

   # Source local modules
   # shellcheck disable=SC1091
   COMMAND_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
   source "$COMMAND_DIR/mycommand/modules/helper.sh"

   mycommand_command() {
       # Use local module functions
       helper_function
       # Also use global lib functions
       ensure_tmux_running
   }
   ```

4. The rest is same as simple command (register, test, etc.)

## Libraries

### lib/core.sh
Core tmux interaction functions:
- `ensure_tmux_running()` - Check if tmux is running
- `get_tray_items()` - Get current tray items
- `add_tray_item()` - Add an item to the tray
- `clear_tray_items()` - Clear all tray items
- `get_visibility()` - Get tray visibility state
- `set_visibility()` - Set tray visibility state

### lib/colors.sh
Color output utilities:
- `error()` - Print error message (red)
- `success()` - Print success message (green)
- `warning()` - Print warning message (yellow)
- `info()` - Print info message (blue)

## Development

```bash
# Enter dev environment with tools (bats, shellcheck)
nix develop

# Run all tests
make tests

# Run linter
make lint

# Run security check
make security-check

# Run both tests and lint
make all

# Run specific test file
bats tests/basic.bats
```

## Architecture

The CLI follows a modular pattern similar to Go's Cobra CLI:

1. **Main entry point** (`bin/tmux-intray`):
   - Parses command-line arguments
   - Loads the appropriate command file
   - Executes the command function

2. **Command files** (`commands/*.sh`):
   - Each file implements a single command
   - Functions follow `<command>_command` naming
   - Can source global libraries from `lib/`
   - Can have their own sub-modules in `commands/<command>/modules/`

3. **Global Libraries** (`lib/*.sh`):
   - Shared utilities and core logic
   - Can be sourced by any command
   - Use for functions used across multiple commands

4. **Command-Specific Modules** (`commands/<command>/modules/*.sh`):
   - Private helper functions for a specific command
   - Keep complex commands organized and maintainable
   - Only accessible to that command

5. **Tests** (`tests/**/*.bats`):
   - Organized by feature/command
   - Use Bats (Bash Automated Testing System)

This structure makes the codebase:
- ✅ Easy to maintain (small, focused files)
- ✅ Easy to extend (add new commands without touching existing ones)
- ✅ Easy to test (command-specific tests)
- ✅ Easy to understand (clear separation of concerns)
- ✅ Scalable (complex commands can have their own modules)
- ✅ Modular (commands are self-contained)

### Example: Add Command Structure

```
commands/
├── add.sh                      # Main entry point
└── add/                        # Private modules
    └── modules/
        ├── validators.sh        # Input validation logic
        └── formatters.sh       # Message formatting logic
```

**add.sh** uses modules:
```bash
# Source local modules
source "$COMMAND_DIR/add/modules/validators.sh"
source "$COMMAND_DIR/add/modules/formatters.sh"

add_command() {
    validate_message "$1"           # From validators.sh
    format_message "$1"            # From formatters.sh
    add_tray_item "$..."           # From lib/core.sh
}
```
