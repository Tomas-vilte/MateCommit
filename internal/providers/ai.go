package providers

import (
	"context"
	"fmt"

	"github.com/thomas-vilte/matecommit/internal/ai"
	"github.com/thomas-vilte/matecommit/internal/ai/gemini"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/ports"
)

// NewCommitSummarizer creates a CommitSummarizer based on the configured provider
func NewCommitSummarizer(ctx context.Context, cfg *config.Config, onConfirmation ai.ConfirmationCallback) (ports.CommitSummarizer, error) {
	if cfg.AIConfig.ActiveAI == "" {
		return nil, fmt.Errorf("no AI provider configured")
	}

	switch cfg.AIConfig.ActiveAI {
	case "gemini":
		return gemini.NewGeminiCommitSummarizer(ctx, cfg, onConfirmation)
	default:
		return nil, fmt.Errorf("AI provider '%s' not supported", cfg.AIConfig.ActiveAI)
	}
}

// NewPRSummarizer creates a PRSummarizer based on the configured provider
func NewPRSummarizer(ctx context.Context, cfg *config.Config, onConfirmation ai.ConfirmationCallback) (ports.PRSummarizer, error) {
	if cfg.AIConfig.ActiveAI == "" {
		return nil, fmt.Errorf("no AI provider configured")
	}

	switch cfg.AIConfig.ActiveAI {
	case "gemini":
		return gemini.NewGeminiPRSummarizer(ctx, cfg, onConfirmation)
	default:
		return nil, fmt.Errorf("AI provider '%s' not supported", cfg.AIConfig.ActiveAI)
	}
}

// NewIssueContentGenerator creates an IssueContentGenerator based on the configured provider
func NewIssueContentGenerator(ctx context.Context, cfg *config.Config, onConfirmation ai.ConfirmationCallback) (ports.IssueContentGenerator, error) {
	if cfg.AIConfig.ActiveAI == "" {
		return nil, fmt.Errorf("no AI provider configured")
	}

	switch cfg.AIConfig.ActiveAI {
	case "gemini":
		return gemini.NewGeminiIssueContentGenerator(ctx, cfg, onConfirmation)
	default:
		return nil, fmt.Errorf("AI provider '%s' not supported", cfg.AIConfig.ActiveAI)
	}
}

// NewReleaseNotesGenerator creates a ReleaseNotesGenerator based on the configured provider
// Note: requires owner and repo which must be obtained from the git service first
func NewReleaseNotesGenerator(ctx context.Context, cfg *config.Config, onConfirmation ai.ConfirmationCallback, owner, repo string) (ports.ReleaseNotesGenerator, error) {
	if cfg.AIConfig.ActiveAI == "" {
		return nil, fmt.Errorf("no AI provider configured")
	}

	switch cfg.AIConfig.ActiveAI {
	case "gemini":
		return gemini.NewReleaseNotesGenerator(ctx, cfg, onConfirmation, owner, repo)
	default:
		return nil, fmt.Errorf("AI provider '%s' not supported", cfg.AIConfig.ActiveAI)
	}
}
