package release

import (
	"fmt"
	"strconv"
	"strings"
)

// getPreviousVersion calculates the previous version from a given version
// e.g.: v1.3.0 -> v1.2.0
func getPreviousVersion(version string) (string, error) {
	version = strings.TrimPrefix(version, "v")

	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid version format: %s", version)
	}

	// Parse parts
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", fmt.Errorf("invalid patch version: %s", parts[2])
	}

	if patch > 0 {
		patch--
	} else if minor > 0 {
		minor--
		patch = 0
	} else if major > 0 {
		major--
		minor = 0
		patch = 0
	} else {
		return "", fmt.Errorf("cannot decrement version v0.0.0")
	}

	return fmt.Sprintf("v%d.%d.%d", major, minor, patch), nil
}
