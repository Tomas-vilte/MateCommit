package errors

import (
	"errors"
	"testing"
)

func TestAppError_WithError(t *testing.T) {
	baseErr := errors.New("original error")
	appErr := ErrGetDiff.WithError(baseErr)

	if appErr.Err != baseErr {
		t.Errorf("Expected underlying error to be %v, got %v", baseErr, appErr.Err)
	}

	if appErr.Type != TypeGit {
		t.Errorf("Expected type %s, got %s", TypeGit, appErr.Type)
	}
}

func TestAppError_WithContext(t *testing.T) {
	appErr := ErrAddFile.WithContext("file", "test.txt").WithContext("stderr", "file not found")

	if appErr.Context["file"] != "test.txt" {
		t.Errorf("Expected file context 'test.txt', got %v", appErr.Context["file"])
	}

	if appErr.Context["stderr"] != "file not found" {
		t.Errorf("Expected stderr context 'file not found', got %v", appErr.Context["stderr"])
	}
}

func TestAppError_Error_Format(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		contains []string
	}{
		{
			name: "Simple error without underlying error",
			err:  ErrNoChanges,
			contains: []string{
				"GIT",
				"No staged changes detected",
			},
		},
		{
			name: "Error with underlying error",
			err:  ErrGetBranch.WithError(errors.New("exit status 1")),
			contains: []string{
				"GIT",
				"Failed to get current branch",
				"exit status 1",
			},
		},
		{
			name: "Error with context including stderr",
			err: ErrAddFile.WithError(errors.New("exit status 128")).
				WithContext("file", "test.go").
				WithContext("stderr", "did not match any files"),
			contains: []string{
				"GIT",
				"Failed to add file to staging",
				"exit status 128",
				"did not match any files",
			},
		},
		{
			name: "Error with multiple context fields",
			err: ErrGetDiff.WithError(errors.New("command failed")).
				WithContext("diff_type", "staged").
				WithContext("stderr", "repository not found"),
			contains: []string{
				"GIT",
				"Failed to get diff",
				"command failed",
				"repository not found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()
			for _, substr := range tt.contains {
				if !contains(errMsg, substr) {
					t.Errorf("Expected error message to contain %q, got: %s", substr, errMsg)
				}
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	appErr := ErrCreateCommit.WithError(baseErr)

	unwrapped := appErr.Unwrap()
	if unwrapped != baseErr {
		t.Errorf("Expected unwrapped error to be %v, got %v", baseErr, unwrapped)
	}

	// Test errors.Is functionality
	if !errors.Is(appErr, baseErr) {
		t.Error("errors.Is should work with AppError")
	}
}

func TestAppError_ChainedContext(t *testing.T) {
	appErr := ErrCreateTag.
		WithError(errors.New("tag exists")).
		WithContext("version", "v1.0.0").
		WithContext("remote", "origin")

	if appErr.Context["version"] != "v1.0.0" {
		t.Errorf("Expected version context, got %v", appErr.Context["version"])
	}

	if appErr.Context["remote"] != "origin" {
		t.Errorf("Expected remote context, got %v", appErr.Context["remote"])
	}

	// Ensure we didn't modify the original error
	if ErrCreateTag.Context != nil {
		t.Error("Original error should not have context")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr))))
}
