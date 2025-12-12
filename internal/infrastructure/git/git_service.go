package git

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
)

var _ ports.GitService = (*GitService)(nil)

type GitService struct {
	trans *i18n.Translations
}

func NewGitService(trans *i18n.Translations) *GitService {
	return &GitService{
		trans: trans,
	}
}

// HasStagedChanges verifica si hay cambios en el área de staging
func (s *GitService) HasStagedChanges(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--quiet")
	err := cmd.Run()

	// Si el comando retorna error (exit status 1), significa que hay cambios staged
	return err != nil && cmd.ProcessState.ExitCode() == 1
}

func (s *GitService) GetChangedFiles(ctx context.Context) ([]models.GitChange, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
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

func (s *GitService) GetDiff(ctx context.Context) (string, error) {
	stagedCmd := exec.CommandContext(ctx, "git", "diff", "--cached")
	stagedOutput, err := stagedCmd.Output()
	if err != nil {
		return "", err
	}

	unstagedCmd := exec.CommandContext(ctx, "git", "diff")
	unstageOutput, err := unstagedCmd.Output()
	if err != nil {
		return "", err
	}

	combinedDiff := string(stagedOutput) + string(unstageOutput)

	if combinedDiff == "" {
		untrackedCmd := exec.CommandContext(ctx, "git", "ls-files", "--others", "--exclude-standard")
		untrackedFiles, err := untrackedCmd.Output()
		if err == nil && len(untrackedFiles) > 0 {
			for _, file := range strings.Split(string(untrackedFiles), "\n") {
				if file != "" {
					fileContentCmd := exec.CommandContext(ctx, "git", "show", ":"+file)
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

func (s *GitService) CreateCommit(ctx context.Context, message string) error {
	if !s.HasStagedChanges(ctx) {
		msg := s.trans.GetMessage("git.no_staged_changes", 0, nil)
		return fmt.Errorf("%s", msg)
	}

	cmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
	return cmd.Run()
}

func (s *GitService) AddFileToStaging(ctx context.Context, file string) error {
	repoRoot, err := s.getRepoRoot(ctx)
	if err != nil {
		msg := s.trans.GetMessage("git.get_repo_root", 0, map[string]interface{}{
			"Error": err,
		})
		return fmt.Errorf("%s", msg)
	}

	cmd := exec.CommandContext(ctx, "git", "add", "--", file)
	cmd.Dir = repoRoot
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fullErr := fmt.Sprintf("%v → %s", err, strings.TrimSpace(stderr.String()))
		msg := s.trans.GetMessage("git.add_file", 0, map[string]interface{}{
			"File":  file,
			"Error": fullErr,
		})
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func (s *GitService) GetCurrentBranch(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		msg := s.trans.GetMessage("git.get_branch_name", 0, map[string]interface{}{
			"Error": err,
		})
		return "", fmt.Errorf("%s", msg)
	}

	branchName := strings.TrimSpace(string(output))
	if branchName == "" {
		msg := s.trans.GetMessage("git.branch_not_detected", 0, nil)
		return "", fmt.Errorf("%s", msg)
	}

	return branchName, nil
}

func (s *GitService) GetRepoInfo(ctx context.Context) (string, string, string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		msg := s.trans.GetMessage("git.get_repo_url", 0, map[string]interface{}{
			"Error": err,
		})
		return "", "", "", fmt.Errorf("%s", msg)
	}

	url := strings.TrimSpace(string(output))
	return parseRepoURL(url, s.trans)
}

func (s *GitService) GetLastTag(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "describe", "--tags", "--abbrev=0")
	output, err := cmd.Output()
	if err != nil {
		// no hay tags
		return "", nil
	}
	return strings.TrimSpace(string(output)), nil
}

func (s *GitService) GetCommitsSinceTag(ctx context.Context, tag string) ([]models.Commit, error) {
	var args []string
	if tag == "" {
		// si no hay tag anterior, obtener todos los commits
		args = []string{"log", "--pretty=format:%H|%s|%b", "--no-merges"}
	} else {
		args = []string{"log", tag + "..HEAD", "--pretty=format:%H|%s|%b", "--no-merges"}
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		msg := s.trans.GetMessage("git.get_commits", 0, map[string]interface{}{
			"Error": err,
		})
		return nil, fmt.Errorf("%s", msg)
	}

	if len(output) == 0 {
		return []models.Commit{}, nil
	}

	var commits []models.Commit
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) >= 2 {
			commit := models.Commit{
				Message: parts[1],
			}
			if len(parts) == 3 {
				commit.Message = parts[1] + "\n" + parts[2]
			}
			commits = append(commits, commit)
		}
	}
	return commits, nil
}

func (s *GitService) GetRecentCommitMessages(ctx context.Context, count int) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "log", fmt.Sprintf("-%d", count), "--pretty=format:%s %b")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (s *GitService) CreateTag(ctx context.Context, version, message string) error {
	cmd := exec.CommandContext(ctx, "git", "tag", "-a", version, "-m", message)
	return cmd.Run()
}

func (s *GitService) PushTag(ctx context.Context, version string) error {
	cmd := exec.CommandContext(ctx, "git", "push", "origin", version)
	return cmd.Run()
}

func (s *GitService) GetCommitCount(ctx context.Context) (int, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	count := 0
	_, _ = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &count)
	return count, nil
}

func parseRepoURL(url string, trans *i18n.Translations) (string, string, string, error) {
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

	msg := trans.GetMessage("git.extract_repo_info", 0, map[string]interface{}{
		"Url": url,
	})
	return "", "", "", fmt.Errorf("%s", msg)
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

// getRepoRoot obtiene la ruta absoluta de la raíz del repositorio git
func (s *GitService) getRepoRoot(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		msg := s.trans.GetMessage("git.get_repo_root", 0, map[string]interface{}{
			"Error": err,
		})
		return "", fmt.Errorf("%s", msg)
	}
	return strings.TrimSpace(string(output)), nil
}
