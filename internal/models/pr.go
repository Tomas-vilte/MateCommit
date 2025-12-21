package models

type ProgressEventType string

const (
	ProgressIssuesDetected  ProgressEventType = "issues_detected"
	ProgressStatsCalculated ProgressEventType = "stats_calculated"
	ProgressIssuesClosing   ProgressEventType = "issues_closing"
	ProgressBreakingChanges ProgressEventType = "breaking_changes"
	ProgressTestPlan        ProgressEventType = "test_plan_generated"
	ProgressGeneric         ProgressEventType = "generic_info"
)

type ProgressEvent struct {
	Type    ProgressEventType
	Message string
	Data    map[string]interface{}
}

type (
	// PRData contains information extracted from a Pull Request.
	PRData struct {
		ID            int
		Title         string
		Creator       string
		Commits       []Commit
		Diff          string
		BranchName    string
		RelatedIssues []Issue
		Description   string
	}

	// Commit represents a commit included in the PR.
	Commit struct {
		Message string
	}

	// PRSummary is the generated summary for the PR, with title, body, and labels.
	PRSummary struct {
		Title  string
		Body   string
		Labels []string
		Usage  *TokenUsage
	}
)
