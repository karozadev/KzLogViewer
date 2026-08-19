package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/karozadev/KzLogViewer/internal/core/ports"
)

// binaryName is the executable file name inside the release archive, as
// produced by GoReleaser.
const binaryName = "kzlogviewer"

// BinaryApplier implements ports.Applier: it downloads a release archive,
// extracts the kzlogviewer binary, and atomically replaces the currently
// running executable.
type BinaryApplier struct {
	httpClient *http.Client
	executable func() (string, error)
}

// NewBinaryApplier builds a ready-to-use BinaryApplier.
func NewBinaryApplier() *BinaryApplier {
	return &BinaryApplier{
		httpClient: &http.Client{Timeout: 2 * time.Minute},
		executable: os.Executable,
	}
}

// Apply implements ports.Applier.
func (a *BinaryApplier) Apply(ctx context.Context, release ports.Release) error {
	archive, err := a.download(ctx, release.DownloadURL)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(archive) }()

	binPath, err := extractBinary(archive, release.AssetName)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(binPath) }()

	// The downloaded binary must be executable; 0755 is required here.
	if err := os.Chmod(binPath, 0o755); err != nil { //nolint:gosec // executable binary requires +x
		return fmt.Errorf("make new binary executable: %w", err)
	}

	target, err := a.executable()
	if err != nil {
		return fmt.Errorf("locate running executable: %w", err)
	}
	return replaceBinary(target, binPath)
}

func (a *BinaryApplier) download(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download release: unexpected status %s", resp.Status)
	}

	tmp, err := os.CreateTemp("", "kzlogviewer-update-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = tmp.Close() }()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("save downloaded release: %w", err)
	}
	return tmp.Name(), nil
}

// extractBinary pulls the kzlogviewer executable out of a tar.gz or zip
// archive (chosen from the asset's extension) into a fresh temp file.
func extractBinary(archivePath, assetName string) (string, error) {
	if filepath.Ext(assetName) == ".zip" {
		return extractFromZip(archivePath)
	}
	return extractFromTarGz(archivePath)
}

// extractFromTarGz reads archivePath, a file this process itself just
// downloaded to a temp directory, not attacker-controlled input.
func extractFromTarGz(archivePath string) (string, error) {
	f, err := os.Open(archivePath) //nolint:gosec // archivePath is our own downloaded temp file
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("open gzip archive: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return "", fmt.Errorf("binary %q not found in archive", binaryName)
		}
		if err != nil {
			return "", fmt.Errorf("read tar archive: %w", err)
		}
		if filepath.Base(hdr.Name) != binaryName {
			continue
		}
		return writeTempBinary(tr)
	}
}

func extractFromZip(archivePath string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("open zip archive: %w", err)
	}
	defer func() { _ = r.Close() }()

	want := binaryName + ".exe"
	for _, file := range r.File {
		if filepath.Base(file.Name) != want {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return "", err
		}
		defer func() { _ = rc.Close() }()
		return writeTempBinary(rc)
	}
	return "", fmt.Errorf("binary %q not found in archive", want)
}

func writeTempBinary(r io.Reader) (string, error) {
	out, err := os.CreateTemp("", "kzlogviewer-bin-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, r); err != nil {
		_ = os.Remove(out.Name())
		return "", err
	}
	return out.Name(), nil
}

// replaceBinary swaps target for newBinary. It moves the running binary
// aside first so the replacement works even on platforms (Windows) that
// refuse to overwrite an executable file that is currently mapped into
// memory.
func replaceBinary(target, newBinary string) error {
	backup := target + ".old"
	_ = os.Remove(backup)

	if err := os.Rename(target, backup); err != nil {
		return fmt.Errorf("back up current binary: %w", err)
	}
	if err := copyFile(newBinary, target); err != nil {
		_ = os.Rename(backup, target)
		return fmt.Errorf("install new binary: %w", err)
	}
	if err := os.Chmod(target, 0o755); err != nil && runtime.GOOS != "windows" { //nolint:gosec // executable binary requires +x
		return fmt.Errorf("make new binary executable: %w", err)
	}
	_ = os.Remove(backup)
	return nil
}

// copyFile installs an executable: src and dst are our own temp path and
// the running binary's own path (from os.Executable), not user input.
func copyFile(src, dst string) error {
	in, err := os.Open(src) //nolint:gosec // src is our own downloaded temp file
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755) //nolint:gosec // installs an executable binary
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, in)
	return err
}
