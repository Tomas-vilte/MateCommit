package dependency

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/vcs"
)

// MockAnalyzer es un mock de DependencyAnalyzer
type MockAnalyzer struct {
	mock.Mock
}

func (m *MockAnalyzer) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockAnalyzer) CanHandle(ctx context.Context, vcsClient vcs.VCSClient, previousTag, currentTag string) bool {
	args := m.Called(ctx, vcsClient, previousTag, currentTag)
	return args.Bool(0)
}

func (m *MockAnalyzer) AnalyzeChanges(ctx context.Context, vcsClient vcs.VCSClient, previousTag, currentTag string) ([]models.DependencyChange, error) {
	args := m.Called(ctx, vcsClient, previousTag, currentTag)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.DependencyChange), args.Error(1)
}

func TestNewAnalyzerRegistry(t *testing.T) {
	registry := NewAnalyzerRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.analyzers)
	assert.Len(t, registry.analyzers, 2, "should have 2 default analyzers (GoMod and PackageJson)")
}

func TestAnalyzerRegistry_AnalyzeAll(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockVCSClient, *MockAnalyzer, *MockAnalyzer)
		expectedCount int
		description   string
	}{
		{
			name: "both analyzers can handle",
			setupMocks: func(vcs *MockVCSClient, a1 *MockAnalyzer, a2 *MockAnalyzer) {
				a1.On("CanHandle", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return(true)
				a1.On("AnalyzeChanges", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return([]models.DependencyChange{
					{Name: "dep1", Type: models.DependencyAdded},
				}, nil)

				a2.On("CanHandle", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return(true)
				a2.On("AnalyzeChanges", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return([]models.DependencyChange{
					{Name: "dep2", Type: models.DependencyAdded},
				}, nil)
			},
			expectedCount: 2,
			description:   "should combine changes from both analyzers",
		},
		{
			name: "only one analyzer can handle",
			setupMocks: func(vcs *MockVCSClient, a1 *MockAnalyzer, a2 *MockAnalyzer) {
				a1.On("CanHandle", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return(true)
				a1.On("AnalyzeChanges", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return([]models.DependencyChange{
					{Name: "dep1", Type: models.DependencyAdded},
				}, nil)

				a2.On("CanHandle", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return(false)
			},
			expectedCount: 1,
			description:   "should only use analyzer that can handle",
		},
		{
			name: "no analyzers can handle",
			setupMocks: func(vcs *MockVCSClient, a1 *MockAnalyzer, a2 *MockAnalyzer) {
				a1.On("CanHandle", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return(false)
				a2.On("CanHandle", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return(false)
			},
			expectedCount: 0,
			description:   "should return empty changes when no analyzer can handle",
		},
		{
			name: "analyzer returns error",
			setupMocks: func(vcs *MockVCSClient, a1 *MockAnalyzer, a2 *MockAnalyzer) {
				a1.On("CanHandle", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return(true)
				a1.On("AnalyzeChanges", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return(nil, errors.New("analysis error"))

				a2.On("CanHandle", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return(true)
				a2.On("AnalyzeChanges", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return([]models.DependencyChange{
					{Name: "dep2", Type: models.DependencyAdded},
				}, nil)
			},
			expectedCount: 1,
			description:   "should continue with other analyzers when one fails",
		},
		{
			name: "analyzer returns empty changes",
			setupMocks: func(vcs *MockVCSClient, a1 *MockAnalyzer, a2 *MockAnalyzer) {
				a1.On("CanHandle", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return(true)
				a1.On("AnalyzeChanges", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return([]models.DependencyChange{}, nil)

				a2.On("CanHandle", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return(true)
				a2.On("AnalyzeChanges", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return([]models.DependencyChange{
					{Name: "dep2", Type: models.DependencyAdded},
				}, nil)
			},
			expectedCount: 1,
			description:   "should handle empty changes from one analyzer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockVCS := new(MockVCSClient)
			mockAnalyzer1 := new(MockAnalyzer)
			mockAnalyzer2 := new(MockAnalyzer)

			registry := &AnalyzerRegistry{
				analyzers: []vcs.DependencyAnalyzer{mockAnalyzer1, mockAnalyzer2},
			}

			tt.setupMocks(mockVCS, mockAnalyzer1, mockAnalyzer2)

			changes, err := registry.AnalyzeAll(context.Background(), mockVCS, "v1.0.0", "v2.0.0")

			assert.NoError(t, err, "AnalyzeAll should not return error")
			assert.Len(t, changes, tt.expectedCount, tt.description)

			mockAnalyzer1.AssertExpectations(t)
			mockAnalyzer2.AssertExpectations(t)
		})
	}
}

func TestAnalyzerRegistry_Integration(t *testing.T) {
	t.Run("default registry has go.mod and package.json analyzers", func(t *testing.T) {
		registry := NewAnalyzerRegistry()

		names := make([]string, 0, len(registry.analyzers))
		for _, a := range registry.analyzers {
			names = append(names, a.Name())
		}

		assert.Contains(t, names, "go.mod")
		assert.Contains(t, names, "package.json")
		assert.Len(t, names, 2)
	})

	t.Run("AnalyzeAll works with real go.mod and package.json content", func(t *testing.T) {
		registry := NewAnalyzerRegistry()

		mockVCS := new(MockVCSClient)
		mockVCS.On("GetFileAtTag", mock.Anything, "v1.0.0", "go.mod").
			Return("module test\n", nil)
		mockVCS.On("GetFileAtTag", mock.Anything, "v1.0.0", "package.json").
			Return(`{"name": "test"}`, nil)
		mockVCS.On("GetFileAtTag", mock.Anything, "v0.9.0", "go.mod").
			Return("module test\n", nil)
		mockVCS.On("GetFileAtTag", mock.Anything, "v0.9.0", "package.json").
			Return(`{"name": "test"}`, nil)

		_, err := registry.AnalyzeAll(context.Background(), mockVCS, "v0.9.0", "v1.0.0")

		assert.NoError(t, err)
	})
}
