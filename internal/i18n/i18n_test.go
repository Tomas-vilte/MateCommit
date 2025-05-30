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
		defer os.RemoveAll(tmpDir)

		createTestFile(t, tmpDir, "active.es.toml", `
		[HelloWorld]
		other = "¡Hola Mundo!"
		`)

		// act
		trans, err := NewTranslations("es", tmpDir)

		// assert
		if err != nil {
			t.Errorf("NewTranslations() no debería retornar error, obtuvo: %v", err)
		}

		if trans == nil {
			t.Error("NewTranslations() no debería retornar nil")
		}
	})

	t.Run("Should fail with empty language", func(t *testing.T) {
		// arrange
		tmpDir := createTempDir(t)
		defer os.RemoveAll(tmpDir)

		// act
		trans, err := NewTranslations("", tmpDir)

		// assert
		if err == nil {
			t.Error("NewTranslations() debería retornar error con idioma vacío")
		}

		if trans != nil {
			t.Error("NewTranslations() debería retornar nil cuando falla")
		}
	})
}

func TestSetLanguage(t *testing.T) {
	t.Run("Should change to a valid language", func(t *testing.T) {
		// arrange
		tmpDir := createTempDir(t)
		defer os.RemoveAll(tmpDir)

		createTestFile(t, tmpDir, "active.es.toml", `[Test]
		other = "Prueba"`)
		createTestFile(t, tmpDir, "active.en.toml", `[Test]
		other = "Test"`)

		trans, err := NewTranslations("en", tmpDir)
		if err != nil {
			t.Fatal("Error en la configuración de la prueba:", err)
		}

		// act
		err = trans.SetLanguage("es")

		// assert
		if err != nil {
			t.Errorf("SetLanguage() no debería retornar error, obtuvo: %v", err)
		}
	})

	t.Run("Should fail with unsupported language", func(t *testing.T) {
		// arrange
		tmpDir := createTempDir(t)
		defer os.RemoveAll(tmpDir)

		createTestFile(t, tmpDir, "active.es.toml", `[Test]
		other = "Prueba"`)

		trans, err := NewTranslations("es", tmpDir)
		if err != nil {
			t.Fatal("Error en la configuración de la prueba:", err)
		}

		// act
		err = trans.SetLanguage("fr")

		// asssert
		if err == nil {
			t.Error("SetLanguage() debería retornar error con idioma no soportado")
		}
	})
}

func TestGetMessage(t *testing.T) {
	t.Run("Should get singular message correctly", func(t *testing.T) {
		// arrange
		tmpDir := createTempDir(t)
		defer os.RemoveAll(tmpDir)

		createTestFile(t, tmpDir, "active.es.toml", `
		[Welcome]
		one = "Bienvenido"
		other = "Bienvenidos"`)

		trans, err := NewTranslations("es", tmpDir)
		if err != nil {
			t.Fatal("Error en la configuración de la prueba:", err)
		}

		// act
		result := trans.GetMessage("Welcome", 1, nil)

		// assert
		expected := "Bienvenido"
		if result != expected {
			t.Errorf("GetMessage() = %v, quiere %v", result, expected)
		}
	})

	t.Run("Should get plural message correctly", func(t *testing.T) {
		// arrange
		tmpDir := createTempDir(t)
		defer os.RemoveAll(tmpDir)

		createTestFile(t, tmpDir, "active.es.toml", `
		[Welcome]
		one = "Bienvenido"
		other = "Bienvenidos"`)

		trans, err := NewTranslations("es", tmpDir)
		if err != nil {
			t.Fatal("Error en la configuración de la prueba:", err)
		}

		// act
		result := trans.GetMessage("Welcome", 2, nil)

		// assert
		expected := "Bienvenidos"
		if result != expected {
			t.Errorf("GetMessage() = %v, quiere %v", result, expected)
		}
	})

	t.Run("Should handle templates correctly", func(t *testing.T) {
		// arrange
		tmpDir := createTempDir(t)
		defer os.RemoveAll(tmpDir)

		createTestFile(t, tmpDir, "active.es.toml", `
		[HelloName]
		other = "¡Hola {{.Name}}!"`)

		trans, err := NewTranslations("es", tmpDir)
		if err != nil {
			t.Fatal("Error en la configuración de la prueba:", err)
		}

		templateData := map[string]interface{}{
			"Name": "Juan",
		}

		// act
		result := trans.GetMessage("HelloName", 0, templateData)

		// assert
		expected := "¡Hola Juan!"
		if result != expected {
			t.Errorf("GetMessage() = %v, quiere %v", result, expected)
		}
	})

	t.Run("Should handle missing messages", func(t *testing.T) {
		// arrange
		tmpDir := createTempDir(t)
		defer os.RemoveAll(tmpDir)

		createTestFile(t, tmpDir, "active.es.toml", `[Test]
		other = "Prueba"`)

		trans, err := NewTranslations("es", tmpDir)
		if err != nil {
			t.Fatal("Error en la configuración de la prueba:", err)
		}

		// act
		result := trans.GetMessage("NonExistent", 1, nil)

		// assert
		expected := "Translation missing: NonExistent"
		if result != expected {
			t.Errorf("GetMessage() = %v, quiere %v", result, expected)
		}
	})
}

func TestNewTranslations_Errors(t *testing.T) {
	t.Run("Should successfully load multiple translation files", func(t *testing.T) {
		// Arrange
		tmpDir := createTempDir(t)
		defer os.RemoveAll(tmpDir)

		// Crear múltiples archivos válidos
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
			t.Errorf("NewTranslations() no debería fallar con archivos válidos: %v", err)
		}
		if trans == nil {
			t.Error("NewTranslations() no debería retornar nil con archivos válidos")
		}
	})
}

func createTempDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "i18n_test")
	if err != nil {
		t.Fatal("No se pudo crear el directorio temporal:", err)
	}
	return tmpDir
}

func createTestFile(t *testing.T, dir, filename, content string) {
	err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
	if err != nil {
		t.Fatal("No se pudo crear el archivo de prueba:", err)
	}
}
