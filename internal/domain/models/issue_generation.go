package models

// IssueGenerationRequest contains the information needed to generate an issue.
// Supports multiple context sources: manual description, git diff, or both.
type IssueGenerationRequest struct {
	// Description is the manual description provided by the user (optional)
	Description string

	// Diff contains local git changes (optional)
	Diff string

	// ChangedFiles is the list of modified files (optional)
	ChangedFiles []string

	// Hint is additional context provided by the user to guide generation (optional)
	Hint string

	// Language is the language for content generation (e.g.: "es", "en")
	Language string
}

// IssueGenerationResult contains the result of an issue's content generation.
type IssueGenerationResult struct {
	// Title is the generated title for the issue
	Title string

	// Description is the full generated description for the issue
	Description string

	// Labels are the suggested labels for the issue
	Labels []string

	// Assignees are the suggested assignees for the issue
	Assignees []string

	// Usage contains metadata on token usage by the AI
	Usage *TokenUsage
}

// DiffAnalysis contains the structured analysis of the diff for label inference.
type DiffAnalysis struct {
	// HasGoFiles indicates if the diff includes .go files
	HasGoFiles bool

	// HasTestFiles indicates if the diff includes test files
	HasTestFiles bool

	// HasDocFiles indicates if the diff includes documentation files
	HasDocFiles bool

	// HasConfigFiles indicates if the diff includes configuration files
	HasConfigFiles bool

	// HasUIFiles indicates if the diff includes UI files (CSS, HTML, JSX, etc)
	HasUIFiles bool

	// Keywords contains keywords found in the diff (fix, feat, refactor, etc)
	Keywords map[string]bool
}
