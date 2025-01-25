package models

type CriteriaStatus string

const (
	CriteriaFullyMet     CriteriaStatus = "full_met"
	CriteriaPartiallyMet CriteriaStatus = "partially_met"
	CriteriaNotMet       CriteriaStatus = "not_met"
)

type (
	CommitInfo struct {
		Files       []string
		Diff        string
		Format      string
		TicketTitle string
		TicketDesc  string
		Criteria    []string
	}

	GitChange struct {
		Path   string
		Status string
	}

	CommitSuggestion struct {
		CommitTitle            string
		Explanation            string
		Files                  []string
		CriteriaMsg            string
		CriteriaStatus         CriteriaStatus
		MissingCriteria        []string
		ImprovementSuggestions []string
	}
)
