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
		summarizer, err := gemini.NewGeminiCommitSummarizer(ctx, cfg, onConfirmation)
		if err != nil {
			return nil, err
		}
		return summarizer, nil
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
		summarizer, err := gemini.NewGeminiPRSummarizer(ctx, cfg, onConfirmation)
		if err != nil {
			return nil, err
		}
		return summarizer, nil
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
		generator, err := gemini.NewGeminiIssueContentGenerator(ctx, cfg, onConfirmation)
		if err != nil {
			return nil, err
		}
		return generator, nil
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
		generator, err := gemini.NewReleaseNotesGenerator(ctx, cfg, onConfirmation, owner, repo)
		if err != nil {
			return nil, err
		}
		return generator, nil
	default:
		return nil, fmt.Errorf("AI provider '%s' not supported", cfg.AIConfig.ActiveAI)
	}
}
