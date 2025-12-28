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
	"time"

	"github.com/thomas-vilte/matecommit/internal/errors"
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

func (b *BinaryBuilder) BuildAndPackageAll(ctx context.Context) ([]string, error) {
	targets := b.GetBuildTargets()
	var archives []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	errChan := make(chan error, len(targets))

	for _, target := range targets {
		wg.Add(1)
		go func(t BuildTarget) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
			}

			binaryPath, err := b.BuildBinary(ctx, t)
			if err != nil {
				errChan <- err
				return
			}

			defer func() {
				if err := os.Remove(binaryPath); err != nil {
					errChan <- err
					return
				}
			}()

			archivePath, err := b.PackageBinary(binaryPath, target)
			if err != nil {
				errChan <- err
				return
			}

			mu.Lock()
			archives = append(archives, archivePath)
			mu.Unlock()
		}(target)
	}
	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return nil, <-errChan
	}

	return archives, nil
}

type DefaultBinaryBuilderFactory struct{}

func (f *DefaultBinaryBuilderFactory) NewBuilder(mainPath, binaryName string, opts ...Option) *BinaryBuilder {
	return NewBinaryBuilder(mainPath, binaryName, opts...)
}
