package models

type CriteriaStatus string

const (
	CriteriaFullyMet     CriteriaStatus = "full_met"
	CriteriaPartiallyMet CriteriaStatus = "partially_met"
	CriteriaNotMet       CriteriaStatus = "not_met"
)

type (
	CommitInfo struct {
		Files      []string
		Diff       string
		TicketInfo *TicketInfo
		IssueInfo  *Issue
	}

	GitChange struct {
		Path   string
		Status string
	}

	CommitSuggestion struct {
		CommitTitle          string               `json:"commit_title"`
		Explanation          string               `json:"explanation"`
		Files                []string             `json:"files"`
		CodeAnalysis         CodeAnalysis         `json:"code_analysis"`
		RequirementsAnalysis RequirementsAnalysis `json:"requirements_analysis"`
	}

	CodeAnalysis struct {
		ChangesOverview string `json:"changes_overview"`
		PrimaryPurpose  string `json:"primary_purpose"`
		TechnicalImpact string `json:"technical_impact"`
	}

	RequirementsAnalysis struct {
		CriteriaStatus         CriteriaStatus `json:"criteria_status"`
		MissingCriteria        []string       `json:"missing_criteria"`
		ImprovementSuggestions []string       `json:"improvement_suggestions"`
	}
)
