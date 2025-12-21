package ports

import (
	"context"
)

// CostAwareAIProvider defines the interface for AI providers that support cost tracking.
type CostAwareAIProvider interface {
	// CountTokens counts the tokens of a prompt without making the actual model call.
	// This allows estimating the cost before executing the generation.
	CountTokens(ctx context.Context, prompt string) (int, error)

	// GetModelName returns the name of the current model (e.g.: "gemini-2.5-flash")
	GetModelName() string

	// GetProviderName returns the name of the provider (e.g.: "gemini", "openai", "anthropic")
	GetProviderName() string
}

// TokenCounter is a simpler interface for providers that only need to count tokens.
type TokenCounter interface {
	CountTokens(ctx context.Context, content string) (int, error)
}
