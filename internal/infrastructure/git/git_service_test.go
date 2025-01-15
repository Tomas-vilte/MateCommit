package git

import (
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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

		// Act - Verificar sin cambios staged
		hasStagedBefore := service.HasStagedChanges()

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
		hasStagedAfter := service.HasStagedChanges()

		// Assert
		if hasStagedBefore {
			t.Error("No debería haber cambios staged antes de agregar archivos")
		}
		if !hasStagedAfter {
			t.Error("Debería haber cambios staged después de agregar archivos")
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
		changes, err := service.GetChangedFiles()

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

		service := NewGitService()

		if err := os.WriteFile("test.txt", []byte("test content"), 0644); err != nil {
			t.Fatalf("Error creando archivo de prueba: %v", err)
		}

		if err := service.StageAllChanges(); err != nil {
			t.Fatalf("Error haciendo stage de los cambios: %v", err)
		}

		// act
		err := service.CreateCommit("Test Commit")

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
		err := service.CreateCommit("Test commit")

		// Assert
		if err == nil {
			t.Error("Se esperaba un error al crear commit sin cambios staged")
		}
		if err.Error() != "no hay cambios en el área de staging" {
			t.Errorf("Mensaje de error inesperado: %v", err)
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
		diff, err := service.GetDiff()

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
		diff, err := service.GetDiff()

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
		diff, err := service.GetDiff()

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
		diff, err := service.GetDiff()

		// Assert
		if err != nil {
			t.Errorf("Error obteniendo diff: %v", err)
		}

		if diff != "" {
			t.Error("El diff debería estar vacío cuando no hay cambios")
		}
	})
}
