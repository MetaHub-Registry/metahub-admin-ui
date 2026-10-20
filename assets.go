package main

import (
	"embed"
	"io/fs"
)

// The SPA build output (vite build → ./dist) is embedded into the binary.
//
// A minimal placeholder dist/index.html is committed so the module builds
// on a fresh clone without an SPA build; `npm run build` overwrites the
// directory (see README). If dist/ is ever removed entirely, test.sh
// recreates the placeholder before running the Go tests.
//
//go:embed all:dist
var distFS embed.FS

// SPA returns the embedded UI assets rooted at dist/.
func SPA() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// Unreachable for a literal embed pattern.
		panic(err)
	}
	return sub
}
