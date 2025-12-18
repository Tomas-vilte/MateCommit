package registry

import (
	"context"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
)

// MockTicketProviderFactory for testing
type MockTicketProviderFactory struct {
	name string
}

func (m *MockTicketProviderFactory) CreateClient(_ context.Context, _ config.TicketProviderConfig, _ *i18n.Translations) (ports.TickerManager, error) {
	return nil, nil
}

func (m *MockTicketProviderFactory) ValidateConfig(_ config.TicketProviderConfig) error {
	return nil
}

func (m *MockTicketProviderFactory) Name() string {
	return m.name
}

func TestNewTicketProviderRegistry(t *testing.T) {
	registry := NewTicketProviderRegistry()

	if registry == nil {
		t.Fatal("Registry should not be nil")
	}

	if registry.factories == nil {
		t.Error("Factories map should be initialized")
	}
}

func TestRegisterTicketProvider(t *testing.T) {
	registry := NewTicketProviderRegistry()
	factory := &MockTicketProviderFactory{name: "mock"}

	err := registry.Register("mock", factory)
	if err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}

	// Try to register the same provider again
	err = registry.Register("mock", factory)
	if err == nil {
		t.Error("Should not allow registering the same provider twice")
	}
}

func TestGetTicketProvider(t *testing.T) {
	registry := NewTicketProviderRegistry()
	factory := &MockTicketProviderFactory{name: "mock"}

	_ = registry.Register("mock", factory)

	retrievedFactory, err := registry.Get("mock")
	if err != nil {
		t.Fatalf("Failed to get registered provider: %v", err)
	}

	if retrievedFactory == nil {
		t.Error("Retrieved factory should not be nil")
	}

	if retrievedFactory.Name() != "mock" {
		t.Errorf("Expected factory name 'mock', got '%s'", retrievedFactory.Name())
	}
}

func TestGetTicketProviderNotFound(t *testing.T) {
	registry := NewTicketProviderRegistry()

	_, err := registry.Get("nonexistent")
	if err == nil {
		t.Error("Should return error for non-existent provider")
	}
}

func TestListTicketProviders(t *testing.T) {
	registry := NewTicketProviderRegistry()

	// Empty registry
	list := registry.List()
	if len(list) != 0 {
		t.Error("List should be empty initially")
	}

	// Register multiple providers
	_ = registry.Register("jira", &MockTicketProviderFactory{name: "jira"})
	_ = registry.Register("linear", &MockTicketProviderFactory{name: "linear"})

	list = registry.List()
	if len(list) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(list))
	}

	// Verify providers are in the list
	found := make(map[string]bool)
	for _, name := range list {
		found[name] = true
	}

	if !found["jira"] || !found["linear"] {
		t.Error("Expected providers not found in list")
	}
}

func TestIsRegisteredTicketProvider(t *testing.T) {
	registry := NewTicketProviderRegistry()

	if registry.IsRegistered("mock") {
		t.Error("Provider should not be registered initially")
	}

	_ = registry.Register("mock", &MockTicketProviderFactory{name: "mock"})

	if !registry.IsRegistered("mock") {
		t.Error("Provider should be registered after Register call")
	}

	if registry.IsRegistered("nonexistent") {
		t.Error("Nonexistent provider should not be registered")
	}
}

func TestCreateClientFromConfig(t *testing.T) {
	registry := NewTicketProviderRegistry()
	factory := &MockTicketProviderFactory{name: "jira"}
	_ = registry.Register("jira", factory)

	cfg := &config.Config{
		ActiveTicketService: "jira",
		UseTicket:           true,
		TicketProviders: map[string]config.TicketProviderConfig{
			"jira": {
				APIKey:  "test-key",
				BaseURL: "https://test.atlassian.net",
				Email:   "test@example.com",
			},
		},
	}

	trans := &i18n.Translations{}
	ctx := context.Background()

	client, err := registry.CreateClientFromConfig(ctx, cfg, trans)
	if err != nil {
		t.Fatalf("Failed to create client from config: %v", err)
	}

	// Mock returns nil, which is acceptable
	if client != nil {
		t.Error("Mock should return nil client")
	}
}

func TestCreateClientFromConfigNoActiveService(t *testing.T) {
	registry := NewTicketProviderRegistry()

	cfg := &config.Config{
		ActiveTicketService: "",
		UseTicket:           false,
	}

	trans := &i18n.Translations{}
	ctx := context.Background()

	client, err := registry.CreateClientFromConfig(ctx, cfg, trans)
	if err != nil {
		t.Fatalf("Should not error when no active service: %v", err)
	}

	if client != nil {
		t.Error("Should return nil client when no active service")
	}
}

func TestCreateClientFromConfigProviderNotFound(t *testing.T) {
	registry := NewTicketProviderRegistry()

	cfg := &config.Config{
		ActiveTicketService: "nonexistent",
		UseTicket:           true,
		TicketProviders: map[string]config.TicketProviderConfig{
			"nonexistent": {
				APIKey: "test-key",
			},
		},
	}

	trans := &i18n.Translations{}
	ctx := context.Background()

	_, err := registry.CreateClientFromConfig(ctx, cfg, trans)
	if err == nil {
		t.Error("Should error when provider is not registered")
	}
}

func TestCreateClientFromConfigNoConfig(t *testing.T) {
	registry := NewTicketProviderRegistry()
	factory := &MockTicketProviderFactory{name: "jira"}
	_ = registry.Register("jira", factory)

	cfg := &config.Config{
		ActiveTicketService: "jira",
		UseTicket:           true,
		TicketProviders:     map[string]config.TicketProviderConfig{}, // Empty config
	}

	trans := &i18n.Translations{}
	ctx := context.Background()

	_, err := registry.CreateClientFromConfig(ctx, cfg, trans)
	if err == nil {
		t.Error("Should error when provider config is missing")
	}
}

// Verify interface implementation
var _ TicketProviderFactory = (*MockTicketProviderFactory)(nil)
