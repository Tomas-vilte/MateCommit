package dependency

import (
	"context"
	"errors"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockVCSClient es un mock del VCSClient
type MockVCSClient struct {
	mock.Mock
}

func (m *MockVCSClient) GetFileAtTag(ctx context.Context, tag, filepath string) (string, error) {
	args := m.Called(ctx, tag, filepath)
	return args.String(0), args.Error(1)
}

func (m *MockVCSClient) UpdatePR(ctx context.Context, prNumber int, summary models.PRSummary) error {
	args := m.Called(ctx, prNumber, summary)
	return args.Error(0)
}

func (m *MockVCSClient) GetPR(ctx context.Context, prNumber int) (models.PRData, error) {
	args := m.Called(ctx, prNumber)
	return args.Get(0).(models.PRData), args.Error(1)
}

func (m *MockVCSClient) GetRepoLabels(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockVCSClient) CreateLabel(ctx context.Context, name string, color string, description string) error {
	args := m.Called(ctx, name, color, description)
	return args.Error(0)
}

func (m *MockVCSClient) AddLabelsToPR(ctx context.Context, prNumber int, labels []string) error {
	args := m.Called(ctx, prNumber, labels)
	return args.Error(0)
}

func (m *MockVCSClient) CreateRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool, buildBinaries bool) error {
	args := m.Called(ctx, release, notes, draft)
	return args.Error(0)
}

func (m *MockVCSClient) GetRelease(ctx context.Context, version string) (*models.VCSRelease, error) {
	args := m.Called(ctx, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.VCSRelease), args.Error(1)
}

func (m *MockVCSClient) UpdateRelease(ctx context.Context, version, body string) error {
	args := m.Called(ctx, version, body)
	return args.Error(0)
}

func (m *MockVCSClient) GetClosedIssuesBetweenTags(ctx context.Context, previousTag, currentTag string) ([]models.Issue, error) {
	args := m.Called(ctx, previousTag, currentTag)
	return args.Get(0).([]models.Issue), args.Error(1)
}

func (m *MockVCSClient) GetMergedPRsBetweenTags(ctx context.Context, previousTag, currentTag string) ([]models.PullRequest, error) {
	args := m.Called(ctx, previousTag, currentTag)
	return args.Get(0).([]models.PullRequest), args.Error(1)
}

func (m *MockVCSClient) GetContributorsBetweenTags(ctx context.Context, previousTag, currentTag string) ([]string, error) {
	args := m.Called(ctx, previousTag, currentTag)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockVCSClient) GetFileStatsBetweenTags(ctx context.Context, previousTag, currentTag string) (*models.FileStatistics, error) {
	args := m.Called(ctx, previousTag, currentTag)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FileStatistics), args.Error(1)
}

func (m *MockVCSClient) GetIssue(ctx context.Context, issueNumber int) (*models.Issue, error) {
	args := m.Called(ctx, issueNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Issue), args.Error(1)
}

func (m *MockVCSClient) GetPRIssues(ctx context.Context, branchName string, commits []string, prDescription string) ([]models.Issue, error) {
	args := m.Called(ctx, branchName, commits, prDescription)
	return args.Get(0).([]models.Issue), args.Error(1)
}

func (m *MockVCSClient) UpdateIssueChecklist(ctx context.Context, issueNumber int, indices []int) error {
	args := m.Called(ctx, issueNumber, indices)
	return args.Error(0)
}

func TestGoModAnalyzer_Name(t *testing.T) {
	analyzer := NewGoModAnalyzer()
	assert.Equal(t, "go.mod", analyzer.Name())
}

func TestGoModAnalyzer_CanHandle(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*MockVCSClient)
		expected    bool
		description string
	}{
		{
			name: "go.mod exists with content",
			setupMock: func(m *MockVCSClient) {
				m.On("GetFileAtTag", mock.Anything, "v1.0.0", "go.mod").
					Return("module github.com/test/project\n", nil)
			},
			expected:    true,
			description: "should return true when go.mod exists and has content",
		},
		{
			name: "go.mod doesn't exist",
			setupMock: func(m *MockVCSClient) {
				m.On("GetFileAtTag", mock.Anything, "v1.0.0", "go.mod").
					Return("", errors.New("file not found"))
			},
			expected:    false,
			description: "should return false when go.mod doesn't exist",
		},
		{
			name: "go.mod exists but is empty",
			setupMock: func(m *MockVCSClient) {
				m.On("GetFileAtTag", mock.Anything, "v1.0.0", "go.mod").
					Return("", nil)
			},
			expected:    false,
			description: "should return false when go.mod is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockVCSClient)
			tt.setupMock(mockClient)

			analyzer := NewGoModAnalyzer()
			result := analyzer.CanHandle(context.Background(), mockClient, "v0.9.0", "v1.0.0")

			assert.Equal(t, tt.expected, result, tt.description)
			mockClient.AssertExpectations(t)
		})
	}
}

func TestGoModAnalyzer_AnalyzeChanges(t *testing.T) {
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
				oldContent := `module test
require (
	github.com/foo/bar v1.0.0
)`
				newContent := `module test
require (
	github.com/foo/bar v2.0.0
	github.com/new/dep v1.0.0
)`
				m.On("GetFileAtTag", mock.Anything, "v0.9.0", "go.mod").Return(oldContent, nil)
				m.On("GetFileAtTag", mock.Anything, "v1.0.0", "go.mod").Return(newContent, nil)
			},
			expectError: false,
			validate: func(t *testing.T, changes []models.DependencyChange) {
				assert.Len(t, changes, 2)
			},
		},
		{
			name: "error reading old go.mod",
			setupMock: func(m *MockVCSClient) {
				m.On("GetFileAtTag", mock.Anything, "v0.9.0", "go.mod").
					Return("", errors.New("tag not found"))
			},
			expectError: true,
			errorMsg:    "error leyendo go.mod viejo",
		},
		{
			name: "error reading new go.mod",
			setupMock: func(m *MockVCSClient) {
				m.On("GetFileAtTag", mock.Anything, "v0.9.0", "go.mod").
					Return("module test\n", nil)
				m.On("GetFileAtTag", mock.Anything, "v1.0.0", "go.mod").
					Return("", errors.New("tag not found"))
			},
			expectError: true,
			errorMsg:    "error leyendo go.mod nuevo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockVCSClient)
			tt.setupMock(mockClient)

			analyzer := NewGoModAnalyzer()
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

func TestGoModAnalyzer_ParseGoMod(t *testing.T) {
	analyzer := NewGoModAnalyzer()

	t.Run("complete go.mod with multiple formats", func(t *testing.T) {
		content := `module github.com/user/project

go 1.21

require (
	github.com/stretchr/testify v1.8.4
	github.com/urfave/cli/v3 v3.0.0
	golang.org/x/oauth2 v0.15.0 // indirect
)

require github.com/google/uuid v1.5.0
`

		deps := analyzer.parseGoMod(content)

		assert.Equal(t, "1.8.4", deps["github.com/stretchr/testify"].version)
		assert.False(t, deps["github.com/stretchr/testify"].indirect)

		assert.Equal(t, "3.0.0", deps["github.com/urfave/cli/v3"].version)
		assert.False(t, deps["github.com/urfave/cli/v3"].indirect)

		assert.Equal(t, "0.15.0", deps["golang.org/x/oauth2"].version)
		assert.True(t, deps["golang.org/x/oauth2"].indirect)

		assert.Equal(t, "1.5.0", deps["github.com/google/uuid"].version)
		assert.False(t, deps["github.com/google/uuid"].indirect)
	})

	t.Run("empty content", func(t *testing.T) {
		deps := analyzer.parseGoMod("")
		assert.Empty(t, deps)
	})

	t.Run("only direct dependencies", func(t *testing.T) {
		content := `module test
require (
	github.com/foo/bar v1.0.0
	github.com/baz/qux v2.0.0
)`
		deps := analyzer.parseGoMod(content)
		assert.Len(t, deps, 2)
		assert.False(t, deps["github.com/foo/bar"].indirect)
		assert.False(t, deps["github.com/baz/qux"].indirect)
	})

	t.Run("only indirect dependencies", func(t *testing.T) {
		content := `module test
require (
	github.com/foo/bar v1.0.0 // indirect
	github.com/baz/qux v2.0.0 // indirect
)`
		deps := analyzer.parseGoMod(content)
		assert.Len(t, deps, 2)
		assert.True(t, deps["github.com/foo/bar"].indirect)
		assert.True(t, deps["github.com/baz/qux"].indirect)
	})

	t.Run("malformed content", func(t *testing.T) {
		content := `this is not a valid go.mod file
random text here
no dependencies`
		deps := analyzer.parseGoMod(content)
		assert.Empty(t, deps)
	})

	t.Run("multiple single-line requires", func(t *testing.T) {
		content := `module test
require github.com/foo/bar v1.0.0
require github.com/baz/qux v2.0.0 // indirect`
		deps := analyzer.parseGoMod(content)
		assert.Len(t, deps, 2)
		assert.Equal(t, "1.0.0", deps["github.com/foo/bar"].version)
		assert.False(t, deps["github.com/foo/bar"].indirect)
		assert.Equal(t, "2.0.0", deps["github.com/baz/qux"].version)
		assert.True(t, deps["github.com/baz/qux"].indirect)
	})
}

func TestGoModAnalyzer_ComputeChanges(t *testing.T) {
	analyzer := NewGoModAnalyzer()

	oldDeps := map[string]goDep{
		"github.com/foo/bar": {version: "1.0.0", indirect: false},
		"github.com/old/dep": {version: "1.0.0", indirect: true},
	}

	newDeps := map[string]goDep{
		"github.com/foo/bar": {version: "2.0.0", indirect: false},
		"github.com/new/dep": {version: "1.0.0", indirect: false},
	}

	changes := analyzer.computeChanges(oldDeps, newDeps)

	assert.Len(t, changes, 3)

	// Verificar que contiene los cambios esperados
	var updated, added, removed *models.DependencyChange
	for i := range changes {
		switch changes[i].Type {
		case models.DependencyUpdated:
			if changes[i].Name == "github.com/foo/bar" {
				updated = &changes[i]
			}
		case models.DependencyAdded:
			if changes[i].Name == "github.com/new/dep" {
				added = &changes[i]
			}
		case models.DependencyRemoved:
			if changes[i].Name == "github.com/old/dep" {
				removed = &changes[i]
			}
		}
	}

	if assert.NotNil(t, updated, "updated change should exist") {
		assert.Equal(t, "1.0.0", updated.OldVersion)
		assert.Equal(t, "2.0.0", updated.NewVersion)
		assert.Equal(t, models.MajorChange, updated.Severity)
		assert.True(t, updated.IsDirect)
	}

	if assert.NotNil(t, added, "added change should exist") {
		assert.Equal(t, "1.0.0", added.NewVersion)
		assert.True(t, added.IsDirect)
	}

	if assert.NotNil(t, removed, "removed change should exist") {
		assert.Equal(t, "1.0.0", removed.OldVersion)
		assert.False(t, removed.IsDirect) // era indirect
	}
}

func TestGoModAnalyzer_CalculateSeverity(t *testing.T) {
	analyzer := NewGoModAnalyzer()

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.calculateSeverity(tt.oldVersion, tt.newVersion)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGoModAnalyzer_ParseVersion(t *testing.T) {
	analyzer := NewGoModAnalyzer()

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.parseVersion(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}
