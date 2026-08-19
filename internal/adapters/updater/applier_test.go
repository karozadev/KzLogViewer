package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/karozadev/KzLogViewer/internal/core/ports"
)

func buildTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("write tar content: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}

func buildZip(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	w, err := zw.Create(name)
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatalf("write zip content: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}

func TestExtractFromTarGz(t *testing.T) {
	content := []byte("#!/bin/sh\necho fake-binary\n")
	data := buildTarGz(t, binaryName, content)

	archive := filepath.Join(t.TempDir(), "release.tar.gz")
	if err := os.WriteFile(archive, data, 0o644); err != nil {
		t.Fatal(err)
	}

	path, err := extractFromTarGz(archive)
	if err != nil {
		t.Fatalf("extractFromTarGz: %v", err)
	}
	defer func() { _ = os.Remove(path) }()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("extracted content mismatch")
	}
}

func TestExtractFromTarGzMissingBinary(t *testing.T) {
	data := buildTarGz(t, "README.md", []byte("hello"))
	archive := filepath.Join(t.TempDir(), "release.tar.gz")
	if err := os.WriteFile(archive, data, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := extractFromTarGz(archive); err == nil {
		t.Fatal("expected an error when the binary is absent from the archive")
	}
}

func TestExtractFromZip(t *testing.T) {
	content := []byte("fake windows binary")
	data := buildZip(t, binaryName+".exe", content)

	archive := filepath.Join(t.TempDir(), "release.zip")
	if err := os.WriteFile(archive, data, 0o644); err != nil {
		t.Fatal(err)
	}

	path, err := extractFromZip(archive)
	if err != nil {
		t.Fatalf("extractFromZip: %v", err)
	}
	defer func() { _ = os.Remove(path) }()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("extracted content mismatch")
	}
}

func TestReplaceBinary(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "kzlogviewer")
	if err := os.WriteFile(target, []byte("old version"), 0o755); err != nil {
		t.Fatal(err)
	}
	newBinary := filepath.Join(dir, "new")
	if err := os.WriteFile(newBinary, []byte("new version"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := replaceBinary(target, newBinary); err != nil {
		t.Fatalf("replaceBinary: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new version" {
		t.Errorf("target content = %q, want %q", got, "new version")
	}
	if _, err := os.Stat(target + ".old"); !os.IsNotExist(err) {
		t.Errorf("expected backup file to be removed")
	}
}

func TestNewBinaryApplier(t *testing.T) {
	if NewBinaryApplier() == nil {
		t.Fatal("expected a non-nil applier")
	}
}

func TestExtractBinaryDispatchesByExtension(t *testing.T) {
	dir := t.TempDir()

	tarPath := filepath.Join(dir, "release.tar.gz")
	if err := os.WriteFile(tarPath, buildTarGz(t, binaryName, []byte("unix")), 0o644); err != nil {
		t.Fatal(err)
	}
	path, err := extractBinary(tarPath, "kzlogviewer_linux_amd64.tar.gz")
	if err != nil {
		t.Fatalf("extractBinary (tar.gz): %v", err)
	}
	_ = os.Remove(path)

	zipPath := filepath.Join(dir, "release.zip")
	if err := os.WriteFile(zipPath, buildZip(t, binaryName+".exe", []byte("win")), 0o644); err != nil {
		t.Fatal(err)
	}
	path, err = extractBinary(zipPath, "kzlogviewer_windows_amd64.zip")
	if err != nil {
		t.Fatalf("extractBinary (zip): %v", err)
	}
	_ = os.Remove(path)
}

func TestDownloadNon200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	a := &BinaryApplier{httpClient: server.Client()}
	if _, err := a.download(context.Background(), server.URL); err == nil {
		t.Fatal("expected an error for non-200 status")
	}
}

func TestReplaceBinaryRenameFailure(t *testing.T) {
	dir := t.TempDir()
	newBinary := filepath.Join(dir, "new")
	if err := os.WriteFile(newBinary, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	missingTarget := filepath.Join(dir, "does", "not", "exist", "kzlogviewer")
	if err := replaceBinary(missingTarget, newBinary); err == nil {
		t.Fatal("expected an error when the target directory does not exist")
	}
}

func TestCopyFileMissingSource(t *testing.T) {
	dir := t.TempDir()
	if err := copyFile(filepath.Join(dir, "missing"), filepath.Join(dir, "out")); err == nil {
		t.Fatal("expected an error for a missing source file")
	}
}

func TestApplyDownloadsExtractsAndInstalls(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a unix-style archive fixture")
	}
	content := []byte("new kzlogviewer binary")
	data := buildTarGz(t, binaryName, content)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(data); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	target := filepath.Join(dir, "kzlogviewer")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	applier := &BinaryApplier{
		httpClient: server.Client(),
		executable: func() (string, error) { return target, nil },
	}

	err := applier.Apply(context.Background(), ports.Release{
		Version:     "v1.0.0",
		AssetName:   "kzlogviewer_linux_amd64.tar.gz",
		DownloadURL: server.URL,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Errorf("installed content = %q, want %q", got, content)
	}
}
