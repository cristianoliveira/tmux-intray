// Package version provides version information for tmux-intray.
package version

// Version is the version of tmux-intray. This can be overridden at build time using ldflags.
var Version = "development"

// Commit is the git commit hash. This can be overridden at build time using ldflags.
var Commit = "unknown"

// String returns the full version string including the commit hash if available.
func String() string {
	if Commit != "unknown" {
		return Version + "+" + Commit
	}
	return Version
}
