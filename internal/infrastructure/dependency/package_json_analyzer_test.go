package dependency

import (
	"context"
	"errors"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPackageJsonAnalyzer_Name(t *testing.T) {
	analyzer := NewPackageJsonAnalyzer()
	assert.Equal(t, "package.json", analyzer.Name())
}

func TestPackageJsonAnalyzer_CanHandle(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*MockVCSClient)
		expected    bool
		description string
	}{
		{
			name: "package.json exists with content",
			setupMock: func(m *MockVCSClient) {
				m.On("GetFileAtTag", mock.Anything, "v1.0.0", "package.json").
					Return(`{"name": "test-project"}`, nil)
			},
			expected:    true,
			description: "should return true when package.json exists and has content",
		},
		{
			name: "package.json doesn't exist",
			setupMock: func(m *MockVCSClient) {
				m.On("GetFileAtTag", mock.Anything, "v1.0.0", "package.json").
					Return("", errors.New("file not found"))
			},
			expected:    false,
			description: "should return false when package.json doesn't exist",
		},
		{
			name: "package.json exists but is empty",
			setupMock: func(m *MockVCSClient) {
				m.On("GetFileAtTag", mock.Anything, "v1.0.0", "package.json").
					Return("", nil)
			},
			expected:    false,
			description: "should return false when package.json is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockVCSClient)
			tt.setupMock(mockClient)

			analyzer := NewPackageJsonAnalyzer()
			result := analyzer.CanHandle(context.Background(), mockClient, "v0.9.0", "v1.0.0")

			assert.Equal(t, tt.expected, result, tt.description)
			mockClient.AssertExpectations(t)
		})
	}
}

func TestPackageJsonAnalyzer_AnalyzeChanges(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*MockVCSClient)
		expectError bool
		errorMsg    string
		validate    func(*testing.T, []models.DependencyChange)
	}{
		{
			name: "successful analysis with changes",
			setupMock: func(m *MockVCSClient) {
				oldContent := `{
					"dependencies": {
						"express": "^4.17.1"
					}
				}`
				newContent := `{
					"dependencies": {
						"express": "^5.0.0",
						"lodash": "^4.17.21"
					}
				}`
				m.On("GetFileAtTag", mock.Anything, "v0.9.0", "package.json").Return(oldContent, nil)
				m.On("GetFileAtTag", mock.Anything, "v1.0.0", "package.json").Return(newContent, nil)
			},
			expectError: false,
			validate: func(t *testing.T, changes []models.DependencyChange) {
				assert.Len(t, changes, 2)
			},
		},
		{
			name: "error reading old package.json",
			setupMock: func(m *MockVCSClient) {
				m.On("GetFileAtTag", mock.Anything, "v0.9.0", "package.json").
					Return("", errors.New("tag not found"))
			},
			expectError: true,
			errorMsg:    "failed to read old package.json",
		},
		{
			name: "error reading new package.json",
			setupMock: func(m *MockVCSClient) {
				m.On("GetFileAtTag", mock.Anything, "v0.9.0", "package.json").
					Return(`{"dependencies": {}}`, nil)
				m.On("GetFileAtTag", mock.Anything, "v1.0.0", "package.json").
					Return("", errors.New("tag not found"))
			},
			expectError: true,
			errorMsg:    "failed to read new package.json",
		},
		{
			name: "error parsing old package.json",
			setupMock: func(m *MockVCSClient) {
				m.On("GetFileAtTag", mock.Anything, "v0.9.0", "package.json").
					Return("invalid json", nil)
				m.On("GetFileAtTag", mock.Anything, "v1.0.0", "package.json").
					Return(`{"dependencies": {}}`, nil)
			},
			expectError: true,
			errorMsg:    "failed to parse old package.json",
		},
		{
			name: "error parsing new package.json",
			setupMock: func(m *MockVCSClient) {
				m.On("GetFileAtTag", mock.Anything, "v0.9.0", "package.json").
					Return(`{"dependencies": {}}`, nil)
				m.On("GetFileAtTag", mock.Anything, "v1.0.0", "package.json").
					Return("invalid json", nil)
			},
			expectError: true,
			errorMsg:    "failed to parse new package.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockVCSClient)
			tt.setupMock(mockClient)

			analyzer := NewPackageJsonAnalyzer()
			changes, err := analyzer.AnalyzeChanges(context.Background(), mockClient, "v0.9.0", "v1.0.0")

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, changes)
				}
			}
			mockClient.AssertExpectations(t)
		})
	}
}

func TestPackageJsonAnalyzer_ParsePackageJson(t *testing.T) {
	analyzer := NewPackageJsonAnalyzer()

	t.Run("complete package.json with dependencies and devDependencies", func(t *testing.T) {
		content := `{
			"name": "test-project",
			"version": "1.0.0",
			"dependencies": {
				"express": "^4.17.1",
				"lodash": "~4.17.21"
			},
			"devDependencies": {
				"jest": "^27.0.0",
				"eslint": "^8.0.0"
			}
		}`

		deps, err := analyzer.parsePackageJson(content)
		assert.NoError(t, err)
		assert.Len(t, deps, 4)

		assert.Equal(t, "^4.17.1", deps["express"].version)
		assert.False(t, deps["express"].isDev)

		assert.Equal(t, "~4.17.21", deps["lodash"].version)
		assert.False(t, deps["lodash"].isDev)

		assert.Equal(t, "^27.0.0", deps["jest"].version)
		assert.True(t, deps["jest"].isDev)

		assert.Equal(t, "^8.0.0", deps["eslint"].version)
		assert.True(t, deps["eslint"].isDev)
	})

	t.Run("only dependencies", func(t *testing.T) {
		content := `{
			"dependencies": {
				"express": "^4.17.1"
			}
		}`

		deps, err := analyzer.parsePackageJson(content)
		assert.NoError(t, err)
		assert.Len(t, deps, 1)
		assert.False(t, deps["express"].isDev)
	})

	t.Run("only devDependencies", func(t *testing.T) {
		content := `{
			"devDependencies": {
				"jest": "^27.0.0"
			}
		}`

		deps, err := analyzer.parsePackageJson(content)
		assert.NoError(t, err)
		assert.Len(t, deps, 1)
		assert.True(t, deps["jest"].isDev)
	})

	t.Run("empty package.json", func(t *testing.T) {
		content := `{}`

		deps, err := analyzer.parsePackageJson(content)
		assert.NoError(t, err)
		assert.Empty(t, deps)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		content := `{invalid json}`

		deps, err := analyzer.parsePackageJson(content)
		assert.Error(t, err)
		assert.Nil(t, deps)
	})
}

func TestPackageJsonAnalyzer_ComputeChanges(t *testing.T) {
	analyzer := NewPackageJsonAnalyzer()

	oldDeps := map[string]npmDep{
		"express": {version: "^4.17.1", isDev: false},
		"old-pkg": {version: "^1.0.0", isDev: true},
	}

	newDeps := map[string]npmDep{
		"express": {version: "^5.0.0", isDev: false},
		"new-pkg": {version: "^1.0.0", isDev: false},
	}

	changes := analyzer.computeChanges(oldDeps, newDeps)

	assert.Len(t, changes, 3)

	var updated, added, removed *models.DependencyChange
	for i := range changes {
		switch changes[i].Type {
		case models.DependencyUpdated:
			if changes[i].Name == "express" {
				updated = &changes[i]
			}
		case models.DependencyAdded:
			if changes[i].Name == "new-pkg" {
				added = &changes[i]
			}
		case models.DependencyRemoved:
			if changes[i].Name == "old-pkg" {
				removed = &changes[i]
			}
		}
	}

	if assert.NotNil(t, updated, "updated change should exist") {
		assert.Equal(t, "4.17.1", updated.OldVersion)
		assert.Equal(t, "5.0.0", updated.NewVersion)
		assert.Equal(t, models.MajorChange, updated.Severity)
		assert.True(t, updated.IsDirect)
	}

	if assert.NotNil(t, added, "added change should exist") {
		assert.Equal(t, "1.0.0", added.NewVersion)
		assert.True(t, added.IsDirect)
	}

	if assert.NotNil(t, removed, "removed change should exist") {
		assert.Equal(t, "1.0.0", removed.OldVersion)
		assert.False(t, removed.IsDirect) // era devDependency
	}
}

func TestPackageJsonAnalyzer_CleanVersion(t *testing.T) {
	analyzer := NewPackageJsonAnalyzer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"caret version", "^4.17.1", "4.17.1"},
		{"tilde version", "~4.17.1", "4.17.1"},
		{"greater than or equal", ">=4.17.1", "4.17.1"},
		{"less than or equal", "<=4.17.1", "4.17.1"},
		{"greater than", ">4.17.1", "4.17.1"},
		{"less than", "<4.17.1", "4.17.1"},
		{"exact version", "=4.17.1", "4.17.1"},
		{"plain version", "4.17.1", "4.17.1"},
		{"version with spaces", " 4.17.1 ", "4.17.1"},
		{"caret with spaces", "^ 4.17.1", "4.17.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.cleanVersion(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPackageJsonAnalyzer_CalculateSeverity(t *testing.T) {
	analyzer := NewPackageJsonAnalyzer()

	tests := []struct {
		name       string
		oldVersion string
		newVersion string
		expected   models.ChangeSeverity
	}{
		{"major bump with caret", "^1.2.3", "^2.0.0", models.MajorChange},
		{"minor bump with tilde", "~1.2.3", "~1.3.0", models.MinorChange},
		{"patch bump", "1.2.3", "1.2.4", models.PatchChange},
		{"major bump plain", "1.2.3", "2.0.0", models.MajorChange},
		{"invalid old version", "abc", "2.0.0", models.UnknownChange},
		{"invalid new version", "1.2.3", "xyz", models.UnknownChange},
		{"same version", "^1.2.3", "^1.2.3", models.UnknownChange},
		{"downgrade", "^2.0.0", "^1.0.0", models.UnknownChange},
		{"with pre-release", "1.2.3-beta", "1.3.0", models.MinorChange},
		{"with build metadata", "1.2.3+build", "1.2.4", models.PatchChange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.calculateSeverity(tt.oldVersion, tt.newVersion)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPackageJsonAnalyzer_ParseVersion(t *testing.T) {
	analyzer := NewPackageJsonAnalyzer()

	tests := []struct {
		name     string
		version  string
		expected []int
	}{
		{"valid semver", "1.2.3", []int{1, 2, 3}},
		{"with pre-release", "1.2.3-beta.1", []int{1, 2, 3}},
		{"with build metadata", "1.2.3+build.123", []int{1, 2, 3}},
		{"with both pre-release and build", "1.2.3-rc1+build", []int{1, 2, 3}},
		{"invalid version", "abc.def.ghi", []int{}},
		{"incomplete version", "1.2", []int{1, 2}},
		{"single number", "1", []int{1}},
		{"empty string", "", []int{}},
		{"only major", "2", []int{2}},
		{"with dash in patch", "1.2.3-rc1", []int{1, 2, 3}},
		{"with plus in patch", "1.2.3+build", []int{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.parseVersion(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}
