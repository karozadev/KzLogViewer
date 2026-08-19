package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/karozadev/KzLogViewer/internal/core/ports"
)

const defaultReleasesURL = "https://api.github.com/repos/karozadev/KzLogViewer/releases/latest"

// GitHubChecker implements ports.UpdateChecker against the GitHub Releases
// API, with no dependency beyond the standard library.
type GitHubChecker struct {
	releasesURL string
	httpClient  *http.Client
}

// NewGitHubChecker builds a checker pointed at the KzLogViewer GitHub
// repository's latest release.
func NewGitHubChecker() *GitHubChecker {
	return &GitHubChecker{
		releasesURL: defaultReleasesURL,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	PublishedAt string        `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

// LatestRelease implements ports.UpdateChecker.
func (c *GitHubChecker) LatestRelease(ctx context.Context) (ports.Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.releasesURL, nil)
	if err != nil {
		return ports.Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ports.Release{}, fmt.Errorf("query GitHub releases: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return ports.Release{}, fmt.Errorf("GitHub releases API returned %s", resp.Status)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return ports.Release{}, fmt.Errorf("decode GitHub release: %w", err)
	}

	assetName := ExpectedAssetName(runtime.GOOS, runtime.GOARCH)
	for _, a := range release.Assets {
		if a.Name == assetName {
			return ports.Release{
				Version:     release.TagName,
				AssetName:   a.Name,
				DownloadURL: a.BrowserDownloadURL,
				PublishedAt: release.PublishedAt,
			}, nil
		}
	}
	return ports.Release{}, fmt.Errorf("no release asset named %q found for %s/%s", assetName, runtime.GOOS, runtime.GOARCH)
}

// ExpectedAssetName returns the archive name GoReleaser publishes for a
// given OS/architecture pair, as configured in .goreleaser.yaml.
func ExpectedAssetName(goos, goarch string) string {
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("kzlogviewer_%s_%s.%s", goos, goarch, ext)
}
