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
