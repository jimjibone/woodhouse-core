package webui

import "embed"

// The all: prefix matters. A plain `build/*` walks subdirectories with Go's
// default rule, which skips any file or directory whose name starts with "_"
// or "." - and Vite names some shared chunks things like "_waGfCGV.js". Such
// a chunk is then missing from the binary at runtime, the browser 404s on the
// module import, and the SPA renders SvelteKit's client-side "500 Internal
// Error" page instead of the app. Whether it bites depends on the chunk names
// a given build happens to produce, so it fails intermittently across builds.
//
//go:embed all:build
var Content embed.FS
