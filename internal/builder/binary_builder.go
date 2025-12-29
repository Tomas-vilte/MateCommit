package builder

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
	"golang.org/x/sync/errgroup"
)

type BuildTarget struct {
	GOOS   string
	GOARCH string
}

type BinaryBuilder struct {
	mainPath   string
	binaryName string
	version    string
	commit     string
	date       string
	buildDir   string
}

type Option func(*BinaryBuilder)

func WithVersion(version string) Option {
	return func(b *BinaryBuilder) {
		b.version = version
	}
}

func WithCommit(commit string) Option {
	return func(b *BinaryBuilder) {
		b.commit = commit
	}
}

func WithBuildDir(dir string) Option {
	return func(b *BinaryBuilder) {
		b.buildDir = dir
	}
}

func WithDate(date string) Option {
	return func(b *BinaryBuilder) {
		b.date = date
	}
}

func NewBinaryBuilder(mainPath, binaryName string, opts ...Option) *BinaryBuilder {
	b := &BinaryBuilder{
		mainPath:   mainPath,
		binaryName: binaryName,
		buildDir:   "./dist",
		date:       time.Now().Format(time.RFC3339),
	}

	for _, opt := range opts {
		opt(b)
	}
	return b
}

func (b *BinaryBuilder) Build() error {
	if b.version == "" {
		return errors.ErrBuildNoVersion
	}

	if b.commit == "" {
		return errors.ErrBuildNoCommit
	}

	if b.buildDir == "" {
		return errors.ErrBuildNoBuildDir
	}

	if b.date == "" {
		return errors.ErrBuildNoDate
	}
	return nil
}

func (b *BinaryBuilder) GetBuildTargets() []BuildTarget {
	return []BuildTarget{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"windows", "amd64"},
		{"windows", "arm64"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
	}
}

func (b *BinaryBuilder) BuildBinary(ctx context.Context, target BuildTarget) (string, error) {
	binaryName := b.binaryName
	if target.GOOS == "windows" {
		binaryName += ".exe"
	}

	outputPath := filepath.Join(b.buildDir, fmt.Sprintf("%s_%s_%s", binaryName, target.GOOS, target.GOARCH))
	if target.GOOS == "windows" {
		outputPath += ".exe"
	}

	ldflags := fmt.Sprintf(
		"-s -w -X main.version=%s -X main.commit=%s -X main.date=%s",
		b.version,
		b.commit,
		b.date,
	)

	cmd := exec.CommandContext(ctx, "go", "build",
		"-o", outputPath,
		"-ldflags", ldflags,
		"-trimpath",
		b.mainPath,
	)

	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		fmt.Sprintf("GOOS=%s", target.GOOS),
		fmt.Sprintf("GOARCH=%s", target.GOARCH),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.ErrBuildFailed.WithError(err).
			WithContext("platform", fmt.Sprintf("%s/%s", target.GOOS, target.GOARCH)).
			WithContext("output", string(output))
	}

	return outputPath, nil
}

func (b *BinaryBuilder) PackageBinary(binaryPath string, target BuildTarget) (string, error) {
	binaryName := b.binaryName
	if target.GOOS == "windows" {
		binaryName += ".exe"
	}

	version := strings.TrimPrefix(b.version, "v")
	binaryNameLower := strings.ToLower(binaryName)
	var archivePath string

	if target.GOOS == "windows" {
		archivePath = filepath.Join(b.buildDir, fmt.Sprintf("%s_%s_%s_%s.zip", binaryNameLower, version, target.GOOS, b.mapArch(target.GOARCH)))
		if err := b.createZip(binaryPath, archivePath, binaryName); err != nil {
			return "", errors.NewAppError(errors.TypeInternal, "failed to create zip archive", err)
		}
	} else {
		archivePath = filepath.Join(b.buildDir, fmt.Sprintf("%s_%s_%s_%s.tar.gz", binaryNameLower, version, target.GOOS, b.mapArch(target.GOARCH)))
		if err := b.createTarGz(binaryPath, archivePath, binaryName); err != nil {
			return "", errors.NewAppError(errors.TypeInternal, "failed to create tar.gz archive", err)
		}
	}

	return archivePath, nil
}

func (b *BinaryBuilder) mapArch(goarch string) string {
	archMap := map[string]string{
		"amd64": "x86_64",
		"arm64": "arm64",
		"386":   "i386",
	}
	if arch, ok := archMap[goarch]; ok {
		return arch
	}
	return goarch
}

func (b *BinaryBuilder) createZip(binaryPath, zipPath, binaryName string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = zipFile.Close()
	}()

	zipWriter := zip.NewWriter(zipFile)

	binaryFile, err := os.Open(binaryPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = binaryFile.Close()
	}()

	writer, err := zipWriter.Create(binaryName)
	if err != nil {
		return err
	}

	if _, err = io.Copy(writer, binaryFile); err != nil {
		return err
	}

	return zipWriter.Close()
}

func (b *BinaryBuilder) createTarGz(binaryPath, tarGzPath, binaryName string) error {
	tarGzFile, err := os.Create(tarGzPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = tarGzFile.Close()
	}()

	gzWriter := gzip.NewWriter(tarGzFile)

	tarWriter := tar.NewWriter(gzWriter)

	binaryFile, err := os.Open(binaryPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = binaryFile.Close()
	}()

	stat, err := binaryFile.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name:    binaryName,
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	if _, err = io.Copy(tarWriter, binaryFile); err != nil {
		return err
	}

	if err := tarWriter.Close(); err != nil {
		return err
	}
	return gzWriter.Close()
}

func (b *BinaryBuilder) BuildAndPackageAll(ctx context.Context, progressCh chan<- models.BuildProgress) ([]string, error) {
	log := logger.FromContext(ctx)
	targets := b.GetBuildTargets()

	var archives []string
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)

	// Enviar evento de inicio
	if progressCh != nil {
		progressCh <- models.BuildProgress{
			Type:  models.BuildProgressStart,
			Total: len(targets),
		}
	}

	log.Info("building binaries for all platforms",
		"platforms", len(targets))

	// Contador atómico para el progreso
	var completed int32

	for _, target := range targets {
		target := target
		g.Go(func() error {
			platform := fmt.Sprintf("%s/%s", target.GOOS, target.GOARCH)

			// Enviar evento de inicio de compilación de plataforma
			if progressCh != nil {
				progressCh <- models.BuildProgress{
					Type:     models.BuildProgressPlatform,
					Platform: platform,
					Current:  int(atomic.LoadInt32(&completed)) + 1,
					Total:    len(targets),
				}
			}

			log.Info("compiling binary",
				"platform", platform)

			binaryPath, err := b.BuildBinary(ctx, target)
			if err != nil {
				log.Error("build failed",
					"platform", platform,
					"error", err)
				return err
			}

			defer func() {
				if err := os.Remove(binaryPath); err != nil {
					return
				}
			}()

			archivePath, err := b.PackageBinary(binaryPath, target)
			if err != nil {
				log.Error("packaging failed",
					"platform", platform,
					"error", err)
				return err
			}

			mu.Lock()
			archives = append(archives, archivePath)
			mu.Unlock()

			// Incrementar contador
			current := atomic.AddInt32(&completed, 1)

			log.Info("binary ready",
				"platform", platform,
				"current", current,
				"total", len(targets),
				"archive", filepath.Base(archivePath))

			return nil
		})
	}

	// Wait retorna el primer error encontrado (si hay alguno)
	if err := g.Wait(); err != nil {
		if progressCh != nil {
			progressCh <- models.BuildProgress{
				Type:  models.BuildProgressError,
				Error: err,
			}
		}
		return nil, err
	}

	log.Info("all binaries built successfully",
		"total", len(archives))

	// Enviar evento de compilación completa
	if progressCh != nil {
		progressCh <- models.BuildProgress{
			Type:  models.BuildProgressComplete,
			Total: len(archives),
		}
	}

	return archives, nil
}

type DefaultBinaryBuilderFactory struct{}

func (f *DefaultBinaryBuilderFactory) NewBuilder(mainPath, binaryName string, opts ...Option) *BinaryBuilder {
	return NewBinaryBuilder(mainPath, binaryName, opts...)
}
