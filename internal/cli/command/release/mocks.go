package release

import (
	"context"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/stretchr/testify/mock"
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

func (m *MockReleaseService) PublishRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool) error {
	args := m.Called(ctx, release, notes, draft)
	return args.Error(0)
}

func (m *MockReleaseService) CreateTag(ctx context.Context, version, message string) error {
	args := m.Called(ctx, version, message)
	return args.Error(0)
}

func (m *MockReleaseService) PushTag(ctx context.Context, version string) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}
