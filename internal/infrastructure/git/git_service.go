package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
)

var _ ports.GitService = (*GitService)(nil)

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

func (s *GitService) CreateCommit(message string) error {
	// Primero verificamos si hay cambios staged
	if !s.HasStagedChanges() {
		return fmt.Errorf("no hay cambios en el área de staging")
	}

	// Creamos el commit
	cmd := exec.Command("git", "commit", "-m", message)
	return cmd.Run()
}

func (s *GitService) AddFileToStaging(file string) error {
	cmd := exec.Command("git", "add", "--all", "--", file)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error al agregar '%s': %v → %s", file, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (s *GitService) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error al obtener el nombre de la branch: %v", err)
	}

	branchName := strings.TrimSpace(string(output))
	if branchName == "" {
		return "", fmt.Errorf("no se pudo detectar el nombre de la branch")
	}

	return branchName, nil
}

func (s *GitService) GetRepoInfo() (string, string, string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", "", "", fmt.Errorf("error al obtener la URL del repositorio: %w", err)
	}

	url := strings.TrimSpace(string(output))
	return parseRepoURL(url)
}

func parseRepoURL(url string) (string, string, string, error) {
	sshRegex := regexp.MustCompile(`git@([^:]+):([^/]+)/(.+)\.git$`)
	httpsRegex := regexp.MustCompile(`https://([^/]+)/([^/]+)/(.+?)(?:\.git)?$`)

	var matches []string
	if sshRegex.MatchString(url) {
		matches = sshRegex.FindStringSubmatch(url)
	} else if httpsRegex.MatchString(url) {
		matches = httpsRegex.FindStringSubmatch(url)
	}

	if len(matches) >= 4 {
		provider := detectProvider(matches[1])
		repoName := strings.TrimSuffix(matches[3], ".git")
		return matches[2], repoName, provider, nil
	}

	return "", "", "", fmt.Errorf("no se pudo extraer el propietario y el repositorio de la URL: %s", url)
}

func detectProvider(host string) string {
	if strings.Contains(host, "github") {
		return "github"
	}
	if strings.Contains(host, "gitlab") {
		return "gitlab"
	}
	return "unknown"
}
