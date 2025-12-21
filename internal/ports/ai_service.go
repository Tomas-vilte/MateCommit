package ports

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
