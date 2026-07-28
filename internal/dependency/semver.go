package dependency

import (
	"strconv"
	"strings"

	"github.com/thomas-vilte/matecommit/internal/models"
)

// calculateSemverSeverity determines the change severity between two version
// strings by comparing their major/minor/patch numbers.
func calculateSemverSeverity(oldVersion, newVersion string) models.ChangeSeverity {
	oldParts := parseSemVer(oldVersion)
	newParts := parseSemVer(newVersion)

	if len(oldParts) < 3 || len(newParts) < 3 {
		return models.UnknownChange
	}

	if newParts[0] > oldParts[0] {
		return models.MajorChange
	}
	if newParts[1] > oldParts[1] {
		return models.MinorChange
	}
	if newParts[2] > oldParts[2] {
		return models.PatchChange
	}

	return models.UnknownChange
}

// parseSemVer extracts [major, minor, patch] from a version string,
// tolerating a leading "v" and trailing pre-release/build metadata.
func parseSemVer(version string) []int {
	version = strings.TrimPrefix(version, "v")

	if idx := strings.IndexAny(version, "-+"); idx != -1 {
		version = version[:idx]
	}

	parts := strings.Split(version, ".")
	result := make([]int, 0, 3)

	for i := 0; i < 3 && i < len(parts); i++ {
		num, err := strconv.Atoi(parts[i])
		if err != nil {
			return []int{}
		}
		result = append(result, num)
	}
	return result
}
