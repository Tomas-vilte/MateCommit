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

func TestAnalyzerRegistry_RegisterAnalyzer(t *testing.T) {
	registry := NewAnalyzerRegistry()
	initialCount := len(registry.analyzers)

	mockAnalyzer := new(MockAnalyzer)
	registry.RegisterAnalyzer(mockAnalyzer)

	assert.Len(t, registry.analyzers, initialCount+1, "should have one more analyzer after registration")
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
			// Create a registry with only mock analyzers
			registry := &AnalyzerRegistry{
				analyzers: []vcs.DependencyAnalyzer{},
			}

			mockVCS := new(MockVCSClient)
			mockAnalyzer1 := new(MockAnalyzer)
			mockAnalyzer2 := new(MockAnalyzer)

			registry.RegisterAnalyzer(mockAnalyzer1)
			registry.RegisterAnalyzer(mockAnalyzer2)

			tt.setupMocks(mockVCS, mockAnalyzer1, mockAnalyzer2)

			changes, err := registry.AnalyzeAll(context.Background(), mockVCS, "v1.0.0", "v2.0.0")

			assert.NoError(t, err, "AnalyzeAll should not return error")
			assert.Len(t, changes, tt.expectedCount, tt.description)

			mockAnalyzer1.AssertExpectations(t)
			mockAnalyzer2.AssertExpectations(t)
		})
	}
}

func TestAnalyzerRegistry_GetSupportedAnalyzers(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockVCSClient, *MockAnalyzer, *MockAnalyzer)
		expectedNames []string
		expectedCount int
		description   string
	}{
		{
			name: "both analyzers supported",
			setupMocks: func(vcs *MockVCSClient, a1 *MockAnalyzer, a2 *MockAnalyzer) {
				a1.On("CanHandle", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return(true)
				a1.On("Name").Return("analyzer1")

				a2.On("CanHandle", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return(true)
				a2.On("Name").Return("analyzer2")
			},
			expectedNames: []string{"analyzer1", "analyzer2"},
			expectedCount: 2,
			description:   "should return both analyzer names",
		},
		{
			name: "only one analyzer supported",
			setupMocks: func(vcs *MockVCSClient, a1 *MockAnalyzer, a2 *MockAnalyzer) {
				a1.On("CanHandle", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return(true)
				a1.On("Name").Return("analyzer1")

				a2.On("CanHandle", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return(false)
			},
			expectedNames: []string{"analyzer1"},
			expectedCount: 1,
			description:   "should return only supported analyzer name",
		},
		{
			name: "no analyzers supported",
			setupMocks: func(vcs *MockVCSClient, a1 *MockAnalyzer, a2 *MockAnalyzer) {
				a1.On("CanHandle", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return(false)
				a2.On("CanHandle", mock.Anything, vcs, "v1.0.0", "v2.0.0").Return(false)
			},
			expectedNames: []string{},
			expectedCount: 0,
			description:   "should return empty list when no analyzers are supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a registry with only mock analyzers
			registry := &AnalyzerRegistry{
				analyzers: []vcs.DependencyAnalyzer{},
			}

			mockVCS := new(MockVCSClient)
			mockAnalyzer1 := new(MockAnalyzer)
			mockAnalyzer2 := new(MockAnalyzer)

			registry.RegisterAnalyzer(mockAnalyzer1)
			registry.RegisterAnalyzer(mockAnalyzer2)

			tt.setupMocks(mockVCS, mockAnalyzer1, mockAnalyzer2)

			supported := registry.GetSupportedAnalyzers(context.Background(), mockVCS, "v1.0.0", "v2.0.0")

			assert.Len(t, supported, tt.expectedCount, tt.description)
			if tt.expectedCount > 0 {
				assert.Equal(t, tt.expectedNames, supported)
			}

			mockAnalyzer1.AssertExpectations(t)
			mockAnalyzer2.AssertExpectations(t)
		})
	}
}

func TestAnalyzerRegistry_Integration(t *testing.T) {
	t.Run("default registry has go.mod and package.json analyzers", func(t *testing.T) {
		registry := NewAnalyzerRegistry()

		// Create a mock VCS that returns content for both files
		mockVCS := new(MockVCSClient)
		mockVCS.On("GetFileAtTag", mock.Anything, "v1.0.0", "go.mod").
			Return("module test\n", nil)
		mockVCS.On("GetFileAtTag", mock.Anything, "v1.0.0", "package.json").
			Return(`{"name": "test"}`, nil)

		supported := registry.GetSupportedAnalyzers(context.Background(), mockVCS, "v0.9.0", "v1.0.0")

		assert.Contains(t, supported, "go.mod")
		assert.Contains(t, supported, "package.json")
		assert.Len(t, supported, 2)

		mockVCS.AssertExpectations(t)
	})

	t.Run("can add custom analyzer to default registry", func(t *testing.T) {
		registry := NewAnalyzerRegistry()

		customAnalyzer := new(MockAnalyzer)
		customAnalyzer.On("Name").Return("custom-analyzer")

		registry.RegisterAnalyzer(customAnalyzer)

		assert.Len(t, registry.analyzers, 3, "should have 3 analyzers (2 default + 1 custom)")
	})
}
