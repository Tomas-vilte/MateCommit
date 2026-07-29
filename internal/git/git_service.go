package git

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/regex"
	"golang.org/x/mod/semver"
)

type GitService struct {
	fallbackName  string
	fallbackEmail string
}

func NewGitService() *GitService {
	return &GitService{}
}

func (s *GitService) SetFallback(name, email string) {
	s.fallbackName = name
	s.fallbackEmail = email
}

// HasStagedChanges checks if there are changes in the staging area
func (s *GitService) HasStagedChanges(ctx context.Context) bool {
	log := logger.FromContext(ctx)

	repoRoot, err := s.GetRepoRoot(ctx)
	if err != nil {
		log.Error("failed to get repo root", "error", err)
		return false
	}

	cmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--quiet")
	cmd.Dir = repoRoot
	err = cmd.Run()

	hasChanges := err != nil && cmd.ProcessState.ExitCode() == 1

	statusCmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	statusCmd.Dir = repoRoot
	statusOutput, _ := statusCmd.Output()
	statusStr := strings.TrimSpace(string(statusOutput))

	log.Debug("checking staged changes",
		"repo_root", repoRoot,
		"has_staged_changes", hasChanges,
		"exit_code", cmd.ProcessState.ExitCode(),
		"git_status", statusStr)

	return hasChanges
}

func (s *GitService) GetChangedFiles(ctx context.Context) ([]string, error) {
	log := logger.FromContext(ctx)

	log.Debug("getting changed files")

	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		log.Error("git status failed",
			"error", err)
		return nil, errors.ErrGetChangedFiles.WithError(err)
	}

	changes := make([]string, 0)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if len(line) > 3 {
			path := strings.TrimSpace(line[3:])

			// Rename/copy entries are reported as "old -> new"; only the
			// new path exists on disk and is relevant for staging/diffing.
			if idx := strings.Index(path, " -> "); idx != -1 {
				path = path[idx+len(" -> "):]
			}

			if path == "" {
				continue
			}

			// A brand-new untracked directory is collapsed by git status
			// into a single entry ending in "/" instead of listing the
			// files inside it. Expand it so callers always see real files.
			if strings.HasSuffix(path, "/") {
				files, err := s.listUntrackedFilesIn(ctx, path)
				if err != nil {
					log.Warn("failed to expand untracked directory",
						"path", path,
						"error", err)
					continue
				}
				changes = append(changes, files...)
				continue
			}

			changes = append(changes, path)
		}
	}

	log.Debug("changed files retrieved",
		"count", len(changes))

	return changes, nil
}

// listUntrackedFilesIn lists the untracked files inside a directory that
// git status collapses into a single "dir/" entry.
func (s *GitService) listUntrackedFilesIn(ctx context.Context, dir string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-files", "--others", "--exclude-standard", "--", dir)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)
	for _, f := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if f != "" {
			files = append(files, f)
		}
	}
	return files, nil
}

func (s *GitService) GetDiff(ctx context.Context) (string, error) {
	log := logger.FromContext(ctx)

	log.Debug("executing git diff")

	stagedCmd := exec.CommandContext(ctx, "git", "diff", "--cached")
	stagedOutput, err := stagedCmd.Output()
	if err != nil {
		log.Error("git diff --cached failed",
			"error", err)
		return "", errors.ErrGetDiff.WithError(err).WithContext("diff_type", "staged")
	}

	unstagedCmd := exec.CommandContext(ctx, "git", "diff")
	unstageOutput, err := unstagedCmd.Output()
	if err != nil {
		log.Error("git diff failed",
			"error", err)
		return "", errors.ErrGetDiff.WithError(err).WithContext("diff_type", "unstaged")
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

		if combinedDiff == "" {
			log.Warn("no differences detected in repository")
			return "", errors.ErrNoDiff
		}
	}

	log.Debug("git diff completed",
		"staged_size", len(stagedOutput),
		"unstaged_size", len(unstageOutput),
		"total_size", len(combinedDiff))

	return combinedDiff, nil
}

func (s *GitService) GetDiffForFiles(ctx context.Context, files []string) (string, error) {
	log := logger.FromContext(ctx)

	if len(files) == 0 {
		return "", errors.ErrNoDiff
	}

	log.Debug("executing git diff for specific files", "count", len(files))

	argsStaged := append([]string{"diff", "--cached", "--"}, files...)
	stagedCmd := exec.CommandContext(ctx, "git", argsStaged...)
	stagedOutput, err := stagedCmd.Output()
	if err != nil {
		log.Error("git diff --cached failed for files", "error", err)
		return "", errors.ErrGetDiff.WithError(err).WithContext("diff_type", "staged_files")
	}

	argsUnstaged := append([]string{"diff", "--"}, files...)
	unstagedCmd := exec.CommandContext(ctx, "git", argsUnstaged...)
	unstageOutput, err := unstagedCmd.Output()
	if err != nil {
		log.Error("git diff failed for files", "error", err)
		return "", errors.ErrGetDiff.WithError(err).WithContext("diff_type", "unstaged_files")
	}

	combinedDiff := string(stagedOutput) + string(unstageOutput)
	
	if combinedDiff == "" {
		// Fallback check for untracked files in our selection
		untrackedCmd := exec.CommandContext(ctx, "git", "ls-files", "--others", "--exclude-standard")
		untrackedFiles, err := untrackedCmd.Output()
		if err == nil && len(untrackedFiles) > 0 {
			untrackedList := strings.Split(string(untrackedFiles), "\n")
			
			// Quick map for O(1) lookups
			selectedMap := make(map[string]bool)
			for _, f := range files {
				selectedMap[f] = true
			}

			for _, file := range untrackedList {
				if file != "" && selectedMap[file] {
					fileContentCmd := exec.CommandContext(ctx, "git", "show", ":"+file)
					content, err := fileContentCmd.Output()
					if err != nil {
						combinedDiff += "\n=== New file" + " " + file + "===\n"
						combinedDiff += string(content)
					}
				}
			}
		}
		
		if combinedDiff == "" {
			return "", errors.ErrNoDiff
		}
	}

	return combinedDiff, nil
}

func (s *GitService) CreateCommit(ctx context.Context, message string) error {
	log := logger.FromContext(ctx)

	if !s.HasStagedChanges(ctx) {
		log.Warn("no staged changes to commit")
		return errors.ErrNoChanges
	}

	repoRoot, err := s.GetRepoRoot(ctx)
	if err != nil {
		return err
	}

	log.Debug("creating git commit",
		"message_length", len(message))

	args := make([]string, 0, 6)
	if name, err := s.GetGitUserName(ctx); err == nil && name != "" {
		args = append(args, "-c", "user.name="+name)
	}
	if email, err := s.GetGitUserEmail(ctx); err == nil && email != "" {
		args = append(args, "-c", "user.email="+email)
	}
	args = append(args, "commit", "-m", message)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoRoot
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())

		log.Error("git commit failed",
			"error", err,
			"stderr", stderrStr)

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

		return errors.ErrCreateCommit.WithError(err).WithContext("stderr", stderrStr)
	}

	log.Info("git commit created successfully")

	return nil
}

func (s *GitService) AddFileToStaging(ctx context.Context, file string) error {
	log := logger.FromContext(ctx)

	repoRoot, err := s.GetRepoRoot(ctx)
	if err != nil {
		return err
	}

	absFile, err := filepath.Abs(file)
	if err != nil {
		log.Warn("failed to get absolute path of file, using original", "file", file, "error", err)
		absFile = file
	}

	log.Debug("adding file to staging",
		"file", file,
		"abs_file", absFile,
		"repo_root", repoRoot)

	cmd := exec.CommandContext(ctx, "git", "add", "--", absFile)
	cmd.Dir = repoRoot
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		log.Error("failed to add file to staging",
			"file", file,
			"abs_file", absFile,
			"error", err,
			"stderr", stderrStr)
		return errors.ErrAddFile.WithError(err).WithContext("file", file).WithContext("stderr", stderrStr)
	}

	statusCmd := exec.CommandContext(ctx, "git", "status", "--porcelain", "--", absFile)
	statusCmd.Dir = repoRoot
	statusOutput, _ := statusCmd.Output()
	log.Debug("git status after add",
		"file", file,
		"abs_file", absFile,
		"status", strings.TrimSpace(string(statusOutput)))

	log.Debug("file added to staging successfully",
		"file", file)
	return nil
}

func (s *GitService) GetCurrentBranch(ctx context.Context) (string, error) {
	log := logger.FromContext(ctx)

	log.Debug("getting current git branch")

	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		log.Error("failed to get current branch",
			"error", err)
		return "", errors.ErrGetBranch.WithError(err)
	}

	branchName := strings.TrimSpace(string(output))
	if branchName == "" {
		log.Warn("no branch name found (detached HEAD?)")
		return "", errors.ErrNoBranch
	}

	log.Debug("current branch retrieved",
		"branch", branchName)

	return branchName, nil
}

func (s *GitService) GetRepoInfo(ctx context.Context) (string, string, string, error) {
	log := logger.FromContext(ctx)

	log.Debug("getting repository info")

	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		log.Debug("failed to get remote URL",
			"error", err)
		return "", "", "", errors.ErrGetRepoURL.WithError(err)
	}

	url := strings.TrimSpace(string(output))
	owner, repo, provider, err := parseRepoURL(url)
	if err != nil {
		log.Error("failed to parse repository URL",
			"url", url,
			"error", err)
		return "", "", "", err
	}

	log.Debug("repository info retrieved",
		"owner", owner,
		"repo", repo,
		"provider", provider)

	return owner, repo, provider, nil
}

func (s *GitService) GetLastTag(ctx context.Context) (string, error) {
	log := logger.FromContext(ctx)

	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--tags", "origin", "refs/tags/*")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		tags := s.parseRemoteTags(string(output))
		if len(tags) > 0 {
			lastTag := s.getLatestSemverTag(tags)
			if lastTag != "" {
				log.Debug("last tag from remote", "tag", lastTag)
				return lastTag, nil
			}
		}
	}

	cmd = exec.CommandContext(ctx, "git", "describe", "--tags", "--abbrev=0", "--match", "v*")
	output, err = cmd.Output()
	if err != nil {
		return "", nil
	}

	tag := strings.TrimSpace(string(output))

	if tag != "" {
		if err := s.ValidateTagExists(ctx, tag); err != nil {
			log.Warn("tag not found in current branch", "tag", tag, "error", err)
			return s.getLastTagInCurrentBranch(ctx)
		}
	}
	return tag, nil
}

func (s *GitService) GetCommitsSinceTag(ctx context.Context, tag string) ([]models.Commit, error) {
	log := logger.FromContext(ctx)

	var args []string
	if tag == "" {
		args = []string{"log", "--pretty=format:%H|%an|%ae|%ad|%s|%b", "--no-merges", "--date=iso"}
	} else {
		if err := s.ValidateTagExists(ctx, tag); err != nil {
			log.Warn("tag not found, trying to fetch from remote", "tag", tag)
			_ = exec.CommandContext(ctx, "git", "fetch", "origin", "tag", tag).Run()
		}

		args = []string{"log", tag + "..HEAD", "--pretty=format:%H|%an|%ae|%ad|%s|%b", "--no-merges", "--date=iso"}
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		log.Warn("failed to get commits with tag range, trying alternative", "error", err)
		return s.getCommitsAlternative(ctx, tag)
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
		parts := strings.SplitN(line, "|", 6)
		if len(parts) >= 5 {
			commit := models.Commit{
				Hash:    parts[0],
				Author:  parts[1],
				Email:   parts[2],
				Date:    parts[3],
				Message: parts[4],
			}
			if len(parts) == 6 && parts[5] != "" {
				commit.Message = parts[4] + "\n" + parts[5]
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
		return nil, errors.ErrGetCommits.WithError(err).WithContext("from_tag", fromTag).WithContext("to_tag", toTag)
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
		return nil, errors.ErrGetRecentCommits.WithError(err).WithContext("count", count)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	return lines, nil
}

func (s *GitService) CreateTag(ctx context.Context, version, message string) error {
	cmd := exec.CommandContext(ctx, "git", "tag", "-a", version, "-m", message)
	if err := cmd.Run(); err != nil {
		return errors.ErrCreateTag.WithError(err).WithContext("version", version)
	}
	return nil
}

func (s *GitService) PushTag(ctx context.Context, version string) error {
	cmd := exec.CommandContext(ctx, "git", "push", "origin", version)
	if err := cmd.Run(); err != nil {
		return errors.ErrPushTag.WithError(err).WithContext("version", version)
	}
	return nil
}

// Push pushes commits to the remote repository
func (s *GitService) Push(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "push")
	if err := cmd.Run(); err != nil {
		return errors.ErrPush.WithError(err)
	}
	return nil
}

func (s *GitService) GetCommitCount(ctx context.Context) (int, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return 0, errors.ErrGetCommitCount.WithError(err)
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
		return "", errors.ErrGetTagDate.WithError(err).WithContext("tag", tag)
	}

	dateStr := strings.TrimSpace(string(output))
	if len(dateStr) >= 10 {
		return dateStr[:10], nil
	}

	return dateStr, nil
}

func (s *GitService) FetchTags(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Debug("fetching tags from remote")

	cmd := exec.CommandContext(ctx, "git", "fetch", "--tags", "origin")
	if err := cmd.Run(); err != nil {
		log.Warn("failed to fetch tags from remote", "error", err)
		return errors.ErrFetchTags.WithError(err)
	}

	log.Debug("tags fetched successfully")
	return nil
}

// ValidateGitConfig checks if git user.name and user.email are configured
func (s *GitService) ValidateGitConfig(ctx context.Context) error {
	log := logger.FromContext(ctx)
	repoRoot, err := s.GetRepoRoot(ctx)
	if err != nil {
		log.Error("failed to get repo root for config validation", "error", err)
		return errors.ErrNotInGitRepo
	}

	validate := func(key string, errType *errors.AppError, fallback string) error {
		cmd := exec.CommandContext(ctx, "git", "config", key)
		cmd.Dir = repoRoot
		output, err := cmd.Output()
		val := strings.TrimSpace(string(output))

		if err != nil || val == "" {
			if fallback != "" {
				log.Info("git config not set, using fallback", "key", key, "fallback", fallback)
				return nil
			}
			log.Error("git config check failed",
				"key", key,
				"error", err,
				"output", val,
				"repo_root", repoRoot)
			return errType
		}
		log.Debug("git config check passed", "key", key, "value", val)
		return nil
	}

	if err := validate("user.name", errors.ErrGitUserNotConfigured, s.fallbackName); err != nil {
		return err
	}
	if err := validate("user.email", errors.ErrGitEmailNotConfigured, s.fallbackEmail); err != nil {
		return err
	}

	return nil
}

// GetGitUserName returns the configured git user.name
func (s *GitService) GetGitUserName(ctx context.Context) (string, error) {
	log := logger.FromContext(ctx)
	repoRoot, _ := s.GetRepoRoot(ctx)

	cmd := exec.CommandContext(ctx, "git", "config", "user.name")
	if repoRoot != "" {
		cmd.Dir = repoRoot
	}
	output, err := cmd.Output()
	val := strings.TrimSpace(string(output))
	if (err != nil || val == "") && s.fallbackName != "" {
		log.Debug("using fallback git user.name", "name", s.fallbackName)
		return s.fallbackName, nil
	}
	if err != nil {
		log.Error("failed to get git user.name", "error", err, "repo_root", repoRoot)
		return "", errors.ErrGetGitUser.WithError(err).WithContext("config_key", "user.name")
	}
	return val, nil
}

// GetGitUserEmail returns the configured git user.email
func (s *GitService) GetGitUserEmail(ctx context.Context) (string, error) {
	log := logger.FromContext(ctx)
	repoRoot, _ := s.GetRepoRoot(ctx)

	cmd := exec.CommandContext(ctx, "git", "config", "user.email")
	if repoRoot != "" {
		cmd.Dir = repoRoot
	}
	output, err := cmd.Output()
	val := strings.TrimSpace(string(output))
	if (err != nil || val == "") && s.fallbackEmail != "" {
		log.Debug("using fallback git user.email", "email", s.fallbackEmail)
		return s.fallbackEmail, nil
	}
	if err != nil {
		log.Error("failed to get git user.email", "error", err, "repo_root", repoRoot)
		return "", errors.ErrGetGitUser.WithError(err).WithContext("config_key", "user.email")
	}
	return val, nil
}

func (s *GitService) ValidateTagExists(ctx context.Context, tag string) error {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", tag+"^{commit}")
	if err := cmd.Run(); err != nil {
		return errors.ErrValidateTag.WithError(err).WithContext("tag", tag)
	}
	return nil
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

	return "", "", "", errors.ErrExtractRepoInfo.WithContext("url", url)
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

// GetRepoRoot gets the absolute path to the root of the git repository
func (s *GitService) GetRepoRoot(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", errors.ErrGetRepoRoot.WithError(err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (s *GitService) parseRemoteTags(output string) []string {
	var tags []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			ref := parts[1]
			if strings.HasPrefix(ref, "refs/tags/") {
				tag := strings.TrimPrefix(ref, "refs/tags/")
				if regex.SemVer.MatchString(tag) {
					tags = append(tags, tag)
				}
			}
		}
	}
	return tags
}

func (s *GitService) getLatestSemverTag(tags []string) string {
	if len(tags) == 0 {
		return ""
	}

	sort.Slice(tags, func(i, j int) bool {
		return semver.Compare("v"+strings.TrimPrefix(tags[i], "v"), "v"+strings.TrimPrefix(tags[j], "v")) > 0
	})
	return tags[0]
}

func (s *GitService) getLastTagInCurrentBranch(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "--oneline", "--decorate", "--simplify-by-decoration", "--all", "--match", "v*", "-1")
	output, err := cmd.Output()
	if err != nil {
		return "", errors.ErrGetCommits.WithError(err).WithContext("operation", "get_last_tag_in_branch")
	}

	line := strings.TrimSpace(string(output))
	tagMatch := regexp.MustCompile(`\(tag:\s*(v[\d.]+)\)`).FindStringSubmatch(line)
	if len(tagMatch) > 1 {
		return tagMatch[1], nil
	}
	return "", nil
}

func (s *GitService) getCommitsAlternative(ctx context.Context, tag string) ([]models.Commit, error) {
	cmd := exec.CommandContext(ctx, "git", "merge-base", tag, "HEAD")
	mergeBase, err := cmd.Output()
	if err != nil {
		return nil, errors.ErrTagNotFound.WithError(err).WithContext("tag", tag)
	}

	baseHash := strings.TrimSpace(string(mergeBase))
	args := []string{"log", baseHash + "..HEAD", "--pretty=format:%H|%an|%ae|%ad|%s|%b", "--no-merges", "--date=iso"}

	cmd = exec.CommandContext(ctx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.ErrGetCommits.WithError(err).WithContext("tag", tag).WithContext("operation", "alternative")
	}
	var commits []models.Commit
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 6)
		if len(parts) >= 5 {
			commit := models.Commit{
				Hash:    parts[0],
				Author:  parts[1],
				Email:   parts[2],
				Date:    parts[3],
				Message: parts[4],
			}
			if len(parts) == 6 && parts[5] != "" {
				commit.Message = parts[4] + "\n" + parts[5]
			}
			commits = append(commits, commit)
		}
	}
	return commits, nil
}
