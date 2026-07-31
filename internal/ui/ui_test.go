package ui

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
)

func TestShownAndIsShown(t *testing.T) {
	t.Run("nil error stays nil", func(t *testing.T) {
		assert.Nil(t, Shown(nil))
	})

	t.Run("wrapped error reports IsShown true", func(t *testing.T) {
		err := Shown(errors.New("boom"))
		assert.True(t, IsShown(err))
		assert.Equal(t, "boom", err.Error())
	})

	t.Run("a plain, never-shown error reports IsShown false", func(t *testing.T) {
		assert.False(t, IsShown(errors.New("boom")))
	})

	t.Run("IsShown sees through additional wrapping (errors.As walks Unwrap)", func(t *testing.T) {
		err := wrapForTest(Shown(errors.New("boom")))
		assert.True(t, IsShown(err))
	})
}

func TestPrintError_ReturnsShownError(t *testing.T) {
	var buf bytes.Buffer

	err := PrintError(&buf, "something broke")

	assert.True(t, IsShown(err), "the error PrintError returns must be marked as already displayed")
	assert.Equal(t, "something broke", err.Error())
	assert.Contains(t, buf.String(), "something broke")
}

func TestHandleAppError_ReturnsShownError(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		assert.Nil(t, HandleAppError(nil))
	})

	t.Run("AppError is marked shown and preserves the original error for errors.As", func(t *testing.T) {
		original := domainErrors.NewAppError(domainErrors.TypeGit, "something git-related broke", nil)

		result := HandleAppError(original)

		assert.True(t, IsShown(result), "must be marked as already displayed so main() doesn't print it again")

		var appErr *domainErrors.AppError
		assert.True(t, errors.As(result, &appErr), "the original AppError must still be reachable via errors.As")
		assert.Equal(t, "something git-related broke", appErr.Message)
	})

	t.Run("plain (non-AppError) error is marked shown too", func(t *testing.T) {
		result := HandleAppError(errors.New("generic failure"))

		assert.True(t, IsShown(result))
		assert.Equal(t, "generic failure", result.Error())
	})
}

func wrapForTest(err error) error {
	return &wrappedForTest{err: err}
}

type wrappedForTest struct{ err error }

func (w *wrappedForTest) Error() string { return "wrapped: " + w.err.Error() }
func (w *wrappedForTest) Unwrap() error { return w.err }
