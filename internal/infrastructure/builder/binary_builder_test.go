package builder

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBinaryBuilder_GetBuildTargets(t *testing.T) {
	trans, _ := i18n.NewTranslations("en", "../../../i18n/locales")
	builder := NewBinaryBuilder("main.go", "app", "v1.0.0", "sha", "date", "build", trans)
	targets := builder.GetBuildTargets()

	assert.Len(t, targets, 6)
	assert.Contains(t, targets, BuildTarget{"linux", "amd64"})
	assert.Contains(t, targets, BuildTarget{"windows", "amd64"})
	assert.Contains(t, targets, BuildTarget{"darwin", "arm64"})
}

func TestBinaryBuilder_PackageBinary_Zip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "binary-builder-test")
	require.NoError(t, err)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("error al eliminar el directorio temporal: %s", err)
		}
	}()

	trans, _ := i18n.NewTranslations("en", "../../../i18n/locales")
	builder := NewBinaryBuilder("main.go", "app", "v1.0.0", "sha", "date", tempDir, trans)

	binaryName := "app.exe"
	binaryPath := filepath.Join(tempDir, binaryName)
	err = os.WriteFile(binaryPath, []byte("dummy content"), 0755)
	require.NoError(t, err)

	target := BuildTarget{"windows", "amd64"}
	archivePath, err := builder.PackageBinary(binaryPath, target)
	require.NoError(t, err)

	_, err = os.Stat(archivePath)
	assert.NoError(t, err)
	assert.Contains(t, archivePath, ".zip")

	r, err := zip.OpenReader(archivePath)
	require.NoError(t, err)
	defer func() {
		if err := r.Close(); err != nil {
			t.Logf("error cerrando zip: %s", err)
		}
	}()

	found := false
	for _, f := range r.File {
		if f.Name == binaryName {
			found = true
			rc, err := f.Open()
			require.NoError(t, err)
			content, err := io.ReadAll(rc)
			require.NoError(t, err)
			assert.Equal(t, []byte("dummy content"), content)
			_ = rc.Close()
		}
	}
	assert.True(t, found, "binary not found in zip")
}

func TestBinaryBuilder_PackageBinary_TarGz(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "binary-builder-test")
	require.NoError(t, err)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("error al eliminar el directorio temporal: %s", err)
		}
	}()

	trans, _ := i18n.NewTranslations("en", "../../../i18n/locales")
	builder := NewBinaryBuilder("main.go", "app", "v1.0.0", "sha", "date", tempDir, trans)

	binaryName := "app"
	binaryPath := filepath.Join(tempDir, binaryName)
	err = os.WriteFile(binaryPath, []byte("dummy content"), 0755)
	require.NoError(t, err)

	target := BuildTarget{"linux", "amd64"}
	archivePath, err := builder.PackageBinary(binaryPath, target)
	require.NoError(t, err)

	_, err = os.Stat(archivePath)
	assert.NoError(t, err)
	assert.Contains(t, archivePath, ".tar.gz")

	f, err := os.Open(archivePath)
	require.NoError(t, err)
	defer func() {
		if err := f.Close(); err != nil {
			t.Logf("error cerrando tar: %s", err)
		}
	}()

	gzr, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer func() {
		if err := gzr.Close(); err != nil {
			t.Logf("error cerrando gzip: %s", err)
		}
	}()

	tr := tar.NewReader(gzr)
	found := false
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		if header.Name == binaryName {
			found = true
			content, err := io.ReadAll(tr)
			require.NoError(t, err)
			assert.Equal(t, []byte("dummy content"), content)
		}
	}
	assert.True(t, found, "binario no encontrado en tar.gz")
}
