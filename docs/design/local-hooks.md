# Local Hooks Design

**Status**: *Design* - Proposed enhancement

## Overview

This document describes the design for adding local hooks support to tmux-intray. Local hooks allow tmux-intray to load and execute hooks from the current working directory (`.tmux-intray/hooks/`) in addition to global hooks (`~/.config/tmux-intray/hooks/`). This enables project-specific hook configurations while maintaining a global set of personal hooks.

## Goals

1. **Enable project-specific hooks**: Allow teams to define hooks for specific projects that can be committed to version control
2. **Maintain backward compatibility**: Existing global hooks should continue to work without any changes
3. **Flexible execution order**: Provide clear and predictable hook execution order when both local and global hooks exist
4. **Security by default**: Protect users from executing untrusted hooks in untrusted directories
5. **Per-directory control**: Allow fine-grained control over which hook points use local hooks

## Current Implementation Summary

The hooks system currently loads hooks from a single directory:

```go
// internal/hooks/hooks.go
func getHooksDir() string {
    config.Load()
    // First check environment variable (highest precedence)
    if dir := os.Getenv("TMUX_INTRAY_HOOKS_DIR"); dir != "" {
        return dir
    }
    // Then check config
    if dir := config.Get("hooks_dir", ""); dir != "" {
        colors.Debug(fmt.Sprintf("hooks_dir from config: %s", dir))
        return dir
    }
    // Default: $XDG_CONFIG_HOME/tmux-intray/hooks
    if configDir := os.Getenv("XDG_CONFIG_HOME"); configDir != "" {
        return filepath.Join(configDir, "tmux-intray", "hooks")
    }
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".config", "tmux-intray", "hooks")
}
```

The `Run()` function executes hooks from a single directory for a given hook point, supporting synchronous and asynchronous execution with configurable failure modes.

## Design

### Directory Structure

Local hooks are stored in a `.tmux-intray/hooks/` directory within the current working directory:

```
project-directory/
├── .tmux-intray/
│   └── hooks/
│       ├── pre-add/
│       │   ├── 01-validate.sh
│       │   └── 02-project-specific-check.sh
│       ├── post-add/
│       │   └── 01-notify-team.sh
│       ├── pre-dismiss/
│       └── post-dismiss/
└── ...other project files...
```

Global hooks remain in the existing location:

```
~/.config/tmux-intray/hooks/
├── pre-add/
│   ├── 01-validate.sh
│   ├── 02-enrich.sh
│   └── 99-log.sh
├── post-add/
│   └── 99-log.sh
├── pre-dismiss/
│   └── 01-confirm.sh
└── ...
```

### Hook Loading and Execution Order

#### Loading Order

1. **Local hooks**: Load from `.tmux-intray/hooks/` in the current working directory
2. **Global hooks**: Load from `~/.config/tmux-intray/hooks/` (or configured directory)

Both sets of hooks are merged and available for execution.

#### Execution Order

For each hook point, hooks execute in the following order:

1. **Local hooks first** (in sorted alphabetical order)
2. **Global hooks second** (in sorted alphabetical order)

Example for `pre-add` hook point:
```
1. .tmux-intray/hooks/pre-add/01-validate.sh      (local)
2. .tmux-intray/hooks/pre-add/02-project-check.sh (local)
3. ~/.config/tmux-intray/hooks/pre-add/01-enrich.sh  (global)
4. ~/.config/tmux-intray/hooks/pre-add/99-log.sh     (global)
```

**Rationale**: Local hooks run first to allow project-specific validation and enrichment before global hooks, which typically handle logging and generic notifications.

### Configuration Options

#### Enable/Disable Local Hooks

New configuration options to control local hooks:

| Configuration Key | Environment Variable | Default | Description |
|-------------------|---------------------|---------|-------------|
| `hooks_local_enabled` | `TMUX_INTRAY_HOOKS_LOCAL_ENABLED` | `true` | Enable/disable local hooks globally |
| `hooks_local_trusted` | `TMUX_INTRAY_HOOKS_LOCAL_TRUSTED` | `false` | Skip security warning for untrusted directories |

#### Per-Hook-Point Control

Local hooks can be enabled/disabled per hook point:

| Configuration Key | Environment Variable | Default | Description |
|-------------------|---------------------|---------|-------------|
| `hooks_local_enabled_pre_add` | `TMUX_INTRAY_HOOKS_LOCAL_ENABLED_pre_add` | `true` | Enable local hooks for pre-add |
| `hooks_local_enabled_post_add` | `TMUX_INTRAY_HOOKS_LOCAL_ENABLED_post_add` | `true` | Enable local hooks for post-add |
| `hooks_local_enabled_pre_dismiss` | `TMUX_INTRAY_HOOKS_LOCAL_ENABLED_pre_dismiss` | `true` | Enable local hooks for pre-dismiss |
| `hooks_local_enabled_post_dismiss` | `TMUX_INTRAY_HOOKS_LOCAL_ENABLED_post_dismiss` | `true` | Enable local hooks for post-dismiss |
| `hooks_local_enabled_cleanup` | `TMUX_INTRAY_HOOKS_LOCAL_ENABLED_cleanup` | `true` | Enable local hooks for cleanup |

Configuration file example (`~/.config/tmux-intray/config.toml`):

```toml
# Disable local hooks for all hook points
hooks_local_enabled = false

# Or enable local hooks only for specific hook points
hooks_local_enabled = true
hooks_local_enabled_pre_add = true
hooks_local_enabled_post_add = false
hooks_local_enabled_pre_dismiss = true
hooks_local_enabled_post_dismiss = true
hooks_local_enabled_cleanup = false
```

Environment variable example:

```bash
# Disable all local hooks
export TMUX_INTRAY_HOOKS_LOCAL_ENABLED=0

# Disable only post-add local hooks
export TMUX_INTRAY_HOOKS_LOCAL_ENABLED_post_add=0
```

### Security Considerations

#### Threat Model

Local hooks execute automatically when tmux-intray runs in a directory containing `.tmux-intray/hooks/`. This introduces several security concerns:

1. **Malicious hook scripts**: A compromised repository could contain malicious hooks
2. **Privilege escalation**: Hooks run with the user's permissions
3. **Supply chain attacks**: Hooks in third-party code could be malicious
4. **Information disclosure**: Hooks could exfiltrate sensitive data

#### Security Controls

##### 1. Opt-in by Default

Local hooks are **enabled by default** for convenience but users can disable them globally:

```bash
export TMUX_INTRAY_HOOKS_LOCAL_ENABLED=0
```

**Rationale**: Defaulting to enabled reduces friction for legitimate use cases. Users who work in untrusted directories can disable them.

##### 2. Untrusted Directory Warning

When executing local hooks in an untrusted directory, display a warning:

```
warning: executing local hooks from untrusted directory: /path/to/project
review hooks in .tmux-intray/hooks/ before proceeding
set TMUX_INTRAY_HOOKS_LOCAL_TRUSTED=1 to suppress this warning
```

A directory is considered **trusted** if:
- User has explicitly marked it trusted via environment variable, OR
- Directory is within a trusted path (e.g., home directory, user-controlled directories)

##### 3. Trust Configuration

Users can mark directories as trusted via configuration:

```toml
# List of trusted directory paths
hooks_local_trusted_paths = [
    "/Users/username/projects",
    "/home/username/work",
]
```

Or via environment variable:

```bash
export TMUX_INTRAY_HOOKS_LOCAL_TRUSTED_PATHS="/home/user/projects:/home/user/work"
```

##### 4. Hook Audit Mode

Optional audit mode to preview hooks without executing:

```bash
export TMUX_INTRAY_HOOKS_AUDIT=1
tmux-intray add "test"
```

Output:
```
[AUDIT MODE] Would execute hooks for 'pre-add':
  .tmux-intray/hooks/pre-add/01-validate.sh
  .tmux-intray/hooks/pre-add/02-project-check.sh
  ~/.config/tmux-intray/hooks/pre-add/01-enrich.sh
  ~/.config/tmux-intray/hooks/pre-add/99-log.sh
[AUDIT MODE] No hooks were executed. Remove TMUX_INTRAY_HOOKS_AUDIT to execute.
```

##### 5. Hook Signature Verification (Future Enhancement)

Optional hook signature verification for high-security environments:

```toml
hooks_local_verify_signatures = true
hooks_local_public_key = "/path/to/public_key.pem"
```

#### Security Best Practices

1. **Review hooks before use**: Always review local hooks before enabling in untrusted directories
2. **Commit hooks to version control**: Store hooks in `.tmux-intray/hooks/` and review them in pull requests
3. **Use audit mode**: Preview hooks before executing in new directories
4. **Disable when not needed**: Set `TMUX_INTRAY_HOOKS_LOCAL_ENABLED=0` in untrusted environments
5. **Limit hook permissions**: Use restrictive file permissions on hook directories

### Implementation Approach

#### Phase 1: Core Infrastructure

1. **Add configuration support**:
   ```go
   // internal/config/config.go
   func setDefaults() {
       // ... existing defaults ...

       // Local hooks configuration
       setDefault("hooks_local_enabled", "true")
       setDefault("hooks_local_trusted", "false")
       setDefault("hooks_local_enabled_pre_add", "true")
       setDefault("hooks_local_enabled_post_add", "true")
       setDefault("hooks_local_enabled_pre_dismiss", "true")
       setDefault("hooks_local_enabled_post_dismiss", "true")
       setDefault("hooks_local_enabled_cleanup", "true")
       setDefault("hooks_local_audit", "false")
   }
   ```

2. **Add hooks directory helpers**:
   ```go
   // internal/hooks/hooks.go

   // getLocalHooksDir returns the local hooks directory path from CWD
   func getLocalHooksDir() string {
       cwd, err := os.Getwd()
       if err != nil {
           return ""
       }
       return filepath.Join(cwd, ".tmux-intray", "hooks")
   }

   // getGlobalHooksDir returns the global hooks directory path
   func getGlobalHooksDir() string {
       // ... existing getHooksDir() logic ...
   }

   // getHooksDirs returns both local and global hooks directories
   func getHooksDirs() []string {
       dirs := []string{}

       // Add local hooks if enabled
       if config.GetBool("hooks_local_enabled", true) {
           if localDir := getLocalHooksDir(); localDir != "" {
               dirs = append(dirs, localDir)
           }
       }

       // Add global hooks
       if globalDir := getGlobalHooksDir(); globalDir != "" {
           dirs = append(dirs, globalDir)
       }

       return dirs
   }
   ```

3. **Update Run() to support multiple directories**:
   ```go
   // internal/hooks/hooks.go

   // Run executes hooks for a hook point from multiple directories
   func Run(hookPoint string, envVars ...string) error {
       dirs := getHooksDirs()

       // Build environment map
       envMap := buildEnvMap(hookPoint, envVars...)

       // Check if local hooks are enabled for this hook point
       localEnabledForHookPoint := isLocalHooksEnabled(hookPoint)

       // Collect and execute hooks from all directories
       for _, dir := range dirs {
           isLocal := dir == getLocalHooksDir()
           if isLocal && !localEnabledForHookPoint {
               continue
           }

           hooks := collectHooks(dir, hookPoint)
           if err := executeHooks(hooks, envMap, isLocal); err != nil {
               return err
           }
       }

       return nil
   }

   func buildEnvMap(hookPoint string, envVars ...string) map[string]string {
       // ... existing envMap building logic ...
       // Add LOCAL_HOOKS_DIR and GLOBAL_HOOKS_DIR
       envMap["LOCAL_HOOKS_DIR"] = getLocalHooksDir()
       envMap["GLOBAL_HOOKS_DIR"] = getGlobalHooksDir()
       return envMap
   }

   func isLocalHooksEnabled(hookPoint string) bool {
       if !config.GetBool("hooks_local_enabled", true) {
           return false
       }
       key := fmt.Sprintf("hooks_local_enabled_%s", strings.ToLower(hookPoint))
       return config.GetBool(key, true)
   }
   ```

#### Phase 2: Security Features

1. **Add trusted directory checking**:
   ```go
   // internal/hooks/hooks.go

   func isDirectoryTrusted(dir string) bool {
       // Check if explicitly trusted
       if config.GetBool("hooks_local_trusted", false) {
           return true
       }

       // Check if directory is in trusted paths list
       trustedPaths := config.Get("hooks_local_trusted_paths", "")
       if trustedPaths != "" {
           for _, trustedPath := range strings.Split(trustedPaths, ":") {
               if strings.HasPrefix(dir, trustedPath) {
                   return true
               }
           }
       }

       // Default: home directory is trusted
       home, _ := os.UserHomeDir()
       return strings.HasPrefix(dir, home)
   }
   ```

2. **Add warning for untrusted directories**:
   ```go
   func executeHooks(hooks []hookInfo, envMap map[string]string, isLocal bool) error {
       if isLocal && !isDirectoryTrusted(getLocalHooksDir()) {
           fmt.Fprintf(os.Stderr,
               "warning: executing local hooks from untrusted directory: %s\n"+
               "  review hooks in .tmux-intray/hooks/ before proceeding\n"+
               "  set TMUX_INTRAY_HOOKS_LOCAL_TRUSTED=1 to suppress this warning\n",
               getLocalHooksDir(),
           )
       }

       // ... existing hook execution logic ...
   }
   ```

3. **Add audit mode**:
   ```go
   func Run(hookPoint string, envVars ...string) error {
       if config.GetBool("hooks_local_audit", false) {
           return auditHooks(hookPoint)
       }

       // ... existing run logic ...
   }

   func auditHooks(hookPoint string) error {
       dirs := getHooksDirs()

       fmt.Fprintf(os.Stderr, "[AUDIT MODE] Would execute hooks for '%s':\n", hookPoint)

       for _, dir := range dirs {
           isLocal := dir == getLocalHooksDir()
           if isLocal && !isLocalHooksEnabled(hookPoint) {
               continue
           }

           hooks := collectHooks(dir, hookPoint)
           for _, hook := range hooks {
               fmt.Fprintf(os.Stderr, "  %s\n", hook.path)
           }
       }

       fmt.Fprintf(os.Stderr, "[AUDIT MODE] No hooks were executed. Remove TMUX_INTRAY_HOOKS_AUDIT to execute.\n")
       return nil
   }
   ```

#### Phase 3: Documentation and Examples

1. **Update hooks documentation**: Add local hooks section to `docs/hooks.md`
2. **Create example hooks**: Add project-specific hook examples in `examples/hooks/local/`
3. **Add security guide**: Create `docs/security.md` with hook security best practices

### Use Cases

#### 1. Project-Specific Notifications

A team working on a critical project can add project-specific validation:

```bash
# .tmux-intray/hooks/pre-add/01-project-check.sh
#!/usr/bin/env bash
# Ensure only authorized users can add notifications for this project

if [[ "$NOTIFICATION_LEVEL" == "critical" ]]; then
    if [[ "$(whoami)" != "prod-ops" ]]; then
        echo "ERROR: Only prod-ops can add critical notifications" >&2
        exit 1
    fi
fi
```

#### 2. Team-Shared Hooks

Hooks committed to the repository ensure consistent behavior across the team:

```bash
# .tmux-intray/hooks/post-add/01-notify-team.sh
#!/usr/bin/env bash
# Send Slack notification for critical alerts

if [[ "$NOTIFICATION_LEVEL" == "critical" ]]; then
    curl -X POST -H 'Content-type: application/json' \
        --data "{\"text\":\"🚨 [$PROJECT_NAME] $NOTIFICATION_MESSAGE\"}" \
        "$SLACK_WEBHOOK_URL" >/dev/null 2>&1 &
fi
```

#### 3. Environment-Specific Behavior

Different behavior in development vs production:

```bash
# .tmux-intray/hooks/pre-add/01-env-check.sh
#!/usr/bin/env bash
# Different notification levels based on environment

if [[ "$ENVIRONMENT" == "production" ]]; then
    # Upgrade all warnings to errors in production
    if [[ "$NOTIFICATION_LEVEL" == "warning" ]]; then
        export NOTIFICATION_LEVEL="error"
    fi
fi
```

#### 4. Project-Specific Sound Notifications

Custom sound notifications for different projects:

```bash
# .tmux-intray/hooks/post-add/01-project-sound.sh
#!/usr/bin/env bash
# Play project-specific sound

if [[ "$PROJECT_NAME" == "alerting-service" ]]; then
    afplay /path/to/alerting-sound.aiff 2>/dev/null || true
elif [[ "$PROJECT_NAME" == "monitoring" ]]; then
    afplay /path/to/monitoring-sound.aiff 2>/dev/null || true
fi
```

### Testing Strategy

#### Unit Tests

1. **Directory resolution**:
   ```go
   func TestGetLocalHooksDir(t *testing.T) {
       tmpDir := t.TempDir()
       oldWd, _ := os.Getwd()
       defer os.Chdir(oldWd)

       os.Chdir(tmpDir)
       localDir := getLocalHooksDir()
       assert.Equal(t, filepath.Join(tmpDir, ".tmux-intray", "hooks"), localDir)
   }
   ```

2. **Multiple directory loading**:
   ```go
   func TestRunHooksFromMultipleDirs(t *testing.T) {
       // Create local hooks
       localDir := t.TempDir()
       localHook := filepath.Join(localDir, "pre-add", "01-local.sh")
       os.MkdirAll(filepath.Dir(localHook), 0755)
       os.WriteFile(localHook, []byte("#!/bin/sh\necho local"), 0755)

       // Create global hooks
       globalDir := t.TempDir()
       globalHook := filepath.Join(globalDir, "pre-add", "01-global.sh")
       os.MkdirAll(filepath.Dir(globalHook), 0755)
       os.WriteFile(globalHook, []byte("#!/bin/sh\necho global"), 0755)

       // Set up environment
       os.Setenv("TMUX_INTRAY_HOOKS_DIR", globalDir)
       os.Setenv("PWD", filepath.Dir(localDir))

       // Run hooks and verify both execute
       // ...
   }
   ```

3. **Execution order**:
   ```go
   func TestHookExecutionOrder(t *testing.T) {
       // Create hooks and capture execution order
       // Verify local hooks execute before global hooks
       // ...
   }
   ```

4. **Per-hook-point enable/disable**:
   ```go
   func TestLocalHooksPerHookPoint(t *testing.T) {
       // Test that hooks_local_enabled_pre_add controls local hooks for pre-add
       // ...
   }
   ```

#### Integration Tests

1. **Security warning**:
   ```go
   func TestUntrustedDirectoryWarning(t *testing.T) {
       // Create local hooks in untrusted directory
       // Verify warning is displayed
       // ...
   }
   ```

2. **Audit mode**:
   ```go
   func TestAuditMode(t *testing.T) {
       // Enable audit mode
       // Verify hooks are listed but not executed
       // ...
   }
   ```

#### Bats Tests

1. **End-to-end workflow**:
   ```bash
   @test "local hooks execute before global hooks" {
       # Create local and global hooks
       # Run tmux-intray add
       # Verify execution order
   }
   ```

2. **Configuration override**:
   ```bash
   @test "can disable local hooks via environment variable" {
       export TMUX_INTRAY_HOOKS_LOCAL_ENABLED=0
       # Create local hooks
       # Verify local hooks don't execute
   }
   ```

### Migration Guide

#### For Existing Users

Existing users with global hooks will experience no changes. Local hooks are opt-in and don't affect existing behavior.

#### For Users Wanting to Use Local Hooks

1. **Create local hooks directory**:
   ```bash
   mkdir -p .tmux-intray/hooks/pre-add
   ```

2. **Add project-specific hooks**:
   ```bash
   cat > .tmux-intray/hooks/pre-add/01-project-check.sh <<'EOF'
   #!/usr/bin/env bash
   # Your project-specific logic
   EOF
   chmod +x .tmux-intray/hooks/pre-add/01-project-check.sh
   ```

3. **Test hooks**:
   ```bash
   # Preview hooks without executing
   export TMUX_INTRAY_HOOKS_AUDIT=1
   tmux-intray add "test notification"

   # Execute hooks
   unset TMUX_INTRAY_HOOKS_AUDIT
   tmux-intray add "test notification"
   ```

4. **Commit to version control** (optional):
   ```bash
   git add .tmux-intray/
   git commit -m "Add project-specific hooks"
   ```

#### For Users Wanting to Disable Local Hooks

Add to `~/.config/tmux-intray/config.toml`:

```toml
hooks_local_enabled = false
```

Or set environment variable:

```bash
export TMUX_INTRAY_HOOKS_LOCAL_ENABLED=0
```

### Backward Compatibility

The design maintains full backward compatibility:

1. **Existing global hooks**: Continue to work without any changes
2. **Configuration options**: Default to current behavior (local hooks enabled, but no impact if none exist)
3. **Environment variables**: New variables don't affect existing functionality
4. **API compatibility**: `Run()` function signature unchanged

### Performance Considerations

1. **Directory scanning**: Hooks are collected from both directories on each call, which adds minimal overhead (single filesystem read per directory)
2. **Execution order**: Local hooks run first, adding no performance impact
3. **Configuration check**: Per-hook-point enable/disable checks are O(1) config lookups
4. **No file watching**: No file watching overhead; hooks are scanned on each invocation

### Future Enhancements

1. **Hook inheritance**: Allow parent directories to inherit hooks from ancestor directories
2. **Hook composition**: Define hook templates and compose them in local hooks
3. **Hook debugging**: Add `--debug-hooks` flag to show detailed hook execution information
4. **Hook dependencies**: Define dependencies between hooks for complex workflows
5. **Hook marketplace**: Share and discover community hooks

## References

- [Hooks System Documentation](../hooks.md)
- [Go Package Structure](./go-package-structure.md)
- [Configuration Guide](../configuration.md)
- [Testing Strategy](../testing/testing-strategy.md)

## Appendix

### Example Local Hook Setup

#### Step 1: Create directory structure

```bash
mkdir -p .tmux-intray/hooks/{pre-add,post-add,pre-dismiss,post-dismiss,cleanup}
```

#### Step 2: Add project-specific validation

```bash
cat > .tmux-intray/hooks/pre-add/01-validate.sh <<'EOF'
#!/usr/bin/env bash
# Validate notification content for this project

# Prevent empty notifications
if [[ -z "$NOTIFICATION_MESSAGE" ]]; then
    echo "ERROR: Notification message cannot be empty" >&2
    exit 1
fi

# Prevent notifications with keywords that should be excluded
for keyword in "password" "secret" "token"; do
    if [[ "$NOTIFICATION_MESSAGE" =~ $keyword ]]; then
        echo "ERROR: Notification contains sensitive data ($keyword)" >&2
        exit 1
    fi
done
EOF
chmod +x .tmux-intray/hooks/pre-add/01-validate.sh
```

#### Step 3: Add project-specific enrichment

```bash
cat > .tmux-intray/hooks/pre-add/02-enrich.sh <<'EOF'
#!/usr/bin/env bash
# Add project metadata to notifications

PROJECT_NAME=$(basename "$(git rev-parse --show-toplevel 2>/dev/null || echo 'unknown')")
export NOTIFICATION_MESSAGE="[$PROJECT_NAME] $NOTIFICATION_MESSAGE"
EOF
chmod +x .tmux-intray/hooks/pre-add/02-enrich.sh
```

#### Step 4: Add project-specific logging

```bash
cat > .tmux-intray/hooks/post-add/01-log.sh <<'EOF'
#!/usr/bin/env bash
# Log notifications to project-specific log file

PROJECT_NAME=$(basename "$(git rev-parse --show-toplevel 2>/dev/null || echo 'unknown')")
LOG_FILE=".tmux-intray/notifications.log"

mkdir -p "$(dirname "$LOG_FILE")"
{
    echo "$(date -u +"%Y-%m-%dT%H:%M:%SZ") [$PROJECT_NAME] $NOTIFICATION_LEVEL: $NOTIFICATION_MESSAGE"
} >>"$LOG_FILE"
EOF
chmod +x .tmux-intray/hooks/post-add/01-log.sh
```

#### Step 5: Add to .gitignore (if not committing hooks)

```bash
cat >> .gitignore <<'EOF'
# Local hooks (optional - remove if committing hooks)
.tmux-intray/hooks/
EOF
```

### Environment Variables Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `TMUX_INTRAY_HOOKS_LOCAL_ENABLED` | `true` | Enable/disable local hooks globally |
| `TMUX_INTRAY_HOOKS_LOCAL_TRUSTED` | `false` | Skip security warning for untrusted directories |
| `TMUX_INTRAY_HOOKS_LOCAL_TRUSTED_PATHS` | `""` | Colon-separated list of trusted directory paths |
| `TMUX_INTRAY_HOOKS_LOCAL_ENABLED_pre_add` | `true` | Enable local hooks for pre-add |
| `TMUX_INTRAY_HOOKS_LOCAL_ENABLED_post_add` | `true` | Enable local hooks for post-add |
| `TMUX_INTRAY_HOOKS_LOCAL_ENABLED_pre_dismiss` | `true` | Enable local hooks for pre-dismiss |
| `TMUX_INTRAY_HOOKS_LOCAL_ENABLED_post_dismiss` | `true` | Enable local hooks for post-dismiss |
| `TMUX_INTRAY_HOOKS_LOCAL_ENABLED_cleanup` | `true` | Enable local hooks for cleanup |
| `TMUX_INTRAY_HOOKS_AUDIT` | `false` | Enable audit mode (preview hooks without executing) |

### Configuration Keys Reference

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `hooks_local_enabled` | bool | `true` | Enable/disable local hooks globally |
| `hooks_local_trusted` | bool | `false` | Skip security warning for untrusted directories |
| `hooks_local_trusted_paths` | list | `[]` | List of trusted directory paths |
| `hooks_local_enabled_pre_add` | bool | `true` | Enable local hooks for pre-add |
| `hooks_local_enabled_post_add` | bool | `true` | Enable local hooks for post-add |
| `hooks_local_enabled_pre_dismiss` | bool | `true` | Enable local hooks for pre-dismiss |
| `hooks_local_enabled_post_dismiss` | bool | `true` | Enable local hooks for post-dismiss |
| `hooks_local_enabled_cleanup` | bool | `true` | Enable local hooks for cleanup |
| `hooks_local_audit` | bool | `false` | Enable audit mode |

---

*Note: This design document is a proposal. Implementation details may change during development. Check the tmux-intray changelog for updates.*
