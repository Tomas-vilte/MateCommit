package dependency

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-vilte/matecommit/internal/models"
)

func TestCalculateSemverSeverity(t *testing.T) {
	tests := []struct {
		name       string
		oldVersion string
		newVersion string
		expected   models.ChangeSeverity
	}{
		{"major bump", "v1.2.3", "v2.0.0", models.MajorChange},
		{"minor bump", "v1.2.3", "v1.3.0", models.MinorChange},
		{"patch bump", "v1.2.3", "v1.2.4", models.PatchChange},
		{"with prefix", "1.2.3", "2.0.0", models.MajorChange},
		{"invalid", "abc", "def", models.UnknownChange},
		{"downgrade major", "v2.0.0", "v1.0.0", models.UnknownChange},
		{"same version", "v1.2.3", "v1.2.3", models.UnknownChange},
		{"pre-release", "v1.2.3-beta", "v1.3.0", models.MinorChange},
		{"invalid old version", "abc", "2.0.0", models.UnknownChange},
		{"invalid new version", "1.2.3", "xyz", models.UnknownChange},
		{"with build metadata", "1.2.3+build", "1.2.4", models.PatchChange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSemverSeverity(tt.oldVersion, tt.newVersion)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSemVer(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected []int
	}{
		{"valid semver", "1.2.3", []int{1, 2, 3}},
		{"with v prefix", "v1.2.3", []int{1, 2, 3}},
		{"with pre-release", "v1.2.3-beta.1", []int{1, 2, 3}},
		{"with build metadata", "v1.2.3+build.123", []int{1, 2, 3}},
		{"invalid version", "abc", []int{}},
		{"incomplete version", "1.2", []int{1, 2}},
		{"single number", "1", []int{1}},
		{"empty string", "", []int{}},
		{"only major", "v2", []int{2}},
		{"with dash", "1.2.3-rc1", []int{1, 2, 3}},
		{"with both pre-release and build", "1.2.3-rc1+build", []int{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSemVer(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}
