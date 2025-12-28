package errors

import "fmt"

// ErrorType defines the category of the error
type ErrorType string

const (
	TypeConfiguration ErrorType = "CONFIGURATION"
	TypeAI            ErrorType = "AI"
	TypeVCS           ErrorType = "VCS"
	TypeGit           ErrorType = "GIT"
	TypeInternal      ErrorType = "INTERNAL"
	TypeUpdate        ErrorType = "UPDATE"
)

// AppError represents a domain-level error with a type and an underlying error
type AppError struct {
	Type       ErrorType
	Message    string
	Context    map[string]interface{}
	Err        error
	Suggestion string
}

func (e *AppError) Error() string {
	var msg string
	if e.Err != nil {
		msg = fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Err)
	} else {
		msg = fmt.Sprintf("%s: %s", e.Type, e.Message)
	}

	if e.Context != nil {
		if stderr, ok := e.Context["stderr"].(string); ok && stderr != "" {
			msg += fmt.Sprintf(" - %s", stderr)
		}
	}

	return msg
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// WithError creates a new AppError with an underlying error
func (e *AppError) WithError(err error) *AppError {
	return &AppError{
		Type:       e.Type,
		Message:    e.Message,
		Context:    e.Context,
		Err:        err,
		Suggestion: e.Suggestion,
	}
}

// WithContext creates a new AppError with additional context
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	ctx := make(map[string]interface{})
	for k, v := range e.Context {
		ctx[k] = v
	}
	ctx[key] = value
	return &AppError{
		Type:       e.Type,
		Message:    e.Message,
		Context:    ctx,
		Err:        e.Err,
		Suggestion: e.Suggestion,
	}
}

func (e *AppError) WithSuggestion(suggestion string) *AppError {
	return &AppError{
		Type:       e.Type,
		Message:    e.Message,
		Context:    e.Context,
		Err:        e.Err,
		Suggestion: suggestion,
	}
}

// NewAppError creates a new AppError
func NewAppError(t ErrorType, msg string, err error) *AppError {
	return &AppError{
		Type:    t,
		Message: msg,
		Err:     err,
	}
}

// Git errors
var (
	ErrNoChanges = NewAppError(TypeGit, "No staged changes detected", nil).
			WithSuggestion("Stage your changes first with: git add <files>")

	ErrGetBranch = NewAppError(TypeGit, "Failed to get current branch", nil).
			WithSuggestion("Make sure you are in a git repository: git status")

	ErrNoBranch = NewAppError(TypeGit, "No branch detected", nil).
			WithSuggestion("Create a branch first: git checkout -b <branch-name>")

	ErrGetRepoRoot = NewAppError(TypeGit, "Failed to get repository root", nil).
			WithSuggestion("Make sure you are inside a git repository")

	ErrGetRepoURL = NewAppError(TypeGit, "Failed to get repository URL", nil).
			WithSuggestion("Add a remote: git remote add origin <url>")

	ErrGetCommits = NewAppError(TypeGit, "Failed to get commits", nil).
			WithSuggestion("Make sure you have commits in your repository: git log")

	ErrGetCommitCount = NewAppError(TypeGit, "Failed to get commit count", nil)

	ErrGetRecentCommits = NewAppError(TypeGit, "Failed to get recent commit messages", nil).
				WithSuggestion("Verify repository has commits: git log --oneline")

	ErrAddFile = NewAppError(TypeGit, "Failed to add file to staging", nil).
			WithSuggestion("Check if the file exists and you have write permissions")

	ErrExtractRepoInfo = NewAppError(TypeGit, "Failed to extract repository info", nil)

	ErrCreateTag = NewAppError(TypeGit, "Failed to create tag", nil).
			WithSuggestion("Make sure the tag doesn't already exist: git tag -l")

	ErrPushTag = NewAppError(TypeGit, "Failed to push tag", nil).
			WithSuggestion("Check your remote connection: git remote -v")

	ErrPush = NewAppError(TypeGit, "Failed to push to remote", nil).
		WithSuggestion("Verify remote is configured: git remote -v")

	ErrFetchTags = NewAppError(TypeGit, "Failed to fetch tags from remote", nil).
			WithSuggestion("Check your network connection and remote access")

	ErrCreateCommit = NewAppError(TypeGit, "Failed to create commit", nil).
			WithSuggestion("Ensure git user is configured:\n   git config --global user.name \"Your Name\"\n   git config --global user.email \"your@email.com\"")

	ErrGetDiff = NewAppError(TypeGit, "Failed to get diff", nil).
			WithSuggestion("Check if you have staged changes: git status")

	ErrNoDiff = NewAppError(TypeGit, "No differences detected", nil).
			WithSuggestion("Stage your changes first: git add <files>")

	ErrInvalidBranch = NewAppError(TypeGit, "Must be on main or master branch to create releases", nil).
				WithSuggestion("Switch to main branch: git checkout main")

	ErrTagNotFound = NewAppError(TypeGit, "Tag not found in repository history", nil).
			WithSuggestion("List available tags: git tag -l")

	ErrInvalidTagFormat = NewAppError(TypeGit, "Tag does not match semver format (vX.Y.Z)", nil).
				WithSuggestion("Use semantic versioning format: v1.0.0, v2.1.3, etc.")

	ErrValidateTag = NewAppError(TypeGit, "Failed to validate tag existence", nil)

	ErrGetChangedFiles = NewAppError(TypeGit, "Failed to get changed files", nil).
				WithSuggestion("Verify you have staged changes: git status")

	ErrGetTagDate = NewAppError(TypeGit, "Failed to get tag date", nil)

	ErrGetGitUser = NewAppError(TypeGit, "Failed to get git user configuration", nil).
			WithSuggestion("Configure git user:\n   git config --global user.name \"Your Name\"\n   git config --global user.email \"your@email.com\"")

	ErrGitUserNotConfigured = NewAppError(TypeGit, "git user.name not configured", nil).
				WithSuggestion("Set your git username: git config --global user.name \"Your Name\"")

	ErrGitEmailNotConfigured = NewAppError(TypeGit, "git user.email not configured", nil).
					WithSuggestion("Set your git email: git config --global user.email \"your@email.com\"")

	ErrNotInGitRepo = NewAppError(TypeGit, "Not in a git repository", nil).
			WithSuggestion("Initialize a git repository: git init")
)

// Configuration errors
var (
	ErrAPIKeyMissing = NewAppError(TypeConfiguration, "AI API key is missing", nil).
				WithSuggestion("Run: matecommit config init --quick")

	ErrTokenMissing = NewAppError(TypeConfiguration, "VCS Token is missing", nil).
			WithSuggestion("Configure GitHub token: matecommit config init --full")

	ErrConfigMissing = NewAppError(TypeConfiguration, "Configuration is missing", nil).
				WithSuggestion("Initialize configuration: matecommit config init")
)

// VCS errors
var (
	ErrRepositoryNotFound = NewAppError(TypeVCS, "repository not found", nil).
				WithSuggestion("Check repository URL and access permissions")

	ErrVCSNotSupported = NewAppError(TypeVCS, "VCS provider not supported", nil).
				WithSuggestion("Currently only GitHub is supported")

	ErrCreateRelease = NewAppError(TypeVCS, "failed to create release", nil).
				WithSuggestion("Check your GitHub token has 'repo' permissions")

	ErrUpdateRelease = NewAppError(TypeVCS, "failed to update release", nil).
				WithSuggestion("Verify the release exists: gh release list")

	ErrGetRelease = NewAppError(TypeVCS, "failed to get release", nil).
			WithSuggestion("List available releases: gh release list")

	ErrUploadAsset = NewAppError(TypeVCS, "failed to upload release asset", nil).
			WithSuggestion("Check file exists and is readable")
)

// GitHub/VCS specific errors
var (
	ErrGitHubTokenInvalid = NewAppError(TypeVCS, "GitHub token is invalid or expired", nil).
				WithSuggestion("Generate a new token at: https://github.com/settings/tokens\nThen run: matecommit config init --full")

	ErrGitHubInsufficientPerms = NewAppError(TypeVCS, "GitHub token has insufficient permissions", nil).
					WithSuggestion("Token needs 'repo' and 'workflow' scopes.\nRegenerate at: https://github.com/settings/tokens")

	ErrGitHubRateLimit = NewAppError(TypeVCS, "GitHub API rate limit exceeded", nil).
				WithSuggestion("Wait a few minutes or use a personal access token for higher limits")
)

// AI errors
var (
	ErrQuotaExceeded = NewAppError(TypeAI, "AI quota exceeded or rate limited", nil).
				WithSuggestion("Wait a few minutes and try again, or check your API quota")

	ErrAIGeneration = NewAppError(TypeAI, "AI generation failed", nil).
			WithSuggestion("Try again or check your API key configuration")

	ErrInvalidAIOutput = NewAppError(TypeAI, "invalid AI output format", nil).
				WithSuggestion("This is likely a temporary issue, please try again")
)

// Gemini/AI specific errors
var (
	ErrGeminiAPIKeyInvalid = NewAppError(TypeAI, "Gemini API key is invalid", nil).
				WithSuggestion("Get a valid API key at: https://makersuite.google.com/app/apikey\nThen run: matecommit config init --quick")

	ErrGeminiQuotaExceeded = NewAppError(TypeAI, "Gemini API quota exceeded", nil).
				WithSuggestion("Wait for quota to reset or upgrade your Gemini plan")
)

// Update errors
var (
	ErrUpdateFailed = NewAppError(TypeUpdate, "Failed to update application", nil).
		WithSuggestion("Try manual update from: https://github.com/thomas-vilte/matecommit/releases")
)

var (
	ErrBuildNoVersion  = NewAppError(TypeInternal, "Build version not specified", nil)
	ErrBuildNoCommit   = NewAppError(TypeInternal, "Build commit not specified", nil)
	ErrBuildNoBuildDir = NewAppError(TypeInternal, "Build directory not specified", nil)
	ErrBuildNoDate     = NewAppError(TypeInternal, "Build date not specified", nil)
	ErrBuildFailed     = NewAppError(TypeInternal, "Build operation failed", nil).
				WithSuggestion("Check build logs for compilation errors or missing dependencies")
	ErrVersionFileNotFound = NewAppError(TypeInternal, "Version file not found", nil).
				WithSuggestion("Specify version file in config: matecommit config set version_file <path>")
)
