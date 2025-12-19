package gemini

import (
	"context"

	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"google.golang.org/genai"
)

var _ ports.CostAwareAIProvider = (*GeminiProvider)(nil)

// GeminiProvider es una base compartida para todos los servicios de gemini que implementa la interfaz ports.CostAwareAIProvider
type GeminiProvider struct {
	Client *genai.Client
	model  string
}

// NewGeminiProvider crea una nueva instancia de GeminiProvider
func NewGeminiProvider(client *genai.Client, model string) *GeminiProvider {
	return &GeminiProvider{
		Client: client,
		model:  model,
	}
}

// CountTokens implementa ports.CostAwareAIProvider
func (g *GeminiProvider) CountTokens(ctx context.Context, prompt string) (int, error) {
	resp, err := g.Client.Models.CountTokens(ctx, g.model, genai.Text(prompt), nil)
	if err != nil {
		return 0, err
	}
	return int(resp.TotalTokens), nil
}

// GetModelName implementa ports.CostAwareAIProvider
func (g *GeminiProvider) GetModelName() string {
	return g.model
}

// GetProviderName implementa ports.CostAwareAIProvider
func (g *GeminiProvider) GetProviderName() string {
	return "gemini"
}
