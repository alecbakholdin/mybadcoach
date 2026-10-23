//go:build tools

// Package deps pins runtime dependencies that later backend tickets consume.
//
// Ticket 01 installs the approved dependency set ahead of its first consumer:
// modernc.org/sqlite is the driver wired in ticket 02, and bcrypt is the
// password hasher wired in ticket 03. Without an import, `go mod tidy` would
// drop both from go.mod before those tickets land. This file is never built
// (the `tools` tag is not set in normal builds) and exists only to anchor the
// versions in the module graph.
package deps

import (
	_ "golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)
