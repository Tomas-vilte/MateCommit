package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/regex"
)

type GitService struct{}

func NewGitService() *GitService {
	return &GitService{}
}

// HasStagedChanges checks if there are changes in the staging area
func (s *GitService) HasStagedChanges(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--quiet")
	err := cmd.Run()

	// If the command returns an error (exit status 1), it means there are staged changes
	return err != nil && cmd.ProcessState.ExitCode() == 1
}

func (s *GitService) GetChangedFiles(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	changes := make([]string, 0)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if len(line) > 3 {
			path := strings.TrimSpace(line[3:])

			if path != "" {
				changes = append(changes, path)
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
						combinedDiff += "\n=== New file" + " " + file + "===\n"
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
		return errors.ErrNoChanges
	}

	cmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())

		if strings.Contains(stderrStr, "Please tell me who you are") ||
			(strings.Contains(stderrStr, "user.name")) &&
				strings.Contains(stderrStr, "user.email") {
			return errors.ErrGitUserNotConfigured
		}
		if strings.Contains(stderrStr, "user.name") {
			return errors.ErrGitUserNotConfigured
		}
		if strings.Contains(stderrStr, "user.email") {
			return errors.ErrGitEmailNotConfigured
		}

		fullErr := fmt.Sprintf("%v: %s", err, stderrStr)
		return fmt.Errorf("%w: %s", errors.ErrCreateCommit, fullErr)
	}
	return nil
}

func (s *GitService) AddFileToStaging(ctx context.Context, file string) error {
	repoRoot, err := s.getRepoRoot(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", errors.ErrGetRepoRoot, err)
	}

	cmd := exec.CommandContext(ctx, "git", "add", "-A", "--", file)
	cmd.Dir = repoRoot
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fullErr := fmt.Sprintf("%v: %s", err, strings.TrimSpace(stderr.String()))
		return fmt.Errorf("%w [%s]: %s", errors.ErrAddFile, file, fullErr)
	}
	return nil
}

func (s *GitService) GetCurrentBranch(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%w: %v", errors.ErrGetBranch, err)
	}

	branchName := strings.TrimSpace(string(output))
	if branchName == "" {
		return "", errors.ErrNoBranch
	}

	return branchName, nil
}

func (s *GitService) GetRepoInfo(ctx context.Context) (string, string, string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", "", "", fmt.Errorf("%w: %v", errors.ErrGetRepoURL, err)
	}

	url := strings.TrimSpace(string(output))
	return parseRepoURL(url)
}

func (s *GitService) GetLastTag(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "describe", "--tags", "--abbrev=0")
	output, err := cmd.Output()
	if err != nil {
		// no tags found
		return "", nil
	}
	return strings.TrimSpace(string(output)), nil
}

func (s *GitService) GetCommitsSinceTag(ctx context.Context, tag string) ([]models.Commit, error) {
	var args []string
	if tag == "" {
		// if no previous tag exists, get all commits
		args = []string{"log", "--pretty=format:%H|%s|%b", "--no-merges"}
	} else {
		args = []string{"log", tag + "..HEAD", "--pretty=format:%H|%s|%b", "--no-merges"}
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errors.ErrGetCommits, err)
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

func (s *GitService) GetCommitsBetweenTags(ctx context.Context, fromTag, toTag string) ([]models.Commit, error) {
	rangeSpec := toTag
	if fromTag != "" {
		rangeSpec = fromTag + ".." + toTag
	}

	args := []string{"log", rangeSpec, "--pretty=format:%H|%s|%b", "--no-merges"}
	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errors.ErrGetCommits, err)
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

func (s *GitService) GetRecentCommitMessages(ctx context.Context, count int) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "log", fmt.Sprintf("-%d", count), "--pretty=format:%s %b")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	return lines, nil
}

func (s *GitService) CreateTag(ctx context.Context, version, message string) error {
	cmd := exec.CommandContext(ctx, "git", "tag", "-a", version, "-m", message)
	return cmd.Run()
}

func (s *GitService) PushTag(ctx context.Context, version string) error {
	cmd := exec.CommandContext(ctx, "git", "push", "origin", version)
	return cmd.Run()
}

// Push pushes commits to the remote repository
func (s *GitService) Push(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "push")
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

// GetTagDate returns the creation date of a tag in YYYY-MM-DD format
func (s *GitService) GetTagDate(ctx context.Context, tag string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--format=%ai", tag)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error getting tag date: %w", err)
	}

	dateStr := strings.TrimSpace(string(output))
	if len(dateStr) >= 10 {
		return dateStr[:10], nil // Return YYYY-MM-DD
	}

	return dateStr, nil
}

// ValidateGitConfig checks if git user.name and user.email are configured
func (s *GitService) ValidateGitConfig(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		return errors.ErrNotInGitRepo
	}

	cmd = exec.CommandContext(ctx, "git", "config", "user.name")
	output, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(output)) == "" {
		return errors.ErrGitUserNotConfigured
	}

	cmd = exec.CommandContext(ctx, "git", "config", "user.email")
	output, err = cmd.Output()
	if err != nil || strings.TrimSpace(string(output)) == "" {
		return errors.ErrGitEmailNotConfigured
	}
	return nil
}

// GetGitUserName returns the configured git user.name
func (s *GitService) GetGitUserName(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "config", "user.name")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetGitUserEmail returns the configured git user.email
func (s *GitService) GetGitUserEmail(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "config", "user.email")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func parseRepoURL(url string) (string, string, string, error) {
	var matches []string
	if regex.SSHRepo.MatchString(url) {
		matches = regex.SSHRepo.FindStringSubmatch(url)
	} else if regex.HTTPSRepo.MatchString(url) {
		matches = regex.HTTPSRepo.FindStringSubmatch(url)
	}

	if len(matches) >= 4 {
		provider := detectProvider(matches[1])
		repoName := strings.TrimSuffix(matches[3], ".git")
		return matches[2], repoName, provider, nil
	}

	return "", "", "", fmt.Errorf("%w [%s]", errors.ErrExtractRepoInfo, url)
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

// getRepoRoot gets the absolute path to the root of the git repository
func (s *GitService) getRepoRoot(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%w: %v", errors.ErrGetRepoRoot, err)
	}
	return strings.TrimSpace(string(output)), nil
}
