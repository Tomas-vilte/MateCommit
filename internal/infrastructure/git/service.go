package git

import (
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"os/exec"
	"strings"
)

type GitService struct {
}

func NewGitService() *GitService {
	return &GitService{}
}

func (s *GitService) GetChangedFiles() ([]models.GitChange, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	changes := make([]models.GitChange, 0)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if len(line) > 3 {
			status := strings.TrimSpace(line[:2])
			path := strings.TrimSpace(line[3:])

			if path != "" {
				changes = append(changes, models.GitChange{
					Path:   path,
					Status: status,
				})
			}
		}
	}

	return changes, nil
}

func (s *GitService) GetDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--no-color")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (s *GitService) CreateCommit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	return cmd.Run()
}
