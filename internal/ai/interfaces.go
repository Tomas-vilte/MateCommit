package ai

import (
	"context"

	"github.com/thomas-vilte/matecommit/internal/models"
)

// CommitSummarizer is an interface that defines the service to generate commit suggestions.
type CommitSummarizer interface {
	// GenerateSuggestions generates a list of commit message suggestions.
	GenerateSuggestions(ctx context.Context, info models.CommitInfo, count int) ([]models.CommitSuggestion, error)
}

// PRSummarizer defines the interface for services that summarize Pull Requests.
type PRSummarizer interface {
	// GeneratePRSummary generates a summary of a Pull Request given a prompt.
	GeneratePRSummary(ctx context.Context, prompt string) (models.PRSummary, error)
}

// ReleaseNotesGenerator defines the interface to generate release notes.
type ReleaseNotesGenerator interface {
	GenerateNotes(ctx context.Context, release *models.Release) (*models.ReleaseNotes, error)
}

// IssueContentGenerator defines the interface to generate issue content with AI.
type IssueContentGenerator interface {
	// GenerateIssueContent generates the title, description, and labels of an issue
	// based on the context provided in the request.
	GenerateIssueContent(ctx context.Context, request models.IssueGenerationRequest) (*models.IssueGenerationResult, error)
}

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
