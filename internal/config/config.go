// Package config resolves KzLogViewer's runtime settings from environment
// variables and command-line flags.
package config

import "os"

// Config holds the settings needed to start the TUI. Docker connection
// settings (DOCKER_HOST and friends) are read directly by the Docker
// adapter from the standard Docker environment variables.
type Config struct {
	// DisableUpdateCheck skips the startup GitHub release check.
	DisableUpdateCheck bool
}

// FromEnvironment reads KZLOGVIEWER_* environment variables into a Config.
func FromEnvironment() Config {
	return Config{
		DisableUpdateCheck: os.Getenv("KZLOGVIEWER_NO_UPDATE_CHECK") != "",
	}
}
