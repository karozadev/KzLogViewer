package ports

import "context"

// Release describes a published KzLogViewer release, as found on GitHub.
type Release struct {
	Version     string
	AssetName   string
	DownloadURL string
	PublishedAt string
}

// UpdateChecker looks up the latest available release.
type UpdateChecker interface {
	LatestRelease(ctx context.Context) (Release, error)
}

// Applier downloads and installs a release in place of the running binary.
type Applier interface {
	Apply(ctx context.Context, release Release) error
}
