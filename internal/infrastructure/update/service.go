package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ReleaseInfo represents the simplified structure of a GitHub release
type ReleaseInfo struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

// Service handles checking for updates
type Service struct {
	client *http.Client
}

// NewUpdateService creates a new update service with a timeout
func NewUpdateService() *Service {
	return &Service{
		client: &http.Client{Timeout: 2 * time.Second},
	}
}

// CheckForUpdates compares the current version with the latest release on GitHub
// Returns the ReleaseInfo if a new version is available, nil otherwise.
func (s *Service) CheckForUpdates(currentVersion string) (*ReleaseInfo, error) {
	url := "https://api.github.com/repos/Tomas-vilte/MateCommit/releases/latest"

	resp, err := s.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error checking updates: status %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	// Normalize versions (remove 'v' prefix if present)
	remoteVer := strings.TrimPrefix(release.TagName, "v")
	currentVer := strings.TrimPrefix(currentVersion, "v")

	// TODO: Use semver for more robust comparison in the future
	if remoteVer != currentVer && remoteVer != "" {
		// Basic string check isn't perfect (1.10 < 1.9 is false, but string wise it might fail depending on length)
		// But for now, if they are different, we assume it's an update or user is on dev version.
		// A better check: if remote != current.
		// We should only notify if we are strictly essentially "different" and likely newer.
		// For MVP, if they don't match, and it's not a downgrade (which we can't easily accept without semver), we notify.
		// To be safer, let's just check inequality.
		return &release, nil
	}

	return nil, nil
}
