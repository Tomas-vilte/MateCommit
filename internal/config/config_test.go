package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("debería manejar error al obtener home directory", func(t *testing.T) {
		t.Setenv("HOME", "")

		_, err := LoadConfig()
		if err == nil {
			t.Error("se esperaba un error al no poder obtener el home directory")
		}
	})

	t.Run("debería manejar error al verificar existencia del archivo", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		configDir := filepath.Join(tmpDir, ".mate-commit")
		if err := os.MkdirAll(configDir, 0000); err != nil {
			t.Fatal(err)
		}

		_, err := LoadConfig()
		if err == nil {
			t.Error("se esperaba un error al verificar existencia del archivo")
		}

		err = os.Chmod(configDir, 0755)
		if err != nil {
			t.Fatal("No se pudo cambiar los permisos del directorio")
		}
	})

	t.Run("debería manejar configuración inválida", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		configDir := filepath.Join(tmpDir, ".mate-commit")
		err := os.MkdirAll(configDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		config := &Config{
			GeminiAPIKey: "key",
			DefaultLang:  "",
			MaxLength:    -1,
		}

		data, _ := json.MarshalIndent(config, "", "  ")
		err = os.WriteFile(filepath.Join(configDir, "config.json"), data, 0644)
		if err != nil {
			t.Fatal(err)
		}

		_, err = LoadConfig()
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

		configDir := filepath.Join(tmpDir, ".mate-commit")
		_ = os.MkdirAll(configDir, 0755)

		err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte("{malformed json"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		_, err = LoadConfig()
		if err == nil {
			t.Error("se esperaba un error al cargar JSON malformado")
		}
	})

}

func TestSaveConfig(t *testing.T) {
	t.Run("debería manejar error al obtener home directory", func(t *testing.T) {
		t.Setenv("HOME", "")

		config := &Config{
			DefaultLang: "en",
			MaxLength:   72,
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

		config := &Config{
			DefaultLang: "en",
			MaxLength:   72,
		}

		err := SaveConfig(config)
		if err == nil {
			t.Error("se esperaba un error al no poder escribir el archivo")
		}
	})

	t.Run("debería validar la configuración antes de guardar", func(t *testing.T) {
		config := &Config{
			DefaultLang: "",
			MaxLength:   0,
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

		config := &Config{
			GeminiAPIKey: "new-key",
			DefaultLang:  "fr",
			UseEmoji:     false,
			MaxLength:    50,
			Format:       "conventional",
		}

		// Act
		err = SaveConfig(config)

		// Assert
		if err != nil {
			t.Errorf("SaveConfig() error = %v", err)
		}

		// Verificar que el archivo se guardó correctamente
		data, err := os.ReadFile(filepath.Join(tmpDir, ".mate-commit", "config.json"))
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
		if savedConfig.DefaultLang != config.DefaultLang {
			t.Errorf("Saved DefaultLang = %v, want %v", savedConfig.DefaultLang, config.DefaultLang)
		}
	})
}

func TestCreateDefaultConfig(t *testing.T) {
	t.Run("debería crear configuración por defecto correctamente", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		config, err := createDefaultConfig(configPath)

		if err != nil {
			t.Errorf("createDefaultConfig() error = %v, want nil", err)
		}

		if config.DefaultLang != defaultLang {
			t.Errorf("DefaultLang = %v, want %v", config.DefaultLang, defaultLang)
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
				DefaultLang: "en",
				MaxLength:   72,
			},
			wantErr: false,
		},
		{
			name: "MaxLength inválido",
			config: &Config{
				DefaultLang: "en",
				MaxLength:   0,
			},
			wantErr: true,
		},
		{
			name: "DefaultLang vacío",
			config: &Config{
				DefaultLang: "",
				MaxLength:   72,
			},
			wantErr: true,
		},
		{
			name: "múltiples campos inválidos",
			config: &Config{
				DefaultLang: "",
				MaxLength:   -1,
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
