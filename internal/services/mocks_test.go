package services

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/thomas-vilte/matecommit/internal/models"
)

type (
	MockAIProvider struct {
		mock.Mock
	}

	MockJiraService struct {
		mock.Mock
	}

	MockPRSummarizer struct {
		mock.Mock
	}

	MockReleaseNotesGenerator struct {
		mock.Mock
	}

	MockIssueContentGenerator struct {
		mock.Mock
	}

	MockIssueTemplateService struct {
		mock.Mock
	}
)

func (m *MockJiraService) GetTicketInfo(ticketID string) (*models.TicketInfo, error) {
	args := m.Called(ticketID)
	return args.Get(0).(*models.TicketInfo), args.Error(1)
}

func (m *MockAIProvider) GenerateSuggestions(ctx context.Context, info models.CommitInfo, count int) ([]models.CommitSuggestion, error) {
	args := m.Called(ctx, info, count)
	return args.Get(0).([]models.CommitSuggestion), args.Error(1)
}

func (m *MockPRSummarizer) GeneratePRSummary(ctx context.Context, prompt string, availableLabels []string) (models.PRSummary, error) {
	args := m.Called(ctx, prompt, availableLabels)
	return args.Get(0).(models.PRSummary), args.Error(1)
}

func (m *MockReleaseNotesGenerator) GenerateNotes(ctx context.Context, release *models.Release) (*models.ReleaseNotes, error) {
	args := m.Called(ctx, release)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ReleaseNotes), args.Error(1)
}

func (m *MockIssueContentGenerator) GenerateIssueContent(ctx context.Context, request models.IssueGenerationRequest) (*models.IssueGenerationResult, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.IssueGenerationResult), args.Error(1)
}

func (m *MockIssueTemplateService) GetTemplatesDir(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockIssueTemplateService) ListTemplates(ctx context.Context) ([]models.TemplateMetadata, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.TemplateMetadata), args.Error(1)
}

func (m *MockIssueTemplateService) LoadTemplate(ctx context.Context, filePath string) (*models.IssueTemplate, error) {
	args := m.Called(ctx, filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.IssueTemplate), args.Error(1)
}

func (m *MockIssueTemplateService) GetTemplateByName(ctx context.Context, name string) (*models.IssueTemplate, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.IssueTemplate), args.Error(1)
}

func (m *MockIssueTemplateService) InitializeTemplates(ctx context.Context, force bool) error {
	args := m.Called(ctx, force)
	return args.Error(0)
}

func (m *MockIssueTemplateService) MergeWithGeneratedContent(template *models.IssueTemplate, generated *models.IssueGenerationResult) *models.IssueGenerationResult {
	args := m.Called(template, generated)
	return args.Get(0).(*models.IssueGenerationResult)
}
