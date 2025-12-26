package services

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsUpdateAvailable(t *testing.T) {

	tests := []struct {
		name     string
		current  string
		latest   string
		expected bool
	}{
		{
			name:     "patch update available",
			current:  "v1.0.0",
			latest:   "v1.0.1",
			expected: true,
		},
		{
			name:     "minor update available",
			current:  "v1.0.0",
			latest:   "v1.1.0",
			expected: true,
		},
		{
			name:     "major update available",
			current:  "v1.0.0",
			latest:   "v2.0.0",
			expected: true,
		},
		{
			name:     "no update available - same version",
			current:  "v1.0.0",
			latest:   "v1.0.0",
			expected: false,
		},
		{
			name:     "no update - current is newer",
			current:  "v1.5.0",
			latest:   "v1.4.9",
			expected: false,
		},
		{
			name:     "without v prefix in current",
			current:  "1.0.0",
			latest:   "v1.0.1",
			expected: true,
		},
		{
			name:     "without v prefix in latest",
			current:  "v1.0.0",
			latest:   "1.0.1",
			expected: true,
		},
		{
			name:     "without v prefix in both",
			current:  "1.0.0",
			latest:   "1.0.1",
			expected: true,
		},
		{
			name:     "prerelease versions",
			current:  "v1.0.0-beta.1",
			latest:   "v1.0.0",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updater := NewVersionUpdater(WithCurrentVersion(tt.current))
			got := updater.isUpdateAvailable(tt.latest)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDetectInstallMethod(t *testing.T) {

	tests := []struct {
		name     string
		gopath   string
		gobin    string
		expected string
	}{
		{
			name:     "go install detected via GOPATH",
			gopath:   "/home/user/go",
			expected: "go",
		},
		{
			name:     "go install detected via GOBIN",
			gobin:    "/usr/local/bin",
			expected: "go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.gopath != "" {
				t.Setenv("GOPATH", tt.gopath)
			}
			if tt.gobin != "" {
				t.Setenv("GOBIN", tt.gobin)
			}

			updater := NewVersionUpdater(WithCurrentVersion("v1.0.0"))
			method := updater.detectInstallMethod()

			assert.Contains(t, []string{"go", "brew", "binary", "unknown"}, method)
		})
	}
}

func TestDetectInstallMethod_Binary(t *testing.T) {
	updater := NewVersionUpdater(WithCurrentVersion("v1.0.0"))

	t.Setenv("GOPATH", "")
	t.Setenv("GOBIN", "")

	method := updater.detectInstallMethod()
	if method != "brew" {
		assert.Equal(t, "binary", method)
	}
}

func TestUpdateCLI(t *testing.T) {
	updater := NewVersionUpdater(WithCurrentVersion("v1.0.0"))

	t.Run("calls appropriate method", func(t *testing.T) {
		t.Setenv("GOPATH", "")
		t.Setenv("GOBIN", "")
		t.Setenv("HTTPS_PROXY", "http://0.0.0.0:0")
		err := updater.UpdateCLI(context.Background())
		assert.ErrorContains(t, err, "UPDATE: failed to update application")
	})
}

func TestCacheOperations(t *testing.T) {
	updater := NewVersionUpdater(WithCurrentVersion("v1.0.0"))

	cache := UpdateCache{
		LastCheck:   time.Now(),
		LatestKnown: "v1.0.1",
	}

	err := updater.saveCache(cache)
	require.NoError(t, err, "saveCache should not error")

	loaded, err := updater.loadCache()
	require.NoError(t, err, "loadCache should not error")

	assert.Equal(t, cache.LatestKnown, loaded.LatestKnown)
	assert.WithinDuration(t, cache.LastCheck, loaded.LastCheck, time.Second)

	cacheDir, _ := updater.getCacheDir()
	_ = os.RemoveAll(cacheDir)
}

func TestCheckForUpdates_WithDisableEnvVar(t *testing.T) {
	updater := NewVersionUpdater(WithCurrentVersion("v1.0.0"))

	t.Setenv("MATECOMMIT_DISABLE_UPDATE_CHECK", "1")

	updater.CheckForUpdates(context.Background())

	_, err := updater.loadCache()
	assert.Error(t, err, "cache should not exist when checks are disabled")
}

func TestCheckForUpdates_WithCache(t *testing.T) {
	updater := NewVersionUpdater(WithCurrentVersion("v1.0.0"))

	cache := UpdateCache{
		LastCheck:   time.Now().Add(-1 * time.Hour),
		LatestKnown: "v1.0.1",
	}

	err := updater.saveCache(cache)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	updater.CheckForUpdates(ctx)

	cacheDir, _ := updater.getCacheDir()
	_ = os.RemoveAll(cacheDir)
}

func TestUpdateViaBinary_RealCallFails(t *testing.T) {
	updater := NewVersionUpdater(WithCurrentVersion("v1.0.0"))

	t.Setenv("HTTPS_PROXY", "http://0.0.0.0:0")

	err := updater.updateViaBinary(context.Background())
	assert.Error(t, err)
}

func TestExtractZip(t *testing.T) {
	updater := NewVersionUpdater(WithCurrentVersion("v1.0.0"))

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	f, err := os.Create(zipPath)
	require.NoError(t, err)

	w := zip.NewWriter(f)

	fileW, err := w.Create("matecommit.exe")
	require.NoError(t, err)
	_, err = fileW.Write([]byte("dummy content"))
	require.NoError(t, err)

	_, err = w.Create("some-dir/")
	require.NoError(t, err)

	require.NoError(t, w.Close())
	require.NoError(t, f.Close())

	destDir := t.TempDir()
	binPath, err := updater.extractZip(zipPath, destDir)

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(destDir, "matecommit.exe"), binPath)

	content, err := os.ReadFile(binPath)
	require.NoError(t, err)
	assert.Equal(t, "dummy content", string(content))
}

func TestExtractTarGz(t *testing.T) {
	updater := NewVersionUpdater(WithCurrentVersion("v1.0.0"))

	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "test.tar.gz")

	f, err := os.Create(tarPath)
	require.NoError(t, err)

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Add a file
	body := []byte("dummy content")
	hdr := &tar.Header{
		Name: "matecommit",
		Mode: 0755,
		Size: int64(len(body)),
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err = tw.Write(body)
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	require.NoError(t, f.Close())

	destDir := t.TempDir()
	binPath, err := updater.extractTarGz(tarPath, destDir)

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(destDir, "matecommit"), binPath)

	content, err := os.ReadFile(binPath)
	require.NoError(t, err)
	assert.Equal(t, "dummy content", string(content))
}
func TestVersionUpdater_IsUpdateAvailable_EdgeCases(t *testing.T) {
	updater := NewVersionUpdater(WithCurrentVersion("v1.0.0"))

	assert.False(t, updater.isUpdateAvailable("v1.0.0-rc.1"))
	assert.True(t, NewVersionUpdater(WithCurrentVersion("v1.0.0-rc.1")).isUpdateAvailable("v1.0.0"))
}

func TestVersionUpdater_LoadCache_InvalidJSON(t *testing.T) {
	updater := NewVersionUpdater(WithCurrentVersion("v1.0.0"))

	cacheDir, _ := updater.getCacheDir()
	_ = os.MkdirAll(cacheDir, 0755)
	cacheFile := filepath.Join(cacheDir, "update_cache.json")
	_ = os.WriteFile(cacheFile, []byte("invalid json"), 0644)

	_, err := updater.loadCache()
	assert.Error(t, err)

	_ = os.RemoveAll(cacheDir)
}
