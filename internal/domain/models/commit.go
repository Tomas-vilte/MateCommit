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
		Title       string   `json:"Title"`
		Explanation string   `json:"Explanation"`
		Files       []string `json:"Files"`
	}
)
