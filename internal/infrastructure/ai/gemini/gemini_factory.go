package gemini

import (
	"context"
	"fmt"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
)

// GeminiProviderFactory implementa AIProviderFactory para Gemini
type GeminiProviderFactory struct{}

// NewGeminiProviderFactory crea una nueva factory para Gemini
func NewGeminiProviderFactory() *GeminiProviderFactory {
	return &GeminiProviderFactory{}
}

// CreateCommitSummarizer crea un servicio Gemini para sugerencias de commits
func (f *GeminiProviderFactory) CreateCommitSummarizer(
	ctx context.Context,
	cfg *config.Config,
	trans *i18n.Translations,
) (ports.CommitSummarizer, error) {
	return NewGeminiService(ctx, cfg, trans)
}

// CreatePRSummarizer crea un servicio Gemini para resumir PRs
func (f *GeminiProviderFactory) CreatePRSummarizer(
	ctx context.Context,
	cfg *config.Config,
	trans *i18n.Translations,
) (ports.PRSummarizer, error) {
	return NewGeminiPRSummarizer(ctx, cfg, trans)
}

// ValidateConfig valida la configuración de Gemini
func (f *GeminiProviderFactory) ValidateConfig(cfg *config.Config) error {
	providerCfg, exists := cfg.AIProviders["gemini"]
	if !exists {
		return fmt.Errorf("configuracion de gemini no encontrada")
	}

	if providerCfg.APIKey == "" {
		return fmt.Errorf("gemini API key es requerida")
	}

	return nil
}

// Name retorna el nombre del proveedor
func (f *GeminiProviderFactory) Name() string {
	return "gemini"
}
