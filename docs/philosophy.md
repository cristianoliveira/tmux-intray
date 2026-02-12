# Project Philosophy

## Core Principle: One and Only One Way

tmux-intray follows the principle that there should be one—and preferably only one—obvious way to do things. This philosophy, inspired by Python's design principles, guides all design decisions in the project.

### Design Principles

1. **Single Configuration Format**
   - Configuration files use TOML format only
   - TOML is chosen for its readability, support for comments, and clean syntax
   - No JSON, YAML, or other configuration formats are supported

2. **Simplicity Over Complexity**
   - Prefer simple solutions over complex ones
   - Avoid configuration options that are rarely needed
   - Provide sensible defaults that work for most use cases

3. **Explicit is Better Than Implicit**
   - Configuration should be clear and obvious
   - Avoid magic values or implicit behaviors
   - Documentation should explain what each option does

4. **Environment Variables for Overrides**
   - Environment variables provide a convenient way to override configuration
   - Useful for debugging, testing, and per-session customization
   - Environment variables always take precedence over file configuration

5. **Stability Predictability**
   - Don't break existing behavior without good reason
   - Configuration file format changes should be backward compatible when possible
   - Default values should remain stable between versions

6. **Minimal Dependencies**
   - Prefer standard library and well-maintained dependencies
   - Avoid introducing dependencies for features that can be implemented simply
   - Choose dependencies with active maintenance and good security practices

### Configuration Hierarchy

Configuration is loaded in this order (later values override earlier ones):

1. **Default Values** - Built-in sensible defaults
2. **Configuration File** - TOML file at `~/.config/tmux-intray/config.toml`
3. **Environment Variables** - `TMUX_INTRAY_*` variables override file values

This hierarchy ensures:
- No configuration needed for most users (defaults work out of the box)
- Easy customization through a single configuration file
- Quick overrides for debugging or special cases via environment variables

### Why TOML?

TOML was chosen as the sole configuration format because:

- **Human-readable**: Clean, simple syntax that's easy to read and write
- **Supports comments**: Essential for explaining configuration options in-place
- **Explicit types**: Clear distinction between strings, numbers, and booleans
- **Widely supported**: Many tools and editors have TOML support
- **No ambiguity**: Unlike YAML, TOML doesn't have ambiguous syntax constructs
- **Minimal punctuation**: Cleaner than JSON, less error-prone than YAML

### Examples

Good configuration:

```toml
# Clear, commented, uses TOML format
max_notifications = 1000
storage_backend = "sqlite"

# Status bar settings
status_enabled = true
status_format = "compact"
```

Bad configuration (what we avoid):

- Multiple config files in different formats
- Complex nested structures for simple settings
- Magic numbers without comments explaining their purpose
- Configuration scattered across multiple locations

### Contributing

When adding new features or configuration options:

1. Prefer existing patterns over introducing new ones
2. Use TOML for any configuration (no new config formats)
3. Provide sensible defaults
4. Document the option clearly
5. Consider whether an environment variable override makes sense
6. Keep it simple—if you're adding complexity, question whether it's necessary

### Migration Policy

When configuration formats must change:

1. Maintain backward compatibility when possible
2. Provide clear migration instructions in the changelog
3. Add deprecation warnings before removing old options
4. Automate migration where feasible