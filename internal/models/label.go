package models

// RepoLabel is a label defined in the repository, with its description as
// configured on GitHub (empty if the label has none set).
type RepoLabel struct {
	Name        string
	Description string
}
