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
		CommitTitle string
		Explanation string
		Files       []string
	}
)
