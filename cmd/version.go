package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// Version is the version of tmux-intray.
// This can be overridden at build time using ldflags.
var Version = "0.1.0"

// versionOutputWriter is the writer used by PrintVersion. Can be changed for testing.
var versionOutputWriter io.Writer = os.Stdout

// GetVersion returns the version string.
// It attempts to read version from go.mod, falling back to the default Version.
func GetVersion() string {
	// Try to get version from go.mod using go list
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Version}}")
	output, err := cmd.Output()
	if err == nil {
		version := strings.TrimSpace(string(output))
		if version != "" && version != "none" {
			return version
		}
	}
	return Version
}

// PrintVersion prints the version information to stdout.
func PrintVersion() {
	fmt.Fprintf(versionOutputWriter, "tmux-intray v%s\n", GetVersion())
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Show the current version of tmux-intray.`,
	Run: func(cmd *cobra.Command, args []string) {
		PrintVersion()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
