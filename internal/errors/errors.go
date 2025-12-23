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
	Type    ErrorType
	Message string
	Context map[string]interface{}
	Err     error
}

func (e *AppError) Error() string {
	var msg string
	if e.Err != nil {
		msg = fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Err)
	} else {
		msg = fmt.Sprintf("%s: %s", e.Type, e.Message)
	}

	// Include stderr context if available for better error messages
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
		Type:    e.Type,
		Message: e.Message,
		Context: e.Context,
		Err:     err,
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
		Type:    e.Type,
		Message: e.Message,
		Context: ctx,
		Err:     e.Err,
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
	ErrNoChanges             = NewAppError(TypeGit, "no staged changes detected", nil)
	ErrGetBranch             = NewAppError(TypeGit, "failed to get current branch", nil)
	ErrNoBranch              = NewAppError(TypeGit, "no branch detected", nil)
	ErrGetRepoRoot           = NewAppError(TypeGit, "failed to get repository root", nil)
	ErrGetRepoURL            = NewAppError(TypeGit, "failed to get repository URL", nil)
	ErrGetCommits            = NewAppError(TypeGit, "failed to get commits", nil)
	ErrGetCommitCount        = NewAppError(TypeGit, "failed to get commit count", nil)
	ErrGetRecentCommits      = NewAppError(TypeGit, "failed to get recent commit messages", nil)
	ErrAddFile               = NewAppError(TypeGit, "failed to add file to staging", nil)
	ErrExtractRepoInfo       = NewAppError(TypeGit, "failed to extract repository info", nil)
	ErrCreateTag             = NewAppError(TypeGit, "failed to create tag", nil)
	ErrPushTag               = NewAppError(TypeGit, "failed to push tag", nil)
	ErrPush                  = NewAppError(TypeGit, "failed to push to remote", nil)
	ErrFetchTags             = NewAppError(TypeGit, "failed to fetch tags from remote", nil)
	ErrCreateCommit          = NewAppError(TypeGit, "failed to create commit", nil)
	ErrGetDiff               = NewAppError(TypeGit, "failed to get diff", nil)
	ErrNoDiff                = NewAppError(TypeGit, "no differences detected", nil)
	ErrInvalidBranch         = NewAppError(TypeGit, "must be on main or master branch to create releases", nil)
	ErrTagNotFound           = NewAppError(TypeGit, "tag not found in repository history", nil)
	ErrInvalidTagFormat      = NewAppError(TypeGit, "tag does not match semver format (vX.Y.Z)", nil)
	ErrValidateTag           = NewAppError(TypeGit, "failed to validate tag existence", nil)
	ErrGetChangedFiles       = NewAppError(TypeGit, "failed to get changed files", nil)
	ErrGetTagDate            = NewAppError(TypeGit, "failed to get tag date", nil)
	ErrGetGitUser            = NewAppError(TypeGit, "failed to get git user configuration", nil)
	ErrGitUserNotConfigured  = NewAppError(TypeGit, "git user.name not configured", nil)
	ErrGitEmailNotConfigured = NewAppError(TypeGit, "git user.email not configured", nil)
	ErrNotInGitRepo          = NewAppError(TypeGit, "not in a git repository", nil)
)

// Configuration errors
var (
	ErrAPIKeyMissing = NewAppError(TypeConfiguration, "AI API key is missing", nil)
	ErrTokenMissing  = NewAppError(TypeConfiguration, "VCS token is missing", nil)
	ErrConfigMissing = NewAppError(TypeConfiguration, "configuration is missing", nil)
)

// VCS errors
var (
	ErrRepositoryNotFound = NewAppError(TypeVCS, "repository not found", nil)
	ErrVCSNotSupported    = NewAppError(TypeVCS, "VCS provider not supported", nil)
	ErrCreateRelease      = NewAppError(TypeVCS, "failed to create release", nil)
	ErrUpdateRelease      = NewAppError(TypeVCS, "failed to update release", nil)
	ErrGetRelease         = NewAppError(TypeVCS, "failed to get release", nil)
	ErrUploadAsset        = NewAppError(TypeVCS, "failed to upload release asset", nil)
)

// GitHub/VCS specific errors
var (
	ErrGitHubTokenInvalid      = NewAppError(TypeVCS, "GitHub token is invalid or expired", nil)
	ErrGitHubInsufficientPerms = NewAppError(TypeVCS, "GitHub token has insufficient permissions", nil)
	ErrGitHubRateLimit         = NewAppError(TypeVCS, "GitHub API rate limit exceeded", nil)
)

// AI errors
var (
	ErrQuotaExceeded   = NewAppError(TypeAI, "AI quota exceeded or rate limited", nil)
	ErrAIGeneration    = NewAppError(TypeAI, "AI generation failed", nil)
	ErrInvalidAIOutput = NewAppError(TypeAI, "invalid AI output format", nil)
)

// Gemini/AI specific errors
var (
	ErrGeminiAPIKeyInvalid = NewAppError(TypeAI, "Gemini API key is invalid", nil)
	ErrGeminiQuotaExceeded = NewAppError(TypeAI, "Gemini API quota exceeded", nil)
)

// Update errors
var (
	ErrUpdateFailed = NewAppError(TypeUpdate, "failed to update application", nil)
)
var (
	ErrBuildNoVersion      = NewAppError(TypeInternal, "build version not specified", nil)
	ErrBuildNoCommit       = NewAppError(TypeInternal, "build commit not specified", nil)
	ErrBuildNoBuildDir     = NewAppError(TypeInternal, "build directory not specified", nil)
	ErrBuildNoDate         = NewAppError(TypeInternal, "build date not specified", nil)
	ErrBuildFailed         = NewAppError(TypeInternal, "build operation failed", nil)
	ErrVersionFileNotFound = NewAppError(TypeInternal, "version file not found", nil)
)
