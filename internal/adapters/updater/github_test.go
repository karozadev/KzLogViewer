package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

func newTestChecker(t *testing.T, release githubRelease) (*GitHubChecker, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(release)
	}))
	t.Cleanup(server.Close)

	return &GitHubChecker{releasesURL: server.URL, httpClient: server.Client()}, server
}

func TestLatestReleaseFindsMatchingAsset(t *testing.T) {
	assetName := ExpectedAssetName(runtime.GOOS, runtime.GOARCH)
	checker, _ := newTestChecker(t, githubRelease{
		TagName:     "v1.2.3",
		PublishedAt: "2024-01-02T00:00:00Z",
		Assets: []githubAsset{
			{Name: "kzlogviewer_other_arch.tar.gz", BrowserDownloadURL: "https://example.invalid/other"},
			{Name: assetName, BrowserDownloadURL: "https://example.invalid/" + assetName},
		},
	})

	release, err := checker.LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	if release.Version != "v1.2.3" {
		t.Errorf("Version = %q", release.Version)
	}
	if release.AssetName != assetName {
		t.Errorf("AssetName = %q, want %q", release.AssetName, assetName)
	}
}

func TestLatestReleaseNoMatchingAsset(t *testing.T) {
	checker, _ := newTestChecker(t, githubRelease{
		TagName: "v1.2.3",
		Assets: []githubAsset{
			{Name: "kzlogviewer_does_not_match.tar.gz"},
		},
	})

	_, err := checker.LatestRelease(context.Background())
	if err == nil {
		t.Fatal("expected an error when no asset matches the current platform")
	}
}

func TestLatestReleaseNon200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := &GitHubChecker{releasesURL: server.URL, httpClient: server.Client()}
	_, err := checker.LatestRelease(context.Background())
	if err == nil {
		t.Fatal("expected an error on non-200 status")
	}
}

func newListTestChecker(t *testing.T, releases []githubRelease) *GitHubChecker {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(releases)
	}))
	t.Cleanup(server.Close)

	return &GitHubChecker{releasesListURL: server.URL, httpClient: server.Client()}
}

func TestReleasesSinceFiltersSortsAndKeepsNotes(t *testing.T) {
	assetName := ExpectedAssetName(runtime.GOOS, runtime.GOARCH)
	asset := githubAsset{Name: assetName, BrowserDownloadURL: "https://example.invalid/" + assetName}

	checker := newListTestChecker(t, []githubRelease{
		{TagName: "v1.2.0", Body: "second", Assets: []githubAsset{asset}},
		{TagName: "v1.0.0", Body: "too old", Assets: []githubAsset{asset}},
		{TagName: "v1.3.0", Body: "third", Draft: true, Assets: []githubAsset{asset}},
		{TagName: "v1.1.0", Body: "first", Assets: []githubAsset{asset}},
	})

	releases, err := checker.ReleasesSince(context.Background(), "v1.0.0")
	if err != nil {
		t.Fatalf("ReleasesSince: %v", err)
	}

	if len(releases) != 2 {
		t.Fatalf("got %d releases, want 2 (draft and v1.0.0 itself excluded): %+v", len(releases), releases)
	}
	if releases[0].Version != "v1.1.0" || releases[0].Notes != "first" {
		t.Errorf("releases[0] = %+v, want v1.1.0/first", releases[0])
	}
	if releases[1].Version != "v1.2.0" || releases[1].Notes != "second" {
		t.Errorf("releases[1] = %+v, want v1.2.0/second", releases[1])
	}
}

func TestReleasesSinceNoneNewer(t *testing.T) {
	checker := newListTestChecker(t, []githubRelease{
		{TagName: "v1.0.0", Body: "initial"},
	})

	releases, err := checker.ReleasesSince(context.Background(), "v1.0.0")
	if err != nil {
		t.Fatalf("ReleasesSince: %v", err)
	}
	if len(releases) != 0 {
		t.Fatalf("got %d releases, want 0", len(releases))
	}
}

func TestReleasesSinceKeepsEntryWithoutMatchingAsset(t *testing.T) {
	checker := newListTestChecker(t, []githubRelease{
		{TagName: "v1.1.0", Body: "no asset for this platform", Assets: []githubAsset{{Name: "kzlogviewer_other.tar.gz"}}},
	})

	releases, err := checker.ReleasesSince(context.Background(), "v1.0.0")
	if err != nil {
		t.Fatalf("ReleasesSince: %v", err)
	}
	if len(releases) != 1 {
		t.Fatalf("got %d releases, want 1", len(releases))
	}
	if releases[0].DownloadURL != "" {
		t.Errorf("expected empty DownloadURL when no asset matches, got %q", releases[0].DownloadURL)
	}
	if releases[0].Notes != "no asset for this platform" {
		t.Errorf("Notes = %q", releases[0].Notes)
	}
}

func TestReleasesSinceNon200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	checker := &GitHubChecker{releasesListURL: server.URL, httpClient: server.Client()}
	if _, err := checker.ReleasesSince(context.Background(), "v1.0.0"); err == nil {
		t.Fatal("expected an error on non-200 status")
	}
}

func TestNewGitHubChecker(t *testing.T) {
	if NewGitHubChecker() == nil {
		t.Fatal("expected a non-nil checker")
	}
}

func TestExpectedAssetName(t *testing.T) {
	if got := ExpectedAssetName("windows", "amd64"); got != "kzlogviewer_windows_amd64.zip" {
		t.Errorf("got %q", got)
	}
	if got := ExpectedAssetName("linux", "amd64"); got != "kzlogviewer_linux_amd64.tar.gz" {
		t.Errorf("got %q", got)
	}
}
