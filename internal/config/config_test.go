package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("debería manejar error al verificar existencia del archivo", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		configDir := filepath.Join(tmpDir, ".mate-commit")
		if err := os.MkdirAll(configDir, 0000); err != nil {
			t.Fatal(err)
		}

		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("Error al eliminar el archivo: %v", err)
			}
		}()

		_, err := LoadConfig(tmpDir)
		if err == nil {
			t.Error("se esperaba un error al verificar existencia del archivo")
		}

		err = os.Chmod(configDir, 0755)
		if err != nil {
			t.Fatal("No se pudo cambiar los permisos del directorio")
		}

		_, err = LoadConfig(tmpDir)
		if err != nil {
			t.Errorf("No se esperaba un error, pero ocurrió: %v", err)
		}
	})

	t.Run("debería manejar configuración inválida", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		// Crear directorio de configuración
		configDir := filepath.Join(tmpDir, ".mate-commit")
		err := os.MkdirAll(configDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		// Crear configuración inválida
		config := &Config{
			GeminiAPIKey: "key",
			Language:     "",
			MaxLength:    -1,
		}

		data, _ := json.MarshalIndent(config, "", "  ")
		err = os.WriteFile(filepath.Join(configDir, "config.json"), data, 0644)
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("Error al eliminar el archivo: %v", err)
			}
		}()

		_, err = LoadConfig(tmpDir) // Cargar desde el directorio base
		if err == nil {
			t.Error("se esperaba un error debido a configuración inválida")
		}
	})

	t.Run("debería manejar errores al guardar configuración", func(t *testing.T) {
		config := &Config{
			GeminiAPIKey: "key",
			MaxLength:    -1,
		}

		err := SaveConfig(config)
		if err == nil {
			t.Error("se esperaba un error al guardar configuración inválida, pero no ocurrió")
		}
	})

	t.Run("debería manejar JSON malformado", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		// Crear directorio de configuración
		configDir := filepath.Join(tmpDir, ".mate-commit")
		_ = os.MkdirAll(configDir, 0755)

		// Crear archivo con JSON malformado
		err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte("{malformed json"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("Error al eliminar el archivo: %v", err)
			}
		}()

		_, err = LoadConfig(tmpDir) // Cargar desde el directorio base
		if err == nil {
			t.Error("se esperaba un error al cargar JSON malformado")
		}
	})

}

func TestSaveConfig(t *testing.T) {
	t.Run("debería crear config.json en directorio .mate-commit si no existe", func(t *testing.T) {
		tmpDir := t.TempDir()
		expectedConfigDir := filepath.Join(tmpDir, ".mate-commit")
		expectedConfigPath := filepath.Join(expectedConfigDir, "config.json")

		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("Error al eliminar el archivo: %v", err)
			}
		}()

		// Act
		loadedConfig, err := LoadConfig(tmpDir)

		// Assert
		if err != nil {
			t.Errorf("LoadConfig() error = %v", err)
		}

		if _, err := os.Stat(expectedConfigDir); os.IsNotExist(err) {
			t.Error("El directorio .mate-commit no fue creado")
		}

		if _, err := os.Stat(expectedConfigPath); os.IsNotExist(err) {
			t.Error("El archivo config.json no fue creado")
		}

		if loadedConfig.Language != defaultLang {
			t.Errorf("DefaultLang = %v, want %v", loadedConfig.Language, defaultLang)
		}

		if loadedConfig.MaxLength != defaultMaxLength {
			t.Errorf("MaxLength = %v, want %v", loadedConfig.MaxLength, defaultMaxLength)
		}

		if loadedConfig.UseEmoji != defaultUseEmoji {
			t.Errorf("UseEmoji = %v, want %v", loadedConfig.UseEmoji, defaultUseEmoji)
		}
	})

	t.Run("debería manejar path directo a archivo json", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "direct-config.json")

		initialConfig := &Config{
			GeminiAPIKey: "test-key",
			Language:     "es",
			UseEmoji:     true,
			MaxLength:    50,
			PathFile:     configPath,
		}

		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("Error al eliminar el archivo: %v", err)
			}
		}()

		data, err := json.MarshalIndent(initialConfig, "", "  ")
		if err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatal(err)
		}

		// Act
		loadedConfig, err := LoadConfig(configPath)

		// Assert
		if err != nil {
			t.Errorf("LoadConfig() con path directo error = %v", err)
		}

		if loadedConfig.GeminiAPIKey != initialConfig.GeminiAPIKey {
			t.Errorf("GeminiAPIKey = %v, want %v", loadedConfig.GeminiAPIKey, initialConfig.GeminiAPIKey)
		}

		if loadedConfig.PathFile != configPath {
			t.Errorf("PathFile = %v, want %v", loadedConfig.PathFile, configPath)
		}
	})
	t.Run("debería manejar error al obtener home directory", func(t *testing.T) {
		t.Setenv("HOME", "")

		config := &Config{
			Language:  "en",
			MaxLength: 72,
		}

		err := SaveConfig(config)
		if err == nil {
			t.Error("se esperaba un error al no poder obtener el home directory")
		}
	})

	t.Run("debería manejar error al escribir archivo", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		configDir := filepath.Join(tmpDir, ".mate-commit")
		if err := os.MkdirAll(configDir, 0000); err != nil {
			t.Fatal(err)
		}

		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("Error al eliminar el archivo: %v", err)
			}
		}()

		config := &Config{
			Language:  "en",
			MaxLength: 72,
		}

		err := SaveConfig(config)
		if err == nil {
			t.Error("se esperaba un error al no poder escribir el archivo")
		}
	})

	t.Run("debería validar la configuración antes de guardar", func(t *testing.T) {
		config := &Config{
			Language:  "",
			MaxLength: 0,
		}

		err := SaveConfig(config)
		if err == nil {
			t.Error("se esperaba un error al guardar configuración inválida")
		}
	})

	t.Run("debería guardar la configuración correctamente", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		configDir := filepath.Join(tmpDir, ".mate-commit")
		err := os.MkdirAll(configDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("Error al eliminar el archivo: %v", err)
			}
		}()

		configPath := filepath.Join(configDir, "config.json")
		config := &Config{
			GeminiAPIKey: "new-key",
			Language:     "fr",
			UseEmoji:     false,
			MaxLength:    50,
			PathFile:     configPath,
		}

		// Act
		err = SaveConfig(config)

		// Assert
		if err != nil {
			t.Errorf("SaveConfig() error = %v", err)
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}

		var savedConfig Config
		if err := json.Unmarshal(data, &savedConfig); err != nil {
			t.Fatal(err)
		}

		if savedConfig.GeminiAPIKey != config.GeminiAPIKey {
			t.Errorf("Saved GeminiAPIKey = %v, want %v", savedConfig.GeminiAPIKey, config.GeminiAPIKey)
		}
		if savedConfig.Language != config.Language {
			t.Errorf("Saved DefaultLang = %v, want %v", savedConfig.Language, config.Language)
		}
		if savedConfig.MaxLength != config.MaxLength {
			t.Errorf("Saved MaxLength = %v, want %v", savedConfig.MaxLength, config.MaxLength)
		}
		if savedConfig.UseEmoji != config.UseEmoji {
			t.Errorf("Saved UseEmoji = %v, want %v", savedConfig.UseEmoji, config.UseEmoji)
		}
	})
}

func TestCreateDefaultConfig(t *testing.T) {
	t.Run("debería crear configuración por defecto correctamente", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		config, err := createDefaultConfig(configPath)

		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("Error al eliminar el archivo: %v", err)
			}
		}()

		if err != nil {
			t.Errorf("createDefaultConfig() error = %v, want nil", err)
		}

		if config.Language != defaultLang {
			t.Errorf("DefaultLang = %v, want %v", config.Language, defaultLang)
		}
		if config.UseEmoji != defaultUseEmoji {
			t.Errorf("UseEmoji = %v, want %v", config.UseEmoji, defaultUseEmoji)
		}
		if config.MaxLength != defaultMaxLength {
			t.Errorf("MaxLength = %v, want %v", config.MaxLength, defaultMaxLength)
		}
	})

	t.Run("debería manejar error al crear el directorio", func(t *testing.T) {
		invalidDir := filepath.Join(string([]byte{0}), "config.json")

		_, err := createDefaultConfig(invalidDir)

		if err == nil {
			t.Error("se esperaba un error al crear el directorio, pero no ocurrió")
		}
	})

	t.Run("debería manejar error al escribir archivo", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.Chmod(tmpDir, 0000); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := os.Chmod(tmpDir, 0755); err != nil {
				t.Fatal("No se pudo cambiar los permisos del directorio")
			}
		}()

		configPath := filepath.Join(tmpDir, "config.json")

		_, err := createDefaultConfig(configPath)
		if err == nil {
			t.Error("se esperaba un error al escribir el archivo")
		}
	})

	t.Run("debería manejar error al escribir archivo", func(t *testing.T) {
		tmpDir := t.TempDir()

		configDir := filepath.Join(tmpDir, "readonly")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.Chmod(configDir, 0444); err != nil {
			t.Fatal(err)
		}

		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				t.Errorf("Error al eliminar el archivo: %v", err)
			}
		}()

		configPath := filepath.Join(configDir, "config.json")
		config := &Config{
			GeminiAPIKey: "test-key",
			Language:     "es",
			MaxLength:    50,
			PathFile:     configPath,
		}

		// Act
		err := SaveConfig(config)

		// Assert
		if err == nil {
			t.Error("se esperaba un error al escribir en directorio sin permisos")
		}

		if err := os.Chmod(configDir, 0755); err != nil {
			t.Fatal("No se pudo restaurar los permisos del directorio")
		}
	})
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "configuración válida",
			config: &Config{
				Language:  "en",
				MaxLength: 72,
			},
			wantErr: false,
		},
		{
			name: "MaxLength inválido",
			config: &Config{
				Language:  "en",
				MaxLength: 0,
			},
			wantErr: true,
		},
		{
			name: "DefaultLang vacío",
			config: &Config{
				Language:  "",
				MaxLength: 72,
			},
			wantErr: true,
		},
		{
			name: "múltiples campos inválidos",
			config: &Config{
				Language:  "",
				MaxLength: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
