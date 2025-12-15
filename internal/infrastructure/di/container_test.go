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
)

// Mock factories for testing
type mockAIFactory struct{}

func (m *mockAIFactory) CreateCommitSummarizer(_ context.Context, _ *config.Config, _ *i18n.Translations) (ports.CommitSummarizer, error) {
	return nil, nil
}

func (m *mockAIFactory) CreatePRSummarizer(_ context.Context, _ *config.Config, _ *i18n.Translations) (ports.PRSummarizer, error) {
	return nil, nil
}

func (m *mockAIFactory) ValidateConfig(_ *config.Config) error {
	return nil
}

func (m *mockAIFactory) Name() string {
	return "mock"
}

type mockVCSFactory struct{}

func (m *mockVCSFactory) CreateClient(_ context.Context, _, _, _ string, _ *i18n.Translations) (ports.VCSClient, error) {
	return nil, nil
}

func (m *mockVCSFactory) ValidateConfig(_ *config.VCSConfig) error {
	return nil
}

func (m *mockVCSFactory) Name() string {
	return "mock"
}

type mockTicketFactory struct{}

func (m *mockTicketFactory) CreateClient(_ context.Context, _ config.TicketProviderConfig, _ *i18n.Translations) (ports.TickerManager, error) {
	return nil, nil
}

func (m *mockTicketFactory) ValidateConfig(_ config.TicketProviderConfig) error {
	return nil
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

	// Try to register the same provider again (should fail)
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

	// Try to register the same provider again (should fail)
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

	// Try to register the same provider again (should fail)
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

	// Verify it's the same instance
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

	// Verify it's the same instance
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

	// Verify it's the same instance
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

// Verify interface implementations
var _ registry.AIProviderFactory = (*mockAIFactory)(nil)
var _ vcsregistry.VCSProviderFactory = (*mockVCSFactory)(nil)
var _ ticketregistry.TicketProviderFactory = (*mockTicketFactory)(nil)
