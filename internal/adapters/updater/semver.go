// Package updater implements the ports.UpdateChecker and ports.Applier
// ports: it looks up the latest KzLogViewer release on GitHub and can
// replace the currently running binary with it.
package updater

import (
	"strconv"
	"strings"
)

// semver is a minimal, dependency-free parse of a "vMAJOR.MINOR.PATCH[-PRERELEASE]"
// version string, sufficient to order KzLogViewer releases.
type semver struct {
	major, minor, patch int
	prerelease          string
}

func parseSemver(raw string) semver {
	s := strings.TrimPrefix(strings.TrimSpace(raw), "v")
	core, prerelease, _ := strings.Cut(s, "-")
	parts := strings.SplitN(core, ".", 3)

	var v semver
	if len(parts) > 0 {
		v.major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) > 1 {
		v.minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) > 2 {
		v.patch, _ = strconv.Atoi(parts[2])
	}
	v.prerelease = prerelease
	return v
}

// IsNewer reports whether latest denotes a version strictly greater than
// current, following semantic versioning precedence: numeric core first,
// then a release (no prerelease suffix) outranks any prerelease, and two
// prereleases are compared lexicographically.
func IsNewer(current, latest string) bool {
	c, l := parseSemver(current), parseSemver(latest)

	if c.major != l.major {
		return l.major > c.major
	}
	if c.minor != l.minor {
		return l.minor > c.minor
	}
	if c.patch != l.patch {
		return l.patch > c.patch
	}
	if c.prerelease == l.prerelease {
		return false
	}
	if c.prerelease == "" {
		return false // current is a full release, latest cannot outrank it at the same core version
	}
	if l.prerelease == "" {
		return true // latest is a full release of the same core version current is a prerelease of
	}
	return l.prerelease > c.prerelease
}
