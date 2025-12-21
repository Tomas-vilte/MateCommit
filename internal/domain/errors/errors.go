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
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
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
	ErrNoChanges       = NewAppError(TypeGit, "no staged changes detected", nil)
	ErrGetBranch       = NewAppError(TypeGit, "failed to get current branch", nil)
	ErrNoBranch        = NewAppError(TypeGit, "no branch detected", nil)
	ErrGetRepoRoot     = NewAppError(TypeGit, "failed to get repository root", nil)
	ErrGetRepoURL      = NewAppError(TypeGit, "failed to get repository URL", nil)
	ErrGetCommits      = NewAppError(TypeGit, "failed to get commits", nil)
	ErrAddFile         = NewAppError(TypeGit, "failed to add file to staging", nil)
	ErrExtractRepoInfo = NewAppError(TypeGit, "failed to extract repository info", nil)
	ErrCreateTag       = NewAppError(TypeGit, "failed to create tag", nil)
	ErrPushTag         = NewAppError(TypeGit, "failed to push tag", nil)
	ErrCreateCommit    = NewAppError(TypeGit, "failed to create commit", nil)
	ErrGetDiff         = NewAppError(TypeGit, "failed to get diff", nil)
	ErrNoDiff          = NewAppError(TypeGit, "no differences detected", nil)
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

// AI errors
var (
	ErrQuotaExceeded   = NewAppError(TypeAI, "AI quota exceeded or rate limited", nil)
	ErrAIGeneration    = NewAppError(TypeAI, "AI generation failed", nil)
	ErrInvalidAIOutput = NewAppError(TypeAI, "invalid AI output format", nil)
)

// Internal errors
var (
	ErrNetwork = NewAppError(TypeInternal, "network error occurred", nil)
	ErrBuild   = NewAppError(TypeInternal, "build operation failed", nil)
)

// Update errors
var (
	ErrUpdateFailed = NewAppError(TypeUpdate, "failed to update application", nil)
)

var (
	ErrBuildNoVersion  = NewAppError(TypeInternal, "build version not specified", nil)
	ErrBuildNoCommit   = NewAppError(TypeInternal, "build commit not specified", nil)
	ErrBuildNoBuildDir = NewAppError(TypeInternal, "build directory not specified", nil)
	ErrBuildNoDate     = NewAppError(TypeInternal, "build date not specified", nil)
	ErrBuildFailed     = NewAppError(TypeInternal, "build operation failed", nil)
)
