package di

import (
	"context"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai/registry"
	ticketregistry "github.com/Tomas-vilte/MateCommit/internal/infrastructure/tickets/registry"
	vcsregistry "github.com/Tomas-vilte/MateCommit/internal/infrastructure/vcs/registry"
	"github.com/stretchr/testify/mock"
)

type mockAIFactory struct {
	mock.Mock
}

func (m *mockAIFactory) CreateCommitSummarizer(ctx context.Context, cfg *config.Config, trans *i18n.Translations) (ports.CommitSummarizer, error) {
	args := m.Called(ctx, cfg, trans)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ports.CommitSummarizer), args.Error(1)
}

func (m *mockAIFactory) CreatePRSummarizer(ctx context.Context, cfg *config.Config, trans *i18n.Translations) (ports.PRSummarizer, error) {
	args := m.Called(ctx, cfg, trans)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ports.PRSummarizer), args.Error(1)
}

func (m *mockAIFactory) CreateIssueContentGenerator(ctx context.Context, cfg *config.Config, trans *i18n.Translations) (ports.IssueContentGenerator, error) {
	args := m.Called(ctx, cfg, trans)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ports.IssueContentGenerator), args.Error(1)
}

func (m *mockAIFactory) ValidateConfig(cfg *config.Config) error {
	args := m.Called(cfg)
	return args.Error(0)
}

func (m *mockAIFactory) Name() string {
	return "mock"
}

type mockVCSFactory struct {
	mock.Mock
}

func (m *mockVCSFactory) CreateClient(ctx context.Context, owner, repo, token string, trans *i18n.Translations) (ports.VCSClient, error) {
	args := m.Called(ctx, owner, repo, token, trans)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ports.VCSClient), args.Error(1)
}

func (m *mockVCSFactory) ValidateConfig(cfg *config.VCSConfig) error {
	args := m.Called(cfg)
	return args.Error(0)
}

func (m *mockVCSFactory) Name() string {
	return "mock"
}

type mockTicketFactory struct {
	mock.Mock
}

func (m *mockTicketFactory) CreateClient(ctx context.Context, cfg config.TicketProviderConfig, trans *i18n.Translations) (ports.TickerManager, error) {
	args := m.Called(ctx, cfg, trans)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ports.TickerManager), args.Error(1)
}

func (m *mockTicketFactory) ValidateConfig(cfg config.TicketProviderConfig) error {
	args := m.Called(cfg)
	return args.Error(0)
}

func (m *mockTicketFactory) Name() string {
	return "mock"
}

func TestNewContainer(t *testing.T) {
	cfg := &config.Config{
		Language: "en",
	}
	trans := &i18n.Translations{}

	container := NewContainer(cfg, trans)

	if container == nil {
		t.Fatal("Container should not be nil")
	}

	if container.config != cfg {
		t.Error("Container config does not match input config")
	}

	if container.translations != trans {
		t.Error("Container translations do not match input translations")
	}

	if container.aiRegistry == nil {
		t.Error("AI registry should be initialized")
	}

	if container.vcsRegistry == nil {
		t.Error("VCS registry should be initialized")
	}

	if container.ticketRegistry == nil {
		t.Error("Ticket registry should be initialized")
	}
}

func TestRegisterAIProvider(t *testing.T) {
	cfg := &config.Config{Language: "en"}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	mockFactory := &mockAIFactory{}
	err := container.RegisterAIProvider("mock", mockFactory)

	if err != nil {
		t.Fatalf("Failed to register AI provider: %v", err)
	}

	err = container.RegisterAIProvider("mock", mockFactory)
	if err == nil {
		t.Error("Should not allow registering the same provider twice")
	}
}

func TestRegisterVCSProvider(t *testing.T) {
	cfg := &config.Config{Language: "en"}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	mockFactory := &mockVCSFactory{}
	err := container.RegisterVCSProvider("mock", mockFactory)

	if err != nil {
		t.Fatalf("Failed to register VCS provider: %v", err)
	}

	err = container.RegisterVCSProvider("mock", mockFactory)
	if err == nil {
		t.Error("Should not allow registering the same provider twice")
	}
}

func TestRegisterTicketProvider(t *testing.T) {
	cfg := &config.Config{Language: "en"}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	mockFactory := &mockTicketFactory{}
	err := container.RegisterTicketProvider("mock", mockFactory)

	if err != nil {
		t.Fatalf("Failed to register Ticket provider: %v", err)
	}

	err = container.RegisterTicketProvider("mock", mockFactory)
	if err == nil {
		t.Error("Should not allow registering the same provider twice")
	}
}

func TestGetAIRegistry(t *testing.T) {
	cfg := &config.Config{Language: "en"}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	aiRegistry := container.GetAIRegistry()
	if aiRegistry == nil {
		t.Error("AI aiRegistry should not be nil")
	}

	if aiRegistry != container.aiRegistry {
		t.Error("Returned aiRegistry should be the same as internal aiRegistry")
	}
}

func TestGetVCSRegistry(t *testing.T) {
	cfg := &config.Config{Language: "en"}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	vcsRegistry := container.GetVCSRegistry()
	if vcsRegistry == nil {
		t.Error("VCS vcsRegistry should not be nil")
	}

	if vcsRegistry != container.vcsRegistry {
		t.Error("Returned vcsRegistry should be the same as internal vcsRegistry")
	}
}

func TestGetTicketRegistry(t *testing.T) {
	cfg := &config.Config{Language: "en"}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	ticketRegistry := container.GetTicketRegistry()
	if ticketRegistry == nil {
		t.Error("Ticket ticketRegistry should not be nil")
	}

	if ticketRegistry != container.ticketRegistry {
		t.Error("Returned ticketRegistry should be the same as internal ticketRegistry")
	}
}

func TestGetConfig(t *testing.T) {
	cfg := &config.Config{Language: "en"}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	returnedCfg := container.GetConfig()
	if returnedCfg != cfg {
		t.Error("Returned config should be the same as input config")
	}
}

func TestGetTranslations(t *testing.T) {
	cfg := &config.Config{Language: "en"}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	returnedTrans := container.GetTranslations()
	if returnedTrans != trans {
		t.Error("Returned translations should be the same as input translations")
	}
}

func TestGetIssueTemplateService(t *testing.T) {
	cfg := &config.Config{Language: "en"}
	trans := &i18n.Translations{}
	container := NewContainer(cfg, trans)

	service := container.GetIssueTemplateService()
	if service == nil {
		t.Fatal("IssueTemplateService should not be nil")
	}

	if service != container.GetIssueTemplateService() {
		t.Error("Returned service should be a singleton")
	}
}

func TestGetIssueGeneratorService(t *testing.T) {
	t.Run("should fail if GitService is not set", func(t *testing.T) {
		cfg := &config.Config{Language: "en"}
		trans := &i18n.Translations{}
		container := NewContainer(cfg, trans)

		_, err := container.GetIssueGeneratorService(context.Background())
		if err == nil {
			t.Error("Expected error because GitService is nil")
		}
	})

	t.Run("should create IssueGeneratorService successfully", func(t *testing.T) {
		cfg := &config.Config{
			Language:          "en",
			ActiveVCSProvider: "mock",
			AIConfig: config.AIConfig{
				ActiveAI: "mock",
			},
			VCSConfigs: make(map[string]config.VCSConfig),
		}
		cfg.VCSConfigs["mock"] = config.VCSConfig{Provider: "mock"}

		trans, _ := i18n.NewTranslations("en", "")
		container := NewContainer(cfg, trans)

		aiFactory := &mockAIFactory{}
		vcsFactory := &mockVCSFactory{}

		// Set expectations - Return nil client/generator is fine for basic instantiation test
		aiFactory.On("CreateIssueContentGenerator", mock.Anything, cfg, trans).Return(nil, nil)
		vcsFactory.On("CreateClient", mock.Anything, "owner", "repo", "", trans).Return(nil, nil)
		vcsFactory.On("ValidateConfig", mock.Anything).Return(nil)

		_ = container.RegisterAIProvider("mock", aiFactory)
		_ = container.RegisterVCSProvider("mock", vcsFactory)

		mockGit := &mockGitService{}
		container.SetGitService(mockGit)

		service, err := container.GetIssueGeneratorService(context.Background())
		if err != nil {
			t.Fatalf("Failed to get IssueGeneratorService: %v", err)
		}

		if service == nil {
			t.Fatal("IssueGeneratorService should not be nil")
		}

		secondService, _ := container.GetIssueGeneratorService(context.Background())
		if service != secondService {
			t.Error("Returned service should be a singleton")
		}

		aiFactory.AssertExpectations(t)
		vcsFactory.AssertExpectations(t)
	})
}

type mockGitService struct {
	ports.GitService
}

func (m *mockGitService) GetRepoInfo(_ context.Context) (string, string, string, error) {
	return "owner", "repo", "mock", nil
}

var _ registry.AIProviderFactory = (*mockAIFactory)(nil)
var _ vcsregistry.VCSProviderFactory = (*mockVCSFactory)(nil)
var _ ticketregistry.TicketProviderFactory = (*mockTicketFactory)(nil)
