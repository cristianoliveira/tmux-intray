// Package hooks provides a hook subsystem for extensibility.
package hooks

// Init initializes the hooks subsystem.
func Init() {
}

// Run executes hooks for a hook point with environment variables.
func Run(hookPoint string, envVars ...string) {
	_ = hookPoint
	_ = envVars
}
