package di

import (
	"context"
	"errors"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai/registry"
	ticketregistry "github.com/Tomas-vilte/MateCommit/internal/infrastructure/tickets/registry"
	vcsregistry "github.com/Tomas-vilte/MateCommit/internal/infrastructure/vcs/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAIFactory struct {
	createCommitSummarizerError error
	createPRSummarizerError     error
}

func (m *mockAIFactory) CreateCommitSummarizer(_ context.Context, _ *config.Config, _ *i18n.Translations) (ports.CommitSummarizer, error) {
	if m.createCommitSummarizerError != nil {
		return nil, m.createCommitSummarizerError
	}
	return &mockCommitSummarizer{}, nil
}

func (m *mockAIFactory) CreatePRSummarizer(_ context.Context, _ *config.Config, _ *i18n.Translations) (ports.PRSummarizer, error) {
	if m.createPRSummarizerError != nil {
		return nil, m.createPRSummarizerError
	}
	return &mockPRSummarizer{}, nil
}

func (m *mockAIFactory) ValidateConfig(_ *config.Config) error {
	return nil
}

func (m *mockAIFactory) Name() string {
	return "mock"
}

type mockVCSFactory struct {
	createClientError error
}

func (m *mockVCSFactory) CreateClient(_ context.Context, _, _, _ string, _ *i18n.Translations) (ports.VCSClient, error) {
	if m.createClientError != nil {
		return nil, m.createClientError
	}
	return &mockVCSClient{}, nil
}

func (m *mockVCSFactory) ValidateConfig(_ *config.VCSConfig) error {
	return nil
}

func (m *mockVCSFactory) Name() string {
	return "mock"
}

type mockTicketFactory struct {
	createClientError error
}

func (m *mockTicketFactory) CreateClient(_ context.Context, _ config.TicketProviderConfig, _ *i18n.Translations) (ports.TickerManager, error) {
	if m.createClientError != nil {
		return nil, m.createClientError
	}
	return &mockTicketManager{}, nil
}

func (m *mockTicketFactory) ValidateConfig(_ config.TicketProviderConfig) error {
	return nil
}

func (m *mockTicketFactory) Name() string {
	return "mock"
}

type mockCommitSummarizer struct{}

func (m *mockCommitSummarizer) GenerateSuggestions(_ context.Context, _ models.CommitInfo, _ int) ([]models.CommitSuggestion, error) {
	return nil, nil
}

type mockPRSummarizer struct{}

func (m *mockPRSummarizer) GeneratePRSummary(_ context.Context, _ string) (models.PRSummary, error) {
	return models.PRSummary{}, nil
}

type mockVCSClient struct{}

func (m *mockVCSClient) UpdatePR(_ context.Context, _ int, _ models.PRSummary) error {
	return nil
}

func (m *mockVCSClient) GetPR(_ context.Context, _ int) (models.PRData, error) {
	return models.PRData{}, nil
}

func (m *mockVCSClient) GetRepoLabels(_ context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockVCSClient) CreateLabel(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *mockVCSClient) AddLabelsToPR(_ context.Context, _ int, _ []string) error {
	return nil
}

func (m *mockVCSClient) CreateRelease(_ context.Context, _ *models.Release, _ *models.ReleaseNotes, _, _ bool) error {
	return nil
}

func (m *mockVCSClient) GetRelease(_ context.Context, _ string) (*models.VCSRelease, error) {
	return nil, nil
}

func (m *mockVCSClient) UpdateRelease(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockVCSClient) GetClosedIssuesBetweenTags(_ context.Context, _, _ string) ([]models.Issue, error) {
	return nil, nil
}

func (m *mockVCSClient) GetMergedPRsBetweenTags(_ context.Context, _, _ string) ([]models.PullRequest, error) {
	return nil, nil
}

func (m *mockVCSClient) GetContributorsBetweenTags(_ context.Context, _, _ string) ([]string, error) {
	return nil, nil
}

func (m *mockVCSClient) GetFileStatsBetweenTags(_ context.Context, _, _ string) (*models.FileStatistics, error) {
	return nil, nil
}

func (m *mockVCSClient) GetIssue(_ context.Context, _ int) (*models.Issue, error) {
	return nil, nil
}

func (m *mockVCSClient) GetFileAtTag(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

func (m *mockVCSClient) GetPRIssues(_ context.Context, _ string, _ []string, _ string) ([]models.Issue, error) {
	return nil, nil
}

func (m *mockVCSClient) UpdateIssueChecklist(_ context.Context, _ int, _ []int) error {
	return nil
}

type mockTicketManager struct{}

func (m *mockTicketManager) GetTicketInfo(_ string) (*models.TicketInfo, error) {
	return nil, nil
}

type mockGitService struct{}

func (m *mockGitService) GetChangedFiles(_ context.Context) ([]models.GitChange, error) {
	return nil, nil
}

func (m *mockGitService) GetDiff(_ context.Context) (string, error) {
	return "", nil
}

func (m *mockGitService) HasStagedChanges(_ context.Context) bool {
	return false
}

func (m *mockGitService) CreateCommit(_ context.Context, _ string) error {
	return nil
}

func (m *mockGitService) AddFileToStaging(_ context.Context, _ string) error {
	return nil
}

func (m *mockGitService) GetCurrentBranch(_ context.Context) (string, error) {
	return "", nil
}

func (m *mockGitService) GetRepoInfo(_ context.Context) (string, string, string, error) {
	return "owner", "repo", "github", nil
}

func (m *mockGitService) GetLastTag(_ context.Context) (string, error) {
	return "", nil
}

func (m *mockGitService) GetCommitCount(_ context.Context) (int, error) {
	return 0, nil
}

func (m *mockGitService) GetCommitsSinceTag(_ context.Context, _ string) ([]models.Commit, error) {
	return nil, nil
}

func (m *mockGitService) GetCommitsBetweenTags(_ context.Context, _, _ string) ([]models.Commit, error) {
	return nil, nil
}

func (m *mockGitService) GetTagDate(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (m *mockGitService) GetRecentCommitMessages(_ context.Context, _ int) (string, error) {
	return "", nil
}

func (m *mockGitService) CreateTag(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockGitService) PushTag(_ context.Context, _ string) error {
	return nil
}

func (m *mockGitService) Push(_ context.Context) error {
	return nil
}

var _ registry.AIProviderFactory = (*mockAIFactory)(nil)
var _ vcsregistry.VCSProviderFactory = (*mockVCSFactory)(nil)
var _ ticketregistry.TicketProviderFactory = (*mockTicketFactory)(nil)
var _ ports.CommitSummarizer = (*mockCommitSummarizer)(nil)
var _ ports.PRSummarizer = (*mockPRSummarizer)(nil)
var _ ports.VCSClient = (*mockVCSClient)(nil)
var _ ports.GitService = (*mockGitService)(nil)
var _ ports.TickerManager = (*mockTicketManager)(nil)

func TestNewContainer(t *testing.T) {
	cfg := &config.Config{Language: "en"}
	trans := &i18n.Translations{}

	container := NewContainer(cfg, trans)

	require.NotNil(t, container)
	assert.Equal(t, cfg, container.config)
	assert.Equal(t, trans, container.translations)
	assert.NotNil(t, container.aiRegistry)
	assert.NotNil(t, container.vcsRegistry)
	assert.NotNil(t, container.ticketRegistry)
}

func TestRegisterAIProvider(t *testing.T) {
	cfg := &config.Config{Language: "en"}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	mockFactory := &mockAIFactory{}
	err := container.RegisterAIProvider("mock", mockFactory)
	require.NoError(t, err)

	err = container.RegisterAIProvider("mock", mockFactory)
	assert.Error(t, err)
}

func TestRegisterVCSProvider(t *testing.T) {
	cfg := &config.Config{Language: "en"}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	mockFactory := &mockVCSFactory{}
	err := container.RegisterVCSProvider("mock", mockFactory)
	require.NoError(t, err)

	err = container.RegisterVCSProvider("mock", mockFactory)
	assert.Error(t, err)
}

func TestRegisterTicketProvider(t *testing.T) {
	cfg := &config.Config{Language: "en"}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	mockFactory := &mockTicketFactory{}
	err := container.RegisterTicketProvider("mock", mockFactory)
	require.NoError(t, err)

	err = container.RegisterTicketProvider("mock", mockFactory)
	assert.Error(t, err)
}

func TestSetGitService_GetGitService(t *testing.T) {
	cfg := &config.Config{Language: "en"}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	mockGit := &mockGitService{}
	container.SetGitService(mockGit)

	retrieved := container.GetGitService()
	assert.Equal(t, mockGit, retrieved)
}

func TestGetCommitService_Success(t *testing.T) {
	cfg := &config.Config{
		Language: "en",
		AIConfig: config.AIConfig{
			ActiveAI: config.AIGemini,
		},
	}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	mockAIFactory := &mockAIFactory{}
	err := container.RegisterAIProvider("gemini", mockAIFactory)
	require.NoError(t, err)

	mockGit := &mockGitService{}
	container.SetGitService(mockGit)

	ctx := context.Background()
	service, err := container.GetCommitService(ctx)

	require.NoError(t, err)
	assert.NotNil(t, service)

	service2, err := container.GetCommitService(ctx)
	require.NoError(t, err)
	assert.Equal(t, service, service2)
}

func TestGetCommitService_WithoutGitService(t *testing.T) {
	cfg := &config.Config{Language: "en"}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	ctx := context.Background()
	service, err := container.GetCommitService(ctx)

	assert.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "servicio git no creado")
}

func TestGetCommitService_WithAICreationError(t *testing.T) {
	cfg := &config.Config{
		Language: "en",
		AIConfig: config.AIConfig{
			ActiveAI: config.AIGemini,
		},
	}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	mockAIFactory := &mockAIFactory{
		createCommitSummarizerError: errors.New("AI creation failed"),
	}
	err := container.RegisterAIProvider("gemini", mockAIFactory)
	require.NoError(t, err)

	mockGit := &mockGitService{}
	container.SetGitService(mockGit)

	ctx := context.Background()
	service, err := container.GetCommitService(ctx)

	require.NoError(t, err)
	assert.NotNil(t, service)
}

func TestGetCommitService_WithoutActiveAI(t *testing.T) {
	cfg := &config.Config{
		Language: "en",
		AIConfig: config.AIConfig{
			ActiveAI: "",
		},
	}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	mockGit := &mockGitService{}
	container.SetGitService(mockGit)

	ctx := context.Background()
	service, err := container.GetCommitService(ctx)

	require.NoError(t, err)
	assert.NotNil(t, service)
}

func TestGetPRSummarizer_Success(t *testing.T) {
	cfg := &config.Config{
		Language: "en",
		AIConfig: config.AIConfig{
			ActiveAI: config.AIGemini,
		},
	}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	mockAIFactory := &mockAIFactory{}
	err := container.RegisterAIProvider("gemini", mockAIFactory)
	require.NoError(t, err)

	ctx := context.Background()
	summarizer, err := container.GetPRSummarizer(ctx)

	require.NoError(t, err)
	assert.NotNil(t, summarizer)
}

func TestGetPRSummarizer_NoActiveAI(t *testing.T) {
	cfg := &config.Config{
		Language: "en",
		AIConfig: config.AIConfig{
			ActiveAI: "",
		},
	}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	ctx := context.Background()
	summarizer, err := container.GetPRSummarizer(ctx)

	assert.Error(t, err)
	assert.Nil(t, summarizer)
	assert.Contains(t, err.Error(), "no hay IA activa configurada")
}

func TestGetPRSummarizer_ProviderNotFound(t *testing.T) {
	cfg := &config.Config{
		Language: "en",
		AIConfig: config.AIConfig{
			ActiveAI: config.AIGemini,
		},
	}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	ctx := context.Background()
	summarizer, err := container.GetPRSummarizer(ctx)

	assert.Error(t, err)
	assert.Nil(t, summarizer)
	assert.Contains(t, err.Error(), "no encontrado")
}

func TestGetPRSummarizer_CreationError(t *testing.T) {
	cfg := &config.Config{
		Language: "en",
		AIConfig: config.AIConfig{
			ActiveAI: config.AIGemini,
		},
	}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	mockAIFactory := &mockAIFactory{
		createPRSummarizerError: errors.New("failed to create PR summarizer"),
	}
	err := container.RegisterAIProvider("gemini", mockAIFactory)
	require.NoError(t, err)

	ctx := context.Background()
	summarizer, err := container.GetPRSummarizer(ctx)

	assert.Error(t, err)
	assert.Nil(t, summarizer)
	assert.Contains(t, err.Error(), "error al crear el servicio de IA para PRs")
}

func TestGetPRService_Success(t *testing.T) {
	cfg := &config.Config{
		Language: "en",
		AIConfig: config.AIConfig{
			ActiveAI: config.AIGemini,
		},
		VCSConfigs: map[string]config.VCSConfig{
			"github": {
				Token: "test-token",
			},
		},
	}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	mockAIFactory := &mockAIFactory{}
	err := container.RegisterAIProvider("gemini", mockAIFactory)
	require.NoError(t, err)

	mockVCSFactory := &mockVCSFactory{}
	err = container.RegisterVCSProvider("github", mockVCSFactory)
	require.NoError(t, err)

	mockGit := &mockGitService{}
	container.SetGitService(mockGit)

	ctx := context.Background()
	service, err := container.GetPRService(ctx)

	require.NoError(t, err)
	assert.NotNil(t, service)

	service2, err := container.GetPRService(ctx)
	require.NoError(t, err)
	assert.Equal(t, service, service2)
}

func TestGetPRService_WithoutGitService(t *testing.T) {
	cfg := &config.Config{Language: "en"}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	ctx := context.Background()
	service, err := container.GetPRService(ctx)

	assert.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "servicio git no creado")
}

func TestGetPRService_WithPRSummarizerError(t *testing.T) {
	cfg := &config.Config{
		Language: "en",
		AIConfig: config.AIConfig{
			ActiveAI: "",
		},
		VCSConfigs: map[string]config.VCSConfig{
			"github": {
				Token: "test-token",
			},
		},
	}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	mockVCSFactory := &mockVCSFactory{}
	err := container.RegisterVCSProvider("github", mockVCSFactory)
	require.NoError(t, err)

	mockGit := &mockGitService{}
	container.SetGitService(mockGit)

	ctx := context.Background()
	service, err := container.GetPRService(ctx)

	require.NoError(t, err)
	assert.NotNil(t, service)
}

func TestGetPRService_VCSClientCreationError(t *testing.T) {
	cfg := &config.Config{
		Language: "en",
		VCSConfigs: map[string]config.VCSConfig{
			"github": {
				Token: "test-token",
			},
		},
	}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	mockGit := &mockGitService{}
	container.SetGitService(mockGit)

	ctx := context.Background()
	service, err := container.GetPRService(ctx)

	assert.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "error al crear cliente VCS")
}
