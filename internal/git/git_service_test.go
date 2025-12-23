package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
)

var originalDir string

func init() {
	var err error
	originalDir, err = os.Getwd()
	if err != nil {
		panic("Error obteniendo directorio original: " + err.Error())
	}
}

func setupTestRepo(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "git-test")
	if err != nil {
		t.Fatalf("Error creando directorio temporal: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Error cambiando al directorio temporal: %v", err)
	}

	cmd := exec.Command("git", "init")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Error inicializando repositorio git: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Error configurando email git: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Error configurando nombre git: %v", err)
	}

	return tempDir
}

func cleanupTestRepo(t *testing.T, dir string) {
	if err := os.Chdir(originalDir); err != nil {
		t.Errorf("Error volviendo al directorio original: %v", err)
		return
	}

	if err := os.RemoveAll(dir); err != nil {
		t.Errorf("Error limpiando directorio de prueba: %v", err)
	}
}

func TestGitService(t *testing.T) {
	t.Run("HasStagedChanges", func(t *testing.T) {
		// Arrange
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		hasStagedBefore := service.HasStagedChanges(context.Background())

		testFile := filepath.Join("test.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatalf("Error creando archivo de prueba: %v", err)
		}

		cmd := exec.Command("git", "add", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Error haciendo stage del archivo: %v", err)
		}

		hasStagedAfter := service.HasStagedChanges(context.Background())

		// Assert
		if hasStagedBefore {
			t.Error("No debería haber cambios staged antes de agregar archivos")
		}
		if !hasStagedAfter {
			t.Error("Debería haber cambios staged después de agregar archivos")
		}
	})

	t.Run("GetCurrentBranch", func(t *testing.T) {
		// Arrange
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		branchName, err := service.GetCurrentBranch(context.Background())

		// Assert
		if err != nil {
			t.Errorf("Error obteniendo la branch actual: %v", err)
		}
		if branchName != "master" {
			t.Errorf("Se esperaba la branch 'main', se obtuvo: %s", branchName)
		}

		// Act - Crear una nueva branch y cambiarse a ella
		newBranch := "feature/test-branch"
		cmd := exec.Command("git", "checkout", "-b", newBranch)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Error creando y cambiando a la nueva branch: %v", err)
		}

		branchName, err = service.GetCurrentBranch(context.Background())

		// Assert
		if err != nil {
			t.Errorf("Error obteniendo la branch actual: %v", err)
		}
		if branchName != newBranch {
			t.Errorf("Se esperaba la branch '%s', se obtuvo: %s", newBranch, branchName)
		}
	})

	t.Run("GetChangedFiles", func(t *testing.T) {
		// arrange
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		if err := os.WriteFile("test.txt", []byte("test content"), 0644); err != nil {
			t.Fatalf("Error creando archivo de prueba: %v", err)
		}

		// act
		changes, err := service.GetChangedFiles(context.Background())

		// assert
		if err != nil {
			t.Errorf("Error obteniendo archivos cambiados: %v", err)
		}

		if len(changes) > 0 {
			expectedPath := "test.txt"

			if changes[0] != expectedPath {
				t.Errorf("Path esperado %s, se obtuvo %s", expectedPath, changes[0])
			}
		}
	})

	t.Run("CreateCommit", func(t *testing.T) {
		// arrange
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		if err := os.WriteFile("test.txt", []byte("test content"), 0644); err != nil {
			t.Fatalf("Error creando archivo de prueba: %v", err)
		}

		if err := exec.Command("git", "add", ".").Run(); err != nil {
			t.Fatalf("Error haciendo stage de los cambios: %v", err)
		}

		// act
		err := service.CreateCommit(context.Background(), "Test Commit")

		// assert
		if err != nil {
			t.Errorf("Error creando commit: %v", err)
		}

		cmd := exec.Command("git", "log", "--oneline")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("Error verificando log de git: %v", err)
		}

		if len(output) == 0 {
			t.Error("No se encontró el commit en el historial")
		}
	})

	t.Run("CreateCommit without organized changes", func(t *testing.T) {
		// Arrange
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		// Act
		err := service.CreateCommit(context.Background(), "Test commit")

		// Assert
		if err == nil {
			t.Error("Se esperaba un error al crear commit sin cambios staged")
		}
		if !errors.Is(err, domainErrors.ErrNoChanges) {
			t.Errorf("Se esperaba ErrNoChanges, obtuve: %v", err)
		}
	})

	t.Run("GetDiff with staged files", func(t *testing.T) {
		// arrange
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		if err := os.WriteFile("test.txt", []byte("test content"), 0644); err != nil {
			t.Fatalf("Error creando archivo: %v", err)
		}

		cmd := exec.Command("git", "add", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Error haciendo stage del archivo: %v", err)
		}

		// act
		diff, err := service.GetDiff(context.Background())

		if err != nil {
			t.Errorf("Error obteniendo diff: %v", err)
		}

		if !strings.Contains(diff, "test.txt") {
			t.Error("El diff no contiene el archivo modificado")
		}

		if !strings.Contains(diff, "test content") {
			t.Error("El diff no contiene el contenido del archivo")
		}
	})

	t.Run("GetDiff with unstaged files", func(t *testing.T) {
		// arrange
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		if err := os.WriteFile("test.txt", []byte("test content"), 0644); err != nil {
			t.Fatalf("Error creando archivo: %v", err)
		}

		cmd := exec.Command("git", "add", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Error haciendo stage del archivo: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "commit inicial")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Error creando commit inicial: %v", err)
		}

		if err := os.WriteFile("test.txt", []byte("contenido modificado"), 0644); err != nil {
			t.Fatalf("Error modificando archivo: %v", err)
		}

		// act
		diff, err := service.GetDiff(context.Background())

		// assert
		if err != nil {
			t.Errorf("Error obteniendo diff: %v", err)
		}

		if !strings.Contains(diff, "test.txt") {
			t.Error("El diff no contiene el archivo modificado")
		}
		if !strings.Contains(diff, "contenido modificado") {
			t.Error("El diff no contiene los cambios sin stage")
		}
	})

	t.Run("GetDiff with new untracked files", func(t *testing.T) {
		// arrange
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		if err := os.WriteFile("nuevo.txt", []byte("archivo nuevo"), 0644); err != nil {
			t.Fatalf("Error creando archivo nuevo: %v", err)
		}

		// act
		diff, err := service.GetDiff(context.Background())

		// assert
		if err != nil {
			t.Errorf("Error obteniendo diff: %v", err)
		}

		if !strings.Contains(diff, "nuevo.txt") {
			t.Error("El diff no contiene el archivo nuevo")
		}
	})

	t.Run("GetDiff unchanged", func(t *testing.T) {
		// Arrange
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		// Act
		diff, err := service.GetDiff(context.Background())

		if err == nil {
			t.Error("Expected ErrNoDiff when there are no changes")
		}

		if !errors.Is(err, domainErrors.ErrNoDiff) {
			t.Errorf("Expected ErrNoDiff, got: %v", err)
		}

		if diff != "" {
			t.Error("El diff debería estar vacío cuando no hay cambios")
		}
	})
}

func TestAddFileToStaging(t *testing.T) {
	t.Run("AddNewFile", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()
		testFile := "newfile.txt"
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			return
		}

		err := service.AddFileToStaging(context.Background(), testFile)
		if err != nil {
			t.Fatalf("Error inesperado: %v", err)
		}

		cmd := exec.Command("git", "diff", "--cached", "--name-status")
		output, _ := cmd.Output()
		if !strings.Contains(string(output), "A\t"+testFile) {
			t.Error("Archivo nuevo no agregado correctamente")
		}
	})

	t.Run("Add deleted file", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()
		testFile := "deleted.txt"

		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			return
		}
		if err := service.AddFileToStaging(context.Background(), testFile); err != nil {
			t.Fatalf("Error al agregar archivo al staging: %v", err)
		}
		if err := service.CreateCommit(context.Background(), "Commit inicial"); err != nil {
			t.Fatalf("Error al crear commit inicial: %v", err)
		}

		if err := os.Remove(testFile); err != nil {
			return
		}

		if err := service.AddFileToStaging(context.Background(), testFile); err != nil {
			t.Fatalf("Error inesperado: %v", err)
		}

		cmd := exec.Command("git", "diff", "--cached", "--name-status")
		output, _ := cmd.Output()
		if !strings.Contains(string(output), "D\t"+testFile) {
			t.Error("Eliminación no registrada en staging")
		}
	})

	t.Run("Non-existent file", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()
		err := service.AddFileToStaging(context.Background(), "fantasma.txt")

		if err == nil {
			t.Error("Se esperaba error")
		}

		expectedMessages := []string{
			"did not match any files",
			"no concordó con ningún archivo",
		}

		match := false
		for _, msg := range expectedMessages {
			if strings.Contains(err.Error(), msg) {
				match = true
				break
			}
		}

		if !match {
			t.Errorf("Mensaje incorrecto. Se obtuvo: %v", err)
		}
	})

	t.Run("Add file in deleted directory", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()
		testFile := "dir1/dir2/archivos.txt"

		if err := os.MkdirAll(filepath.Dir(testFile), 0755); err != nil {
			return
		}
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			return
		}
		if err := service.AddFileToStaging(context.Background(), testFile); err != nil {
			t.Fatalf("Error al agregar archivo al staging: %v", err)
		}
		if err := service.CreateCommit(context.Background(), "Commit inicial"); err != nil {
			t.Fatalf("Error al crear commit inicial: %v", err)
		}

		if err := os.RemoveAll(filepath.Dir(testFile)); err != nil {
			return
		}

		if err := service.AddFileToStaging(context.Background(), testFile); err != nil {
			t.Fatalf("Error inesperado: %v", err)
		}

		cmd := exec.Command("git", "diff", "--cached", "--name-status")
		output, _ := cmd.Output()
		if !strings.Contains(string(output), "D\t"+testFile) {
			t.Error("La eliminación no se registró en staging")
		}
	})
}

func TestGetRepoInfo(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		expectedOwner    string
		expectedRepo     string
		expectedProvider string
		expectedError    bool
	}{
		{
			name:             "GitHub HTTPS URL",
			url:              "https://github.com/owner/repo.git",
			expectedOwner:    "owner",
			expectedRepo:     "repo",
			expectedProvider: "github",
			expectedError:    false,
		},
		{
			name:             "GitHub SSH URL",
			url:              "git@github.com:owner/repo.git",
			expectedOwner:    "owner",
			expectedRepo:     "repo",
			expectedProvider: "github",
			expectedError:    false,
		},
		{
			name:             "GitLab HTTPS URL",
			url:              "https://gitlab.com/owner/repo.git",
			expectedOwner:    "owner",
			expectedRepo:     "repo",
			expectedProvider: "gitlab",
			expectedError:    false,
		},
		{
			name:             "Invalid URL",
			url:              "invalid-url",
			expectedOwner:    "",
			expectedRepo:     "",
			expectedProvider: "",
			expectedError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, provider, err := parseRepoURL(tt.url)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOwner, owner)
				assert.Equal(t, tt.expectedRepo, repo)
				assert.Equal(t, tt.expectedProvider, provider)
			}
		})
	}
}

func TestGitService_NewMethods(t *testing.T) {
	createCommitHelper := func(t *testing.T, filename, message string) {
		if err := os.WriteFile(filename, []byte("content: "+message), 0644); err != nil {
			t.Fatalf("Error creando archivo %s: %v", filename, err)
		}
		if err := exec.Command("git", "add", filename).Run(); err != nil {
			t.Fatalf("Error haciendo stage de %s: %v", filename, err)
		}
		if err := exec.Command("git", "commit", "-m", message).Run(); err != nil {
			t.Fatalf("Error haciendo commit: %v", err)
		}
	}

	t.Run("GetLastTag", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		tag, err := service.GetLastTag(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "", tag, "Debería retornar string vacío si no hay tags")

		createCommitHelper(t, "file1.txt", "Initial commit")

		err = exec.Command("git", "tag", "-a", "v0.1.0", "-m", "Version 0.1.0").Run()
		assert.NoError(t, err)

		createCommitHelper(t, "file2.txt", "Second commit")
		err = exec.Command("git", "tag", "-a", "v1.0.0", "-m", "Version 1.0.0").Run()
		assert.NoError(t, err)

		tag, err = service.GetLastTag(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "v1.0.0", tag, "Debería retornar el último tag")
	})

	t.Run("GetCommitsSinceTag", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		createCommitHelper(t, "init.txt", "chore: initial commit")
		_ = exec.Command("git", "tag", "-a", "v1.0.0", "-m", "v1.0.0").Run()

		createCommitHelper(t, "feat.txt", "feat: amazing feature")
		createCommitHelper(t, "fix.txt", "fix: critical bug")

		commits, err := service.GetCommitsSinceTag(context.Background(), "v1.0.0")
		assert.NoError(t, err)
		assert.Len(t, commits, 2)
		assert.Contains(t, commits[0].Message, "fix: critical bug")
		assert.Contains(t, commits[1].Message, "feat: amazing feature")

		commits, err = service.GetCommitsSinceTag(context.Background(), "")
		assert.NoError(t, err)
		assert.Len(t, commits, 3) // feat, fix, chore
	})

	t.Run("CreateTag", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()
		createCommitHelper(t, "file.txt", "work done")

		err := service.CreateTag(context.Background(), "v2.0.0", "Release v2.0.0")
		assert.NoError(t, err)

		output, _ := exec.Command("git", "tag", "-l", "v2.0.0").Output()
		assert.Contains(t, string(output), "v2.0.0")
	})

	t.Run("GetCommitCount", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		createCommitHelper(t, "one.txt", "one")
		count, err := service.GetCommitCount(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 1, count)

		createCommitHelper(t, "two.txt", "two")
		count, err = service.GetCommitCount(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("PushTag", func(t *testing.T) {
		localDir := setupTestRepo(t)
		defer cleanupTestRepo(t, localDir)

		remoteDir, err := os.MkdirTemp("", "git-remote-test")
		if err != nil {
			t.Fatalf("Error creando dir remoto: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(remoteDir); err != nil {
				t.Fatalf("Error al eliminar directorio remoto: %v", err)
			}
		}()

		if err := exec.Command("git", "init", "--bare", remoteDir).Run(); err != nil {
			t.Fatalf("Error iniciando bare repo: %v", err)
		}

		if err := exec.Command("git", "remote", "add", "origin", remoteDir).Run(); err != nil {
			t.Fatalf("Error agregando remote: %v", err)
		}

		service := NewGitService()
		createCommitHelper(t, "code.txt", "ready to release")

		err = service.CreateTag(context.Background(), "v1.0.0", "Release")
		assert.NoError(t, err)

		err = service.PushTag(context.Background(), "v1.0.0")
		assert.NoError(t, err)

		cmd := exec.Command("git", "--git-dir", remoteDir, "tag")
		out, err := cmd.Output()
		assert.NoError(t, err)
		assert.Contains(t, string(out), "v1.0.0")
	})
}

func TestGitService_ErrorHandling(t *testing.T) {
	t.Run("GetDiff returns ErrNoDiff when no changes", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		testFile := "test.txt"
		if err := os.WriteFile(testFile, []byte("initial content"), 0644); err != nil {
			t.Fatalf("Error creating file: %v", err)
		}
		cmd := exec.Command("git", "add", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Error staging file: %v", err)
		}
		cmd = exec.Command("git", "commit", "-m", "initial commit")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Error committing: %v", err)
		}

		_, err := service.GetDiff(context.Background())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domainErrors.ErrNoDiff)
	})

	t.Run("CreateTag with error returns proper AppError with context", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		testFile := "test.txt"
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Error creating file: %v", err)
		}
		cmd := exec.Command("git", "add", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Error staging: %v", err)
		}
		cmd = exec.Command("git", "commit", "-m", "initial")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Error committing: %v", err)
		}

		err := service.CreateTag(context.Background(), "v1.0.0", "Release v1.0.0")
		assert.NoError(t, err)

		err = service.CreateTag(context.Background(), "v1.0.0", "Duplicate")
		assert.Error(t, err)

		var appErr *domainErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domainErrors.TypeGit, appErr.Type)
		assert.Equal(t, "v1.0.0", appErr.Context["version"])
	})

	t.Run("AddFileToStaging with non-existent file returns error with context", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		err := service.AddFileToStaging(context.Background(), "non-existent.txt")
		assert.Error(t, err)

		var appErr *domainErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domainErrors.TypeGit, appErr.Type)
		assert.Equal(t, "non-existent.txt", appErr.Context["file"])
		errMsg := err.Error()
		hasExpectedMsg := strings.Contains(errMsg, "did not match any files") ||
			strings.Contains(errMsg, "no concordó con ningún archivo")
		assert.True(t, hasExpectedMsg, "Expected error message about file not matching, got: %s", errMsg)
	})

	t.Run("GetCommitCount returns error with proper wrapping", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "not-git-repo")
		assert.NoError(t, err)
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Fatalf("Error removing temp dir: %v", err)
			}
		}()

		originalWd, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(originalWd); err != nil {
				t.Fatalf("Error changing working directory: %v", err)
			}
		}()

		err = os.Chdir(tempDir)
		assert.NoError(t, err)

		service := NewGitService()
		count, err := service.GetCommitCount(context.Background())
		assert.Error(t, err)
		assert.Equal(t, 0, count)

		var appErr *domainErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domainErrors.TypeGit, appErr.Type)
		assert.Contains(t, err.Error(), "failed to get commit count")
	})
}

// TestGitService_ValidateTagExists tests tag validation
func TestGitService_ValidateTagExists(t *testing.T) {
	t.Run("ValidateTagExists returns error for non-existent tag", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		testFile := "test.txt"
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Error creating file: %v", err)
		}
		cmd := exec.Command("git", "add", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Error staging: %v", err)
		}
		cmd = exec.Command("git", "commit", "-m", "test")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Error committing: %v", err)
		}

		err := service.ValidateTagExists(context.Background(), "v999.999.999")
		assert.Error(t, err)

		var appErr *domainErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, "v999.999.999", appErr.Context["tag"])
	})

	t.Run("ValidateTagExists succeeds for existing tag", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		testFile := "test.txt"
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Error creating file: %v", err)
		}
		cmd := exec.Command("git", "add", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Error staging: %v", err)
		}
		cmd = exec.Command("git", "commit", "-m", "test")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Error committing: %v", err)
		}

		err := service.CreateTag(context.Background(), "v1.0.0", "Test tag")
		assert.NoError(t, err)

		err = service.ValidateTagExists(context.Background(), "v1.0.0")
		assert.NoError(t, err)
	})
}

// TestGitService_GetRecentCommitMessages tests getting recent commit messages
func TestGitService_GetRecentCommitMessages(t *testing.T) {
	t.Run("GetRecentCommitMessages returns messages", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		commits := []string{"First commit", "Second commit", "Third commit"}
		for i, msg := range commits {
			testFile := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
			if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
				t.Fatalf("Error creating file: %v", err)
			}
			cmd := exec.Command("git", "add", ".")
			if err := cmd.Run(); err != nil {
				t.Fatalf("Error staging: %v", err)
			}
			cmd = exec.Command("git", "commit", "-m", msg)
			if err := cmd.Run(); err != nil {
				t.Fatalf("Error committing: %v", err)
			}
		}

		messages, err := service.GetRecentCommitMessages(context.Background(), 2)
		assert.NoError(t, err)
		assert.Len(t, messages, 2)
		assert.Contains(t, messages[0], "Third commit")
		assert.Contains(t, messages[1], "Second commit")
	})

	t.Run("GetRecentCommitMessages returns error with context when not in repo", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "not-git-repo")
		assert.NoError(t, err)
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Fatalf("Error removing temp dir: %v", err)
			}
		}()

		originalWd, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(originalWd); err != nil {
				t.Fatalf("Error changing working directory: %v", err)
			}
		}()

		err = os.Chdir(tempDir)
		assert.NoError(t, err)

		service := NewGitService()
		messages, err := service.GetRecentCommitMessages(context.Background(), 5)
		assert.Error(t, err)
		assert.Nil(t, messages)

		var appErr *domainErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domainErrors.TypeGit, appErr.Type)
		assert.Equal(t, 5, appErr.Context["count"])
	})
}

// TestGitService_GetGitUserInfo tests getting git user configuration
func TestGitService_GetGitUserInfo(t *testing.T) {
	t.Run("GetGitUserName returns configured name", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()
		name, err := service.GetGitUserName(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "Test User", name)
	})

	t.Run("GetGitUserEmail returns configured email", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()
		email, err := service.GetGitUserEmail(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "test@example.com", email)
	})

	t.Run("GetGitUserName returns error with context when not configured", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		cmd := exec.Command("git", "config", "--local", "--unset", "user.name")
		_ = cmd.Run()
		cmd = exec.Command("git", "config", "--global", "--unset", "user.name")
		_ = cmd.Run()

		cmd = exec.Command("git", "config", "user.name")
		_, err := cmd.Output()
		if err == nil {
			t.Skip("Cannot fully unset git user.name due to global/system config")
		}

		service := NewGitService()
		name, err := service.GetGitUserName(context.Background())
		if err != nil {
			assert.Empty(t, name)
			var appErr *domainErrors.AppError
			if errors.As(err, &appErr) {
				assert.Equal(t, "user.name", appErr.Context["config_key"])
			}
		}
	})
}

func TestGitService_GetTagDate(t *testing.T) {
	t.Run("GetTagDate returns date for existing tag", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		testFile := "test.txt"
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Error creating file: %v", err)
		}
		cmd := exec.Command("git", "add", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Error staging: %v", err)
		}
		cmd = exec.Command("git", "commit", "-m", "test")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Error committing: %v", err)
		}

		err := service.CreateTag(context.Background(), "v1.0.0", "Release")
		assert.NoError(t, err)

		date, err := service.GetTagDate(context.Background(), "v1.0.0")
		assert.NoError(t, err)
		assert.NotEmpty(t, date)
		assert.Len(t, date, 10)
	})

	t.Run("GetTagDate returns error with context for non-existent tag", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		date, err := service.GetTagDate(context.Background(), "v999.0.0")
		assert.Error(t, err)
		assert.Empty(t, date)

		var appErr *domainErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, "v999.0.0", appErr.Context["tag"])
	})
}

func TestGitService_Push(t *testing.T) {
	t.Run("Push returns error when no remote configured", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		err := service.Push(context.Background())
		assert.Error(t, err)

		var appErr *domainErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domainErrors.TypeGit, appErr.Type)
		assert.Contains(t, err.Error(), "failed to push to remote")
	})
}

func TestGitService_GetCommitsBetweenTags(t *testing.T) {
	t.Run("GetCommitsBetweenTags returns error with tag context", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		service := NewGitService()

		commits, err := service.GetCommitsBetweenTags(context.Background(), "v1.0.0", "v2.0.0")
		assert.Error(t, err)
		assert.Nil(t, commits)

		var appErr *domainErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, "v1.0.0", appErr.Context["from_tag"])
		assert.Equal(t, "v2.0.0", appErr.Context["to_tag"])
	})
}
