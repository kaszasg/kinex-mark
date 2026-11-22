// Package web embeds the static SvelteKit build into the Go binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:build
var build embed.FS

// Files returns the static UI rooted at the build directory.
func Files() (fs.FS, error) {
	return fs.Sub(build, "build")
}
