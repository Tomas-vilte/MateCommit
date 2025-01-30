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
	}

	GitChange struct {
		Path   string
		Status string
	}

	CommitSuggestion struct {
		CommitTitle          string
		Explanation          string
		Files                []string
		CodeAnalysis         CodeAnalysis
		RequirementsAnalysis RequirementsAnalysis
	}

	CodeAnalysis struct {
		ChangesOverview string
		PrimaryPurpose  string
		TechnicalImpact string
	}

	RequirementsAnalysis struct {
		CriteriaStatus         CriteriaStatus
		MissingCriteria        []string
		ImprovementSuggestions []string
	}
)
