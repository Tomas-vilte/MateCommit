package release

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/thomas-vilte/matecommit/internal/models"
)

type MockReleaseService struct {
	mock.Mock
}

func (m *MockReleaseService) AnalyzeNextRelease(ctx context.Context) (*models.Release, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Release), args.Error(1)
}

func (m *MockReleaseService) GenerateReleaseNotes(ctx context.Context, release *models.Release) (*models.ReleaseNotes, error) {
	args := m.Called(ctx, release)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ReleaseNotes), args.Error(1)
}

func (m *MockReleaseService) PublishRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool, buildBinaries bool, progressCh chan<- models.BuildProgress) error {
	args := m.Called(ctx, release, notes, draft, buildBinaries, progressCh)
	return args.Error(0)
}

func (m *MockReleaseService) CreateTag(ctx context.Context, version, message string) error {
	args := m.Called(ctx, version, message)
	return args.Error(0)
}

func (m *MockReleaseService) TagExists(ctx context.Context, version string) bool {
	args := m.Called(ctx, version)
	return args.Bool(0)
}

func (m *MockReleaseService) PushTag(ctx context.Context, version string) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *MockReleaseService) GetRelease(ctx context.Context, version string) (*models.VCSRelease, error) {
	args := m.Called(ctx, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.VCSRelease), args.Error(1)
}

func (m *MockReleaseService) UpdateRelease(ctx context.Context, version, body string) error {
	args := m.Called(ctx, version, body)
	return args.Error(0)
}

func (m *MockReleaseService) EnrichReleaseContext(ctx context.Context, release *models.Release) error {
	args := m.Called(ctx, release)
	return args.Error(0)
}

func (m *MockReleaseService) UpdateLocalChangelog(ctx context.Context, release *models.Release, notes *models.ReleaseNotes) error {
	args := m.Called(ctx, release, notes)
	return args.Error(0)
}

func (m *MockReleaseService) CommitChangelog(ctx context.Context, version string) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *MockReleaseService) PushChanges(ctx context.Context, version string) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *MockReleaseService) UpdateAppVersion(ctx context.Context, version string) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *MockReleaseService) ValidateMainBranch(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockReleaseService) BuildChangelogPreview(ctx context.Context, release *models.Release, notes *models.ReleaseNotes) string {
	args := m.Called(ctx, release, notes)
	return args.String(0)
}
