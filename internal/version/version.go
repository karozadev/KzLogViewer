// Package version holds build-time metadata injected via -ldflags.
package version

// Version is the semantic version of the running binary. It is overridden
// at build time by GoReleaser (-X github.com/karozadev/KzLogViewer/internal/version.Version=...).
// The default "dev" value identifies a binary built without release
// metadata, e.g. via `go build` or `go run`.
var Version = "dev"

// Commit is the git commit SHA the binary was built from.
var Commit = "none"

// Date is the build timestamp, in RFC3339 format.
var Date = "unknown"
