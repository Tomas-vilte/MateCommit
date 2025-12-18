package services

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/fatih/color"
	"github.com/google/go-github/v80/github"
	"golang.org/x/mod/semver"
)

type VersionUpdater struct {
	currentVersion string
	trans          *i18n.Translations
}

type UpdateCache struct {
	LastCheck   time.Time `json:"last_check"`
	LatestKnown string    `json:"latest_known"`
}

func NewVersionUpdater(version string, trans *i18n.Translations) *VersionUpdater {
	return &VersionUpdater{
		currentVersion: version,
		trans:          trans,
	}
}

func (v *VersionUpdater) CheckForUpdates(ctx context.Context) {
	if os.Getenv("MATECOMMIT_DISABLE_UPDATE_CHECK") != "" {
		return
	}

	cache, err := v.loadCache()
	if err == nil && time.Since(cache.LastCheck) < 24*time.Hour {
		if cache.LatestKnown != "" && v.isUpdateAvailable(cache.LatestKnown) {
			v.printUpdateNotification(cache.LatestKnown)
		}
		return
	}

	client := github.NewClient(nil)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	release, _, err := client.Repositories.GetLatestRelease(ctx, "Tomas-vilte", "MateCommit")
	if err != nil {
		return
	}

	latestVersion := release.GetTagName()

	_ = v.saveCache(UpdateCache{
		LastCheck:   time.Now(),
		LatestKnown: latestVersion,
	})

	if v.isUpdateAvailable(latestVersion) {
		v.printUpdateNotification(latestVersion)
	}
}

func (v *VersionUpdater) UpdateCLI(ctx context.Context) error {
	method := v.detectInstallMethod()

	switch method {
	case "go":
		return v.updateViaGo(ctx)
	case "brew":
		return v.updateViaBrew(ctx)
	case "binary":
		return v.updateViaBinary(ctx)
	default:
		return fmt.Errorf("%s", v.trans.GetMessage("update.method_not_detected", 0, nil))
	}
}

func (v *VersionUpdater) detectInstallMethod() string {
	execPath, err := os.Executable()
	if err != nil {
		return "unknown"
	}

	if strings.Contains(execPath, "/Cellar/") || strings.Contains(execPath, "homebrew") || strings.Contains(execPath, "/opt/homebrew") {
		return "brew"
	}

	if gopath := os.Getenv("GOPATH"); gopath != "" {
		if strings.HasPrefix(execPath, gopath) {
			return "go"
		}
	}

	if gobin := os.Getenv("GOBIN"); gobin != "" {
		if strings.HasPrefix(execPath, gobin) {
			return "go"
		}
	}

	return "binary"
}

func (v *VersionUpdater) updateViaGo(ctx context.Context) error {
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("%s", v.trans.GetMessage("update.go_not_found", 0, nil))
	}

	cmd := exec.CommandContext(ctx, "go", "install", "github.com/Tomas-vilte/MateCommit@latest")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", v.trans.GetMessage("update.error", 0, nil), string(output))
	}
	return nil
}

func (v *VersionUpdater) updateViaBrew(ctx context.Context) error {
	if _, err := exec.LookPath("brew"); err != nil {
		return fmt.Errorf("%s", v.trans.GetMessage("update.brew_not_found", 0, nil))
	}

	cmd := exec.CommandContext(ctx, "brew", "upgrade", "matecommit")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", v.trans.GetMessage("update.error", 0, nil), string(output))
	}
	return nil
}

func (v *VersionUpdater) updateViaBinary(ctx context.Context) error {
	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(ctx, "Tomas-vilte", "MateCommit")
	if err != nil {
		return fmt.Errorf("%s", v.trans.GetMessage("update.errors.get_latest", 0, map[string]interface{}{"Error": err}))
	}
	latestVersion := release.GetTagName()

	goos := runtime.GOOS
	goarch := runtime.GOARCH

	archMap := map[string]string{
		"amd64": "x86_64",
		"arm64": "arm64",
		"386":   "i386",
	}

	arch, ok := archMap[goarch]
	if !ok {
		return fmt.Errorf("%s", v.trans.GetMessage("update.errors.arch_not_supported", 0, map[string]interface{}{"Arch": goarch}))
	}

	var assetName string
	switch goos {
	case "linux":
		assetName = fmt.Sprintf("matecommit_%s_linux_%s.tar.gz", strings.TrimPrefix(latestVersion, "v"), arch)
	case "darwin":
		assetName = fmt.Sprintf("matecommit_%s_darwin_%s.tar.gz", strings.TrimPrefix(latestVersion, "v"), arch)
	case "windows":
		assetName = fmt.Sprintf("matecommit_%s_windows_%s.zip", strings.TrimPrefix(latestVersion, "v"), arch)
	default:
		return fmt.Errorf("%s", v.trans.GetMessage("update.errors.os_not_supported", 0, map[string]interface{}{"OS": goos}))
	}

	var assetURL string
	for _, asset := range release.Assets {
		if asset.GetName() == assetName {
			assetURL = asset.GetBrowserDownloadURL()
			break
		}
	}

	if assetURL == "" {
		return fmt.Errorf("%s", v.trans.GetMessage("update.errors.binary_not_found_manual", 0, map[string]interface{}{
			"OS":   goos,
			"Arch": goarch,
			"URL":  release.GetHTMLURL(),
		}))
	}

	tmpDir, err := os.MkdirTemp("", "matecommit-update-*")
	if err != nil {
		return fmt.Errorf("%s", v.trans.GetMessage("update.errors.create_temp", 0, map[string]interface{}{"Error": err}))
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			return
		}
	}()

	fmt.Println(v.trans.GetMessage("update.downloading", 0, map[string]interface{}{
		"Version": latestVersion,
	}))

	resp, err := http.Get(assetURL)
	if err != nil {
		return fmt.Errorf("%s", v.trans.GetMessage("update.errors.download", 0, map[string]interface{}{"Error": err}))
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			return
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", v.trans.GetMessage("update.errors.download_http", 0, map[string]interface{}{"StatusCode": resp.StatusCode}))
	}

	archivePath := filepath.Join(tmpDir, assetName)
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("%s", v.trans.GetMessage("update.errors.create_temp_file", 0, map[string]interface{}{"Error": err}))
	}

	_, err = io.Copy(archiveFile, resp.Body)
	_ = archiveFile.Close()
	if err != nil {
		return fmt.Errorf("%s", v.trans.GetMessage("update.errors.save_binary", 0, map[string]interface{}{"Error": err}))
	}

	fmt.Println(v.trans.GetMessage("update.extracting", 0, nil))

	var binaryPath string
	if strings.HasSuffix(assetName, ".tar.gz") {
		binaryPath, err = v.extractTarGz(archivePath, tmpDir)
	} else if strings.HasSuffix(assetName, ".zip") {
		binaryPath, err = v.extractZip(archivePath, tmpDir)
	} else {
		return fmt.Errorf("%s", v.trans.GetMessage("archive_format_not_supported", 0, map[string]interface{}{"Format": assetName}))
	}

	if err != nil {
		return fmt.Errorf("%s", v.trans.GetMessage("update.errors.extract", 0, map[string]interface{}{"Error": err}))
	}

	currentExec, err := os.Executable()
	if err != nil {
		return fmt.Errorf("%s", v.trans.GetMessage("update.errors.get_executable", 0, map[string]interface{}{"Error": err}))
	}

	currentExec, err = filepath.EvalSymlinks(currentExec)
	if err != nil {
		return fmt.Errorf("%s", v.trans.GetMessage("update.errors.resolve_symlinks", 0, map[string]interface{}{"Error": err}))
	}

	backupPath := currentExec + ".backup"
	if err := os.Rename(currentExec, backupPath); err != nil {
		return fmt.Errorf("%s", v.trans.GetMessage("update.errors.create_backup", 0, map[string]interface{}{"Error": err}))
	}

	fmt.Println(v.trans.GetMessage("update.installing", 0, nil))

	if err := v.copyFile(binaryPath, currentExec); err != nil {
		_ = os.Rename(backupPath, currentExec)
		return fmt.Errorf("%s", v.trans.GetMessage("update.errors.install_binary", 0, map[string]interface{}{"Error": err}))
	}

	if err := os.Chmod(currentExec, 0755); err != nil {
		_ = os.Remove(currentExec)
		_ = os.Rename(backupPath, currentExec)
		return fmt.Errorf("%s", v.trans.GetMessage("update.error.chmod", 0, map[string]interface{}{"Error": err}))
	}

	_ = os.Remove(backupPath)

	fmt.Println(v.trans.GetMessage("update.success", 0, map[string]interface{}{
		"Version": latestVersion,
	}))

	return nil
}

func (v *VersionUpdater) isUpdateAvailable(latest string) bool {
	current := v.currentVersion
	if !strings.HasPrefix(current, "v") {
		current = "v" + current
	}
	if !strings.HasPrefix(latest, "v") {
		latest = "v" + latest
	}

	if !semver.IsValid(current) || !semver.IsValid(latest) {
		return current != latest
	}

	return semver.Compare(latest, current) > 0
}

func (v *VersionUpdater) printUpdateNotification(latest string) {
	yellow := color.New(color.FgYellow, color.Bold).SprintFunc()
	green := color.New(color.FgGreen, color.Bold).SprintFunc()

	boxTop := v.trans.GetMessage("update.box_top", 0, nil)
	boxBottom := v.trans.GetMessage("update.box_bottom", 0, nil)

	msgAvailable := v.trans.GetMessage("update.available", 0, map[string]interface{}{
		"Current": v.currentVersion,
		"Latest":  green(latest),
	})

	method := v.detectInstallMethod()
	var updateCmd string
	switch method {
	case "brew":
		updateCmd = green("brew upgrade matecommit")
	case "go":
		updateCmd = green("go install github.com/Tomas-vilte/MateCommit@latest")
	default:
		updateCmd = green("matecommit update")
	}

	msgCommand := v.trans.GetMessage("update.command", 0, map[string]interface{}{
		"Command": updateCmd,
	})

	fmt.Printf("\n%s\n", yellow(boxTop))
	fmt.Printf("%s         %s\n", yellow("│"), yellow("│"))
	fmt.Printf("%s %s %s\n", yellow("│"), msgAvailable, yellow("│"))
	fmt.Printf("%s %s    %s\n", yellow("│"), msgCommand, yellow("│"))
	fmt.Printf("%s         %s\n", yellow("│"), yellow("│"))
	fmt.Printf("%s\n\n", yellow(boxBottom))
}

func (v *VersionUpdater) getCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(homeDir, ".config", "matecommit")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}
	return cacheDir, nil
}

func (v *VersionUpdater) loadCache() (UpdateCache, error) {
	cacheDir, err := v.getCacheDir()
	if err != nil {
		return UpdateCache{}, err
	}

	cachePath := filepath.Join(cacheDir, "last_update_check.json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return UpdateCache{}, err
	}

	var cache UpdateCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return UpdateCache{}, err
	}

	return cache, nil
}

func (v *VersionUpdater) saveCache(cache UpdateCache) error {
	cacheDir, err := v.getCacheDir()
	if err != nil {
		return err
	}

	cachePath := filepath.Join(cacheDir, "last_update_check.json")
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath, data, 0644)
}

func (v *VersionUpdater) extractTarGz(archivePath, destDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := file.Close(); err != nil {
			return
		}
	}()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := gzr.Close(); err != nil {
			return
		}
	}()

	tr := tar.NewReader(gzr)

	var binaryPath string
	binaryName := "matecommit"
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return "", err
			}
		case tar.TypeReg:
			outFile, err := os.Create(target)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				_ = outFile.Close()
				return "", err
			}
			_ = outFile.Close()

			baseName := filepath.Base(header.Name)
			if baseName == binaryName {
				binaryPath = target
				_ = os.Chmod(target, 0755)
			}
		}
	}

	if binaryPath == "" {
		return "", fmt.Errorf("%s", v.trans.GetMessage("binary_not_found_archive", 0, nil))
	}

	return binaryPath, nil
}

func (v *VersionUpdater) extractZip(archivePath, destDir string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := r.Close(); err != nil {
			return
		}
	}()

	var binaryPath string
	binaryName := "matecommit.exe"
	for _, f := range r.File {
		target := filepath.Join(destDir, f.Name)

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return "", err
		}

		outFile, err := os.Create(target)
		if err != nil {
			return "", err
		}

		rc, err := f.Open()
		if err != nil {
			_ = outFile.Close()
			return "", err
		}

		_, err = io.Copy(outFile, rc)
		_ = rc.Close()
		_ = outFile.Close()
		if err != nil {
			return "", err
		}

		baseName := filepath.Base(f.Name)
		if baseName == binaryName {
			binaryPath = target
			_ = os.Chmod(target, 0755)
		}
	}

	if binaryPath == "" {
		return "", fmt.Errorf("%s", v.trans.GetMessage("binary_not_found_archive", 0, nil))
	}

	return binaryPath, nil
}

func (v *VersionUpdater) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			return
		}
	}()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if err := destFile.Close(); err != nil {
			return
		}
	}()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
