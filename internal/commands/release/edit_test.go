package release

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func getTestTranslations(t *testing.T) *i18n.Translations {
	trans, err := i18n.NewTranslations("en", "../../i18n/locales")
	if err != nil {
		t.Logf("Warning: using empty translations: %v", err)
		trans = &i18n.Translations{}
	}
	return trans
}

func TestEditReleaseAction_Success(t *testing.T) {
	mockService := new(MockReleaseService)
	mockGit := new(MockGitService)
	trans := getTestTranslations(t)

	existingRelease := &models.VCSRelease{
		TagName: "v1.2.0",
		Name:    "Release v1.2.0",
		Body:    "Original release notes",
	}

	mockService.On("GetRelease", mock.Anything, "v1.2.0").Return(existingRelease, nil)
	mockService.On("UpdateRelease", mock.Anything, "v1.2.0", mock.AnythingOfType("string")).Return(nil)

	tmpDir := t.TempDir()
	mockEditor := filepath.Join(tmpDir, "mock-editor.sh")
	editorScript := `#!/bin/bash
echo "Edited content" >> "$1"
`
	err := os.WriteFile(mockEditor, []byte(editorScript), 0755)
	require.NoError(t, err)

	app := &cli.Command{
		Commands: []*cli.Command{
			{
				Name: "edit",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "version",
						Aliases:  []string{"v"},
						Required: true,
					},
					&cli.StringFlag{
						Name:  "editor",
						Value: mockEditor,
					},
				},
				Action: editReleaseAction(mockService, mockGit, trans),
			},
		},
	}

	err = app.Run(context.Background(), []string{"matecommit", "edit", "--version", "v1.2.0", "--editor", mockEditor})
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}

func TestEditReleaseAction_GetReleaseError(t *testing.T) {
	mockService := new(MockReleaseService)
	mockGit := new(MockGitService)
	trans := getTestTranslations(t)

	mockService.On("GetRelease", mock.Anything, "v1.2.0").Return((*models.VCSRelease)(nil), errors.New("release not found"))

	app := &cli.Command{
		Commands: []*cli.Command{
			{
				Name: "edit",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "version",
						Aliases:  []string{"v"},
						Required: true,
					},
				},
				Action: editReleaseAction(mockService, mockGit, trans),
			},
		},
	}

	err := app.Run(context.Background(), []string{"matecommit", "edit", "--version", "v1.2.0"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "release not found")

	mockService.AssertExpectations(t)
}

func TestEditReleaseAction_EditorError(t *testing.T) {
	mockService := new(MockReleaseService)
	mockGit := new(MockGitService)
	trans := getTestTranslations(t)

	existingRelease := &models.VCSRelease{
		TagName: "v1.2.0",
		Name:    "Release v1.2.0",
		Body:    "Original release notes",
	}

	mockService.On("GetRelease", mock.Anything, "v1.2.0").Return(existingRelease, nil)

	app := &cli.Command{
		Commands: []*cli.Command{
			{
				Name: "edit",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "version",
						Aliases:  []string{"v"},
						Required: true,
					},
					&cli.StringFlag{
						Name:  "editor",
						Value: "/nonexistent/editor",
					},
				},
				Action: editReleaseAction(mockService, mockGit, trans),
			},
		},
	}

	err := app.Run(context.Background(), []string{"matecommit", "edit", "--version", "v1.2.0", "--editor", "/nonexistent/editor"})
	assert.Error(t, err)

	mockService.AssertCalled(t, "GetRelease", mock.Anything, "v1.2.0")
}

func TestEditReleaseAction_UpdateReleaseError(t *testing.T) {
	mockService := new(MockReleaseService)
	mockGit := new(MockGitService)
	trans := getTestTranslations(t)

	existingRelease := &models.VCSRelease{
		TagName: "v1.2.0",
		Name:    "Release v1.2.0",
		Body:    "Original release notes",
	}

	mockService.On("GetRelease", mock.Anything, "v1.2.0").Return(existingRelease, nil)
	mockService.On("UpdateRelease", mock.Anything, "v1.2.0", mock.AnythingOfType("string")).Return(errors.New("update failed"))

	tmpDir := t.TempDir()
	mockEditor := filepath.Join(tmpDir, "mock-editor.sh")
	editorScript := `#!/bin/bash
exit 0
`
	err := os.WriteFile(mockEditor, []byte(editorScript), 0755)
	require.NoError(t, err)

	app := &cli.Command{
		Commands: []*cli.Command{
			{
				Name: "edit",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "version",
						Aliases:  []string{"v"},
						Required: true,
					},
					&cli.StringFlag{
						Name:  "editor",
						Value: mockEditor,
					},
				},
				Action: editReleaseAction(mockService, mockGit, trans),
			},
		},
	}

	err = app.Run(context.Background(), []string{"matecommit", "edit", "--version", "v1.2.0", "--editor", mockEditor})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")

	mockService.AssertExpectations(t)
}

func TestNewEditCommand(t *testing.T) {
	trans := getTestTranslations(t)
	mockGit := new(MockGitService)

	factory := &ReleaseCommandFactory{
		gitService: mockGit,
	}

	cmd := factory.newEditCommand(trans)

	assert.NotNil(t, cmd)
	assert.Equal(t, "edit", cmd.Name)
	assert.Contains(t, cmd.Aliases, "e")
	assert.NotNil(t, cmd.Action)
	assert.Len(t, cmd.Flags, 3)

	versionFlag := cmd.Flags[0].(*cli.StringFlag)
	assert.Equal(t, "version", versionFlag.Name)
	assert.Contains(t, versionFlag.Aliases, "v")
	assert.True(t, versionFlag.Required)

	editorFlag := cmd.Flags[1].(*cli.StringFlag)
	assert.Equal(t, "editor", editorFlag.Name)
	assert.Contains(t, editorFlag.Aliases, "e")
	assert.NotEmpty(t, editorFlag.Value)

	aiFlag := cmd.Flags[2].(*cli.BoolFlag)
	assert.Equal(t, "ai", aiFlag.Name)
	assert.Contains(t, aiFlag.Aliases, "a")
	assert.False(t, aiFlag.Value)
}

func TestGetDefaultEditor_EDITOR(t *testing.T) {
	originalEditor := os.Getenv("EDITOR")
	defer func() {
		if originalEditor != "" {
			_ = os.Setenv("EDITOR", originalEditor)
		} else {
			_ = os.Unsetenv("EDITOR")
		}
	}()

	_ = os.Setenv("EDITOR", "emacs")
	editor := getDefaultEditor()
	assert.Equal(t, "emacs", editor)
}

func TestGetDefaultEditor_VISUAL(t *testing.T) {
	originalEditor := os.Getenv("EDITOR")
	originalVisual := os.Getenv("VISUAL")
	defer func() {
		if originalEditor != "" {
			_ = os.Setenv("EDITOR", originalEditor)
		} else {
			_ = os.Unsetenv("EDITOR")
		}
		if originalVisual != "" {
			_ = os.Setenv("VISUAL", originalVisual)
		} else {
			_ = os.Unsetenv("VISUAL")
		}
	}()

	_ = os.Unsetenv("EDITOR")
	_ = os.Setenv("VISUAL", "code")
	editor := getDefaultEditor()
	assert.Equal(t, "code", editor)
}

func TestGetDefaultEditor_Default(t *testing.T) {
	originalEditor := os.Getenv("EDITOR")
	originalVisual := os.Getenv("VISUAL")
	defer func() {
		if originalEditor != "" {
			_ = os.Setenv("EDITOR", originalEditor)
		}
		if originalVisual != "" {
			_ = os.Setenv("VISUAL", originalVisual)
		}
	}()

	_ = os.Unsetenv("EDITOR")
	_ = os.Unsetenv("VISUAL")
	editor := getDefaultEditor()
	assert.Contains(t, []string{"nano", "vim", "vi"}, editor)
}

func TestGetDefaultEditor_AllBranches(t *testing.T) {
	originalEditor := os.Getenv("EDITOR")
	originalVisual := os.Getenv("VISUAL")
	defer func() {
		if originalEditor != "" {
			_ = os.Setenv("EDITOR", originalEditor)
		}
		if originalVisual != "" {
			_ = os.Setenv("VISUAL", originalVisual)
		}
	}()

	_ = os.Setenv("EDITOR", "test-editor-1")
	_ = os.Unsetenv("VISUAL")
	assert.Equal(t, "test-editor-1", getDefaultEditor())

	_ = os.Unsetenv("EDITOR")
	_ = os.Setenv("VISUAL", "test-editor-2")
	assert.Equal(t, "test-editor-2", getDefaultEditor())

	_ = os.Unsetenv("EDITOR")
	_ = os.Unsetenv("VISUAL")
	result := getDefaultEditor()

	assert.NotEmpty(t, result)

	_, err := exec.LookPath("nano")
	if err == nil {
		assert.Equal(t, "nano", result)
		return
	}

	_, err = exec.LookPath("vim")
	if err == nil {
		assert.Equal(t, "vim", result)
		return
	}

	assert.Equal(t, "vi", result)
}

func TestGetDefaultEditor_FallbackToVi(t *testing.T) {
	originalEditor := os.Getenv("EDITOR")
	originalVisual := os.Getenv("VISUAL")
	originalPath := os.Getenv("PATH")
	defer func() {
		if originalEditor != "" {
			_ = os.Setenv("EDITOR", originalEditor)
		}
		if originalVisual != "" {
			_ = os.Setenv("VISUAL", originalVisual)
		}
		_ = os.Setenv("PATH", originalPath)
	}()

	_ = os.Unsetenv("EDITOR")
	_ = os.Unsetenv("VISUAL")

	_ = os.Setenv("PATH", "")

	result := getDefaultEditor()
	assert.Equal(t, "vi", result)
}

func TestEditReleaseAction_WriteFileError(t *testing.T) {
	mockService := new(MockReleaseService)
	mockGit := new(MockGitService)
	trans := getTestTranslations(t)

	existingRelease := &models.VCSRelease{
		TagName: "v1.2.0",
		Name:    "Release v1.2.0",
		Body:    "Original release notes",
	}

	mockService.On("GetRelease", mock.Anything, "v1.2.0").Return(existingRelease, nil)

	app := &cli.Command{
		Commands: []*cli.Command{
			{
				Name: "edit",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "version",
						Aliases:  []string{"v"},
						Required: true,
					},
					&cli.StringFlag{
						Name:  "editor",
						Value: "true",
					},
				},
				Action: editReleaseAction(mockService, mockGit, trans),
			},
		},
	}

	mockService.On("UpdateRelease", mock.Anything, "v1.2.0", mock.AnythingOfType("string")).Return(nil)

	err := app.Run(context.Background(), []string{"matecommit", "edit", "--version", "v1.2.0", "--editor", "true"})
	assert.NoError(t, err)
}

func TestEditReleaseAction_CompleteFlow(t *testing.T) {
	mockService := new(MockReleaseService)
	mockGit := new(MockGitService)
	trans := getTestTranslations(t)

	existingRelease := &models.VCSRelease{
		TagName: "v2.0.0",
		Name:    "Release v2.0.0",
		Body:    "# Release Notes\n\nOriginal content here",
	}

	mockService.On("GetRelease", mock.Anything, "v2.0.0").Return(existingRelease, nil)

	var capturedBody string
	mockService.On("UpdateRelease", mock.Anything, "v2.0.0", mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			capturedBody = args.Get(2).(string)
		}).
		Return(nil)

	tmpDir := t.TempDir()
	mockEditor := filepath.Join(tmpDir, "edit-script.sh")
	editorScript := `#!/bin/bash
# Read the file, append something, and write back
echo "" >> "$1"
echo "## Additional Notes" >> "$1"
echo "- Feature A" >> "$1"
`
	err := os.WriteFile(mockEditor, []byte(editorScript), 0755)
	require.NoError(t, err)

	app := &cli.Command{
		Commands: []*cli.Command{
			{
				Name: "edit",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "version",
						Aliases:  []string{"v"},
						Required: true,
					},
					&cli.StringFlag{
						Name:  "editor",
						Value: mockEditor,
					},
				},
				Action: editReleaseAction(mockService, mockGit, trans),
			},
		},
	}

	err = app.Run(context.Background(), []string{"matecommit", "edit", "--version", "v2.0.0", "--editor", mockEditor})
	assert.NoError(t, err)

	assert.Contains(t, capturedBody, "Original content here")
	assert.Contains(t, capturedBody, "Additional Notes")
	assert.Contains(t, capturedBody, "Feature A")

	mockService.AssertExpectations(t)
}
