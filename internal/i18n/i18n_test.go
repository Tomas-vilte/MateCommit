package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTranslations(t *testing.T) {
	t.Run("Should successfully create translations with valid language", func(t *testing.T) {
		// Arrange
		tmpDir := createTempDir(t)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("error deleting directory: %v", err)
			}
		}()

		createTestFile(t, tmpDir, "active.es.toml", `
		[HelloWorld]
		other = "¡Hola Mundo!"
		`)

		// act
		trans, err := NewTranslations("es", tmpDir)

		// assert
		if err != nil {
			t.Errorf("NewTranslations() should not return an error, got: %v", err)
		}

		if trans == nil {
			t.Error("NewTranslations() should not return nil")
		}
	})

	t.Run("Should fail with empty language", func(t *testing.T) {
		// arrange
		tmpDir := createTempDir(t)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("error deleting directory: %v", err)
			}
		}()

		// act
		trans, err := NewTranslations("", tmpDir)

		// assert
		if err == nil {
			t.Error("NewTranslations() should return error with empty language")
		}

		if trans != nil {
			t.Error("NewTranslations() should return nil when it fails")
		}
	})
}

func TestSetLanguage(t *testing.T) {
	t.Run("Should change to a valid language", func(t *testing.T) {
		// arrange
		tmpDir := createTempDir(t)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("error deleting directory: %v", err)
			}
		}()

		createTestFile(t, tmpDir, "active.es.toml", `[Test]
		other = "Prueba"`)
		createTestFile(t, tmpDir, "active.en.toml", `[Test]
		other = "Test"`)

		trans, err := NewTranslations("en", tmpDir)
		if err != nil {
			t.Fatal("Error in test setup:", err)
		}

		// act
		err = trans.SetLanguage("es")

		// assert
		if err != nil {
			t.Errorf("SetLanguage() should not return error, got: %v", err)
		}
	})

	t.Run("Should fail with unsupported language", func(t *testing.T) {
		// arrange
		tmpDir := createTempDir(t)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("error deleting directory: %v", err)
			}
		}()

		createTestFile(t, tmpDir, "active.es.toml", `[Test]
		other = "Prueba"`)

		trans, err := NewTranslations("es", tmpDir)
		if err != nil {
			t.Fatal("Error in test setup:", err)
		}

		// act
		err = trans.SetLanguage("fr")

		// asssert
		if err == nil {
			t.Error("SetLanguage() should return error with unsupported language")
		}
	})
}

func TestGetMessage(t *testing.T) {
	t.Run("Should get singular message correctly", func(t *testing.T) {
		// arrange
		tmpDir := createTempDir(t)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("error deleting directory: %v", err)
			}
		}()

		createTestFile(t, tmpDir, "active.es.toml", `
		[Welcome]
		one = "Bienvenido"
		other = "Bienvenidos"`)

		trans, err := NewTranslations("es", tmpDir)
		if err != nil {
			t.Fatal("Error in test setup:", err)
		}

		// act
		result := trans.GetMessage("Welcome", 1, nil)

		// assert
		expected := "Bienvenido"
		if result != expected {
			t.Errorf("GetMessage() = %v, want %v", result, expected)
		}
	})

	t.Run("Should get plural message correctly", func(t *testing.T) {
		// arrange
		tmpDir := createTempDir(t)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("error deleting directory: %v", err)
			}
		}()

		createTestFile(t, tmpDir, "active.es.toml", `
		[Welcome]
		one = "Bienvenido"
		other = "Bienvenidos"`)

		trans, err := NewTranslations("es", tmpDir)
		if err != nil {
			t.Fatal("Error in test setup:", err)
		}

		// act
		result := trans.GetMessage("Welcome", 2, nil)

		// assert
		expected := "Bienvenidos"
		if result != expected {
			t.Errorf("GetMessage() = %v, want %v", result, expected)
		}
	})

	t.Run("Should handle templates correctly", func(t *testing.T) {
		// arrange
		tmpDir := createTempDir(t)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("error deleting directory: %v", err)
			}
		}()

		createTestFile(t, tmpDir, "active.es.toml", `
		[HelloName]
		other = "¡Hola {{.Name}}!"`)

		trans, err := NewTranslations("es", tmpDir)
		if err != nil {
			t.Fatal("Error in test setup:", err)
		}

		templateData := map[string]interface{}{
			"Name": "Juan",
		}

		// act
		result := trans.GetMessage("HelloName", 0, templateData)

		// assert
		expected := "¡Hola Juan!"
		if result != expected {
			t.Errorf("GetMessage() = %v, want %v", result, expected)
		}
	})

	t.Run("Should handle missing messages", func(t *testing.T) {
		// arrange
		tmpDir := createTempDir(t)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("error deleting directory: %v", err)
			}
		}()

		createTestFile(t, tmpDir, "active.es.toml", `[Test]
		other = "Prueba"`)

		trans, err := NewTranslations("es", tmpDir)
		if err != nil {
			t.Fatal("Error in test setup:", err)
		}

		// act
		result := trans.GetMessage("NonExistent", 1, nil)

		// assert
		expected := "Missing translation: NonExistent"
		if result != expected {
			t.Errorf("GetMessage() = %v, want %v", result, expected)
		}
	})
}

func TestNewTranslations_Errors(t *testing.T) {
	t.Run("Should successfully load multiple translation files", func(t *testing.T) {
		// Arrange
		tmpDir := createTempDir(t)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("error deleting directory: %v", err)
			}
		}()

		// Create multiple valid files
		createTestFile(t, tmpDir, "active.es.toml", `
		[Hello]
		other = "Hola"`)

		createTestFile(t, tmpDir, "active.en.toml", `
		[Hello]
		other = "Hello"`)

		// Act
		trans, err := NewTranslations("es", tmpDir)

		// Assert
		if err != nil {
			t.Errorf("NewTranslations() should not fail with valid files: %v", err)
		}
		if trans == nil {
			t.Error("NewTranslations() should not return nil with valid files")
		}
	})
}

func createTempDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "i18n_test")
	if err != nil {
		t.Fatal("Could not create temporary directory:", err)
	}
	return tmpDir
}

func createTestFile(t *testing.T, dir, filename, content string) {
	err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
	if err != nil {
		t.Fatal("Could not create test file:", err)
	}
}
