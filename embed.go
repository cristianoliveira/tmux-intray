// Package assets provides embedded files for tmux-intray.
package assets

import "embed"

//go:embed bin
//go:embed commands
//go:embed lib
//go:embed scripts
//go:embed tmux-intray.tmux
//go:embed go.mod
var FS embed.FS
