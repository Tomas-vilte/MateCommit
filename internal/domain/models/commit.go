package models

type (
	CommitInfo struct {
		Files  []string
		Diff   string
		Format string
	}

	GitChange struct {
		Path   string
		Status string
	}

	CommitSuggestion struct {
		Message     string
		Type        string
		Scope       string
		Description string
		Score       float64
	}
)
