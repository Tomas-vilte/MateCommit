package gemini

import (
	"context"

	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"google.golang.org/genai"
)

var _ ports.CostAwareAIProvider = (*GeminiProvider)(nil)

// GeminiProvider is a shared base for all Gemini services that implements the ports.CostAwareAIProvider interface
type GeminiProvider struct {
	Client *genai.Client
	model  string
}

// NewGeminiProvider creates a new instance of GeminiProvider
func NewGeminiProvider(client *genai.Client, model string) *GeminiProvider {
	return &GeminiProvider{
		Client: client,
		model:  model,
	}
}

// CountTokens implements ports.CostAwareAIProvider
func (g *GeminiProvider) CountTokens(ctx context.Context, prompt string) (int, error) {
	resp, err := g.Client.Models.CountTokens(ctx, g.model, genai.Text(prompt), nil)
	if err != nil {
		return 0, err
	}
	return int(resp.TotalTokens), nil
}

// GetModelName implements ports.CostAwareAIProvider
func (g *GeminiProvider) GetModelName() string {
	return g.model
}

// GetProviderName implements ports.CostAwareAIProvider
func (g *GeminiProvider) GetProviderName() string {
	return "gemini"
}
