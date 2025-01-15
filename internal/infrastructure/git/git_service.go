package git

import (
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"os/exec"
	"strings"
)

type GitService struct {
}

func NewGitService() *GitService {
	return &GitService{}
}

// HasStagedChanges verifica si hay cambios en el área de staging
func (s *GitService) HasStagedChanges() bool {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	err := cmd.Run()

	// Si el comando retorna error (exit status 1), significa que hay cambios staged
	return err != nil && cmd.ProcessState.ExitCode() == 1
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
	stagedCmd := exec.Command("git", "diff", "--cached")
	stagedOutput, err := stagedCmd.Output()
	if err != nil {
		return "", err
	}

	unstagedCmd := exec.Command("git", "diff")
	unstageOutput, err := unstagedCmd.Output()
	if err != nil {
		return "", err
	}

	combinedDiff := string(stagedOutput) + string(unstageOutput)

	if combinedDiff == "" {
		untrackedCmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
		untrackedFiles, err := untrackedCmd.Output()
		if err == nil && len(untrackedFiles) > 0 {
			for _, file := range strings.Split(string(untrackedFiles), "\n") {
				if file != "" {
					fileContentCmd := exec.Command("git", "show", ":"+file)
					content, err := fileContentCmd.Output()
					if err != nil {
						combinedDiff += "\n=== Nuevo archivo" + " " + file + "===\n"
						combinedDiff += string(content)
					}
				}
			}
		}
	}
	return combinedDiff, nil
}

func (s *GitService) StageAllChanges() error {
	cmd := exec.Command("git", "add", ".")
	return cmd.Run()
}

func (s *GitService) CreateCommit(message string) error {
	// Primero verificamos si hay cambios staged
	if !s.HasStagedChanges() {
		return fmt.Errorf("no hay cambios en el área de staging")
	}

	// Creamos el commit
	cmd := exec.Command("git", "commit", "-m", message)
	return cmd.Run()
}
