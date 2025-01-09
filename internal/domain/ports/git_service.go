package ports

type GitService interface {
	GetChangedFiles() ([]string, error)
	GetDiff() (string, error)
	CreateCommit(message string) error
}
