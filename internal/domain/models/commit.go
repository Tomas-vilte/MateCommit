package models

type (
	CommitInfo struct {
		Files      []string
		Diff       string
		CommitType string // puede se feat, fix, docs, etc
		Message    string
	}
	
	GitChange struct {
		Path   string
		Status string
	}
)
