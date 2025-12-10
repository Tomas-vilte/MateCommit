package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
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

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)

		// Act - Verificar sin cambios staged
		hasStagedBefore := service.HasStagedChanges(context.Background())

		// Crear y hacer stage de un archivo
		testFile := filepath.Join("test.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatalf("Error creando archivo de prueba: %v", err)
		}

		// Stage el archivo
		cmd := exec.Command("git", "add", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Error haciendo stage del archivo: %v", err)
		}

		// Act - Verificar con cambios staged
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

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)

		// Act - Obtener la branch actual (debería ser 'main' por defecto)
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

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)

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
			expectedChange := models.GitChange{
				Path:   "test.txt",
				Status: "??",
			}

			if changes[0].Path != expectedChange.Path {
				t.Errorf("Path esperado %s, se obtuvo %s", expectedChange.Path, changes[0].Path)
			}

			if changes[0].Status != expectedChange.Status {
				t.Errorf("Status esperado %s, se obtuvo %s", expectedChange.Status, changes[0].Status)
			}
		}
	})

	t.Run("CreateCommit", func(t *testing.T) {
		// arrange
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)

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

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)

		// Act
		err := service.CreateCommit(context.Background(), "Test commit")

		// Assert
		if err == nil {
			t.Error("Se esperaba un error al crear commit sin cambios staged")
		}
		if err.Error() != "No staged changes found" {
			t.Errorf("Mensaje de error inesperado: %v", err)
		}
	})

	t.Run("GetDiff with staged files", func(t *testing.T) {
		// arrange
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)

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

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)

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

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)

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

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)

		// Act
		diff, err := service.GetDiff(context.Background())

		// Assert
		if err != nil {
			t.Errorf("Error obteniendo diff: %v", err)
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

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)
		testFile := "newfile.txt"
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			return
		}

		err := service.AddFileToStaging(context.Background(), testFile)
		if err != nil {
			t.Fatalf("Error inesperado: %v", err)
		}

		// verificar staging
		cmd := exec.Command("git", "diff", "--cached", "--name-status")
		output, _ := cmd.Output()
		if !strings.Contains(string(output), "A\t"+testFile) {
			t.Error("Archivo nuevo no agregado correctamente")
		}
	})

	t.Run("Add deleted file", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)
		testFile := "deleted.txt"

		// Crear y committear archivo
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			return
		}
		if err := service.AddFileToStaging(context.Background(), testFile); err != nil {
			t.Fatalf("Error al agregar archivo al staging: %v", err)
		}
		if err := service.CreateCommit(context.Background(), "Commit inicial"); err != nil {
			t.Fatalf("Error al crear commit inicial: %v", err)
		}

		// Eliminar y agregar al staging
		if err := os.Remove(testFile); err != nil {
			return
		}

		if err := service.AddFileToStaging(context.Background(), testFile); err != nil {
			t.Fatalf("Error inesperado: %v", err)
		}

		// Verificar eliminación en staging
		cmd := exec.Command("git", "diff", "--cached", "--name-status")
		output, _ := cmd.Output()
		if !strings.Contains(string(output), "D\t"+testFile) {
			t.Error("Eliminación no registrada en staging")
		}
	})

	t.Run("Non-existent file", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)
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

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)
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

	trans, _ := i18n.NewTranslations("en", "")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, provider, err := parseRepoURL(tt.url, trans)

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

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)

		// Caso 1: Sin tags
		tag, err := service.GetLastTag(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "", tag, "Debería retornar string vacío si no hay tags")

		// Caso 2: Con tags
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

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)

		// Crear historial
		createCommitHelper(t, "init.txt", "chore: initial commit")
		_ = exec.Command("git", "tag", "-a", "v1.0.0", "-m", "v1.0.0").Run()

		createCommitHelper(t, "feat.txt", "feat: amazing feature")
		createCommitHelper(t, "fix.txt", "fix: critical bug")

		// Caso 1: Commits desde v1.0.0
		commits, err := service.GetCommitsSinceTag(context.Background(), "v1.0.0")
		assert.NoError(t, err)
		assert.Len(t, commits, 2)
		// Git log suele devolver en orden cronológico inverso (el más reciente primero)
		assert.Contains(t, commits[0].Message, "fix: critical bug")
		assert.Contains(t, commits[1].Message, "feat: amazing feature")

		// Caso 2: Sin tag previo (debería traer todos)
		commits, err = service.GetCommitsSinceTag(context.Background(), "")
		assert.NoError(t, err)
		assert.Len(t, commits, 3) // feat, fix, chore
	})

	t.Run("CreateTag", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)
		createCommitHelper(t, "file.txt", "work done")

		err := service.CreateTag(context.Background(), "v2.0.0", "Release v2.0.0")
		assert.NoError(t, err)

		// Verificar que el tag existe
		output, _ := exec.Command("git", "tag", "-l", "v2.0.0").Output()
		assert.Contains(t, string(output), "v2.0.0")
	})

	t.Run("GetCommitCount", func(t *testing.T) {
		tempDir := setupTestRepo(t)
		defer cleanupTestRepo(t, tempDir)

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)

		// Repositorio vacío (recién inicializado, sin commits) puede dar error o 0 dependiendo de la versión de git/setup
		// Vamos a crear al menos uno para asegurar
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
		// Setup local repo
		localDir := setupTestRepo(t)
		defer cleanupTestRepo(t, localDir)

		// Setup bare remote repo to simulate origin
		remoteDir, err := os.MkdirTemp("", "git-remote-test")
		if err != nil {
			t.Fatalf("Error creando dir remoto: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(remoteDir); err != nil {
				t.Fatalf("Error al eliminar directorio remoto: %v", err)
			}
		}()

		// Init bare repo
		if err := exec.Command("git", "init", "--bare", remoteDir).Run(); err != nil {
			t.Fatalf("Error iniciando bare repo: %v", err)
		}

		// Add remote to local
		if err := exec.Command("git", "remote", "add", "origin", remoteDir).Run(); err != nil {
			t.Fatalf("Error agregando remote: %v", err)
		}

		trans, _ := i18n.NewTranslations("en", "")
		service := NewGitService(trans)
		createCommitHelper(t, "code.txt", "ready to release")

		// Push antes de tener el tag debería fallar o no hacer nada relevante, primero creamos el tag
		err = service.CreateTag(context.Background(), "v1.0.0", "Release")
		assert.NoError(t, err)

		// Test PushTag
		err = service.PushTag(context.Background(), "v1.0.0")
		assert.NoError(t, err)

		// Verificar en el remoto que el tag existe
		cmd := exec.Command("git", "--git-dir", remoteDir, "tag")
		out, err := cmd.Output()
		assert.NoError(t, err)
		assert.Contains(t, string(out), "v1.0.0")
	})
}
