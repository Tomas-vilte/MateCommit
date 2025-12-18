package issues

import (
	"context"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/stretchr/testify/mock"
)

type MockIssueGeneratorService struct {
	mock.Mock
}

func (m *MockIssueGeneratorService) GenerateFromDiff(ctx context.Context, hint string, skipLabels bool) (*models.IssueGenerationResult, error) {
	args := m.Called(ctx, hint, skipLabels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.IssueGenerationResult), args.Error(1)
}

func (m *MockIssueGeneratorService) GenerateFromDescription(ctx context.Context, description string, skipLabels bool) (*models.IssueGenerationResult, error) {
	args := m.Called(ctx, description, skipLabels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.IssueGenerationResult), args.Error(1)
}

func (m *MockIssueGeneratorService) GenerateFromPR(ctx context.Context, prNumber int, hint string, skipLabels bool) (*models.IssueGenerationResult, error) {
	args := m.Called(ctx, prNumber, hint, skipLabels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.IssueGenerationResult), args.Error(1)
}

func (m *MockIssueGeneratorService) GenerateWithTemplate(ctx context.Context, templateName string, hint string, fromDiff bool, description string, skipLabels bool) (*models.IssueGenerationResult, error) {
	args := m.Called(ctx, templateName, hint, fromDiff, description, skipLabels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.IssueGenerationResult), args.Error(1)
}

func (m *MockIssueGeneratorService) CreateIssue(ctx context.Context, result *models.IssueGenerationResult, assignees []string) (*models.Issue, error) {
	args := m.Called(ctx, result, assignees)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Issue), args.Error(1)
}

func (m *MockIssueGeneratorService) GetAuthenticatedUser(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockIssueGeneratorService) InferBranchName(issueNumber int, labels []string) string {
	args := m.Called(issueNumber, labels)
	return args.String(0)
}

func (m *MockIssueGeneratorService) LinkIssueToPR(ctx context.Context, prNumber int, issueNumber int) error {
	args := m.Called(ctx, prNumber, issueNumber)
	return args.Error(0)
}

type MockIssueTemplateService struct {
	mock.Mock
}

func (m *MockIssueTemplateService) GetTemplatesDir() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockIssueTemplateService) ListTemplates() ([]models.TemplateMetadata, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TemplateMetadata), args.Error(1)
}

func (m *MockIssueTemplateService) LoadTemplate(filePath string) (*models.IssueTemplate, error) {
	args := m.Called(filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.IssueTemplate), args.Error(1)
}

func (m *MockIssueTemplateService) GetTemplateByName(name string) (*models.IssueTemplate, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.IssueTemplate), args.Error(1)
}

func (m *MockIssueTemplateService) InitializeTemplates(force bool) error {
	args := m.Called(force)
	return args.Error(0)
}

func (m *MockIssueTemplateService) MergeWithGeneratedContent(template *models.IssueTemplate, generated *models.IssueGenerationResult) *models.IssueGenerationResult {
	args := m.Called(template, generated)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*models.IssueGenerationResult)
}
