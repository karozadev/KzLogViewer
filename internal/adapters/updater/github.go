package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"time"

	"github.com/karozadev/KzLogViewer/internal/core/ports"
)

const (
	repoOwner = "karozadev"
	repoName  = "KzLogViewer"

	defaultReleasesURL     = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"
	defaultReleasesListURL = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases"
)

// GitHubChecker implements ports.UpdateChecker against the GitHub Releases
// API, with no dependency beyond the standard library.
type GitHubChecker struct {
	releasesURL     string
	releasesListURL string
	httpClient      *http.Client
}

// NewGitHubChecker builds a checker pointed at the KzLogViewer GitHub
// repository's releases.
func NewGitHubChecker() *GitHubChecker {
	return &GitHubChecker{
		releasesURL:     defaultReleasesURL,
		releasesListURL: defaultReleasesListURL,
		httpClient:      &http.Client{Timeout: 10 * time.Second},
	}
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	PublishedAt string        `json:"published_at"`
	Body        string        `json:"body"`
	Draft       bool          `json:"draft"`
	Assets      []githubAsset `json:"assets"`
}

// LatestRelease implements ports.UpdateChecker.
func (c *GitHubChecker) LatestRelease(ctx context.Context) (ports.Release, error) {
	var release githubRelease
	if err := c.getJSON(ctx, c.releasesURL, &release); err != nil {
		return ports.Release{}, err
	}
	rel, ok := toRelease(release)
	if !ok {
		return ports.Release{}, fmt.Errorf("no release asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return rel, nil
}

// ReleasesSince implements ports.UpdateChecker. It lists every published,
// non-draft release strictly newer than currentVersion and returns them
// oldest first, so their Notes can be concatenated into one cumulative
// changelog when several versions are skipped by a single update.
func (c *GitHubChecker) ReleasesSince(ctx context.Context, currentVersion string) ([]ports.Release, error) {
	var all []githubRelease
	if err := c.getJSON(ctx, c.releasesListURL+"?per_page=100", &all); err != nil {
		return nil, err
	}

	var newer []ports.Release
	for _, r := range all {
		if r.Draft || !IsNewer(currentVersion, r.TagName) {
			continue
		}
		rel, _ := toRelease(r) // a missing matching asset still keeps the entry for changelog purposes
		newer = append(newer, rel)
	}

	sort.Slice(newer, func(i, j int) bool { return IsNewer(newer[i].Version, newer[j].Version) })
	return newer, nil
}

func (c *GitHubChecker) getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("query GitHub releases: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub releases API returned %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode GitHub release: %w", err)
	}
	return nil
}

// toRelease converts a githubRelease into a ports.Release, filling in the
// download asset matching the current platform when present.
func toRelease(r githubRelease) (ports.Release, bool) {
	rel := ports.Release{
		Version:     r.TagName,
		PublishedAt: r.PublishedAt,
		Notes:       r.Body,
	}
	assetName := ExpectedAssetName(runtime.GOOS, runtime.GOARCH)
	for _, a := range r.Assets {
		if a.Name == assetName {
			rel.AssetName = a.Name
			rel.DownloadURL = a.BrowserDownloadURL
			return rel, true
		}
	}
	return rel, false
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
