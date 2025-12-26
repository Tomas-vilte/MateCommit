package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("should handle error when checking file existence", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		configDir := filepath.Join(tmpDir, ".config")
		if err := os.MkdirAll(configDir, 0000); err != nil {
			t.Fatal(err)
		}

		defer func() {
			_ = os.Chmod(configDir, 0755)
		}()

		_, err := LoadConfig(tmpDir)
		if err == nil {
			t.Error("expected an error when checking file existence")
		}

		err = os.Chmod(configDir, 0755)
		if err != nil {
			t.Fatal("Could not change directory permissions")
		}

		_, err = LoadConfig(tmpDir)
		if err != nil {
			t.Errorf("Did not expect an error, but one occurred: %v", err)
		}
	})

	t.Run("should handle invalid configuration", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		configDir := filepath.Join(tmpDir, ".config", "matecommit")
		err := os.MkdirAll(configDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		config := &Config{
			Language: "",
		}

		data, _ := json.MarshalIndent(config, "", "  ")
		err = os.WriteFile(filepath.Join(configDir, "config.json"), data, 0644)
		if err != nil {
			t.Fatal(err)
		}

		_, err = LoadConfig(tmpDir)
		if err == nil {
			t.Error("expected an error due to invalid configuration")
		}
	})

	t.Run("should handle errors when saving configuration", func(t *testing.T) {
		config := &Config{
			Language: "",
		}

		err := SaveConfig(config)
		if err == nil {
			t.Error("expected an error when saving invalid configuration, but none occurred")
		}
	})

	t.Run("should handle malformed JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		configDir := filepath.Join(tmpDir, ".config", "matecommit")
		_ = os.MkdirAll(configDir, 0755)

		err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte("{malformed json"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		_, err = LoadConfig(tmpDir)
		if err == nil {
			t.Error("expected an error when loading malformed JSON")
		}
	})

}

func TestSaveConfig(t *testing.T) {
	t.Run("should create config.json in .config/matecommit directory if it doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		expectedConfigDir := filepath.Join(tmpDir, ".config", "matecommit")
		expectedConfigPath := filepath.Join(expectedConfigDir, "config.json")

		// Act
		loadedConfig, err := LoadConfig(tmpDir)

		// Assert
		if err != nil {
			t.Errorf("LoadConfig() error = %v", err)
		}

		if _, err := os.Stat(expectedConfigDir); os.IsNotExist(err) {
			t.Error(".config/matecommit directory was not created")
		}

		if _, err := os.Stat(expectedConfigPath); os.IsNotExist(err) {
			t.Error("config.json file was not created")
		}

		if loadedConfig.Language != defaultLang {
			t.Errorf("DefaultLang = %v, want %v", loadedConfig.Language, defaultLang)
		}

		if loadedConfig.UseEmoji != defaultUseEmoji {
			t.Errorf("UseEmoji = %v, want %v", loadedConfig.UseEmoji, defaultUseEmoji)
		}
	})

	t.Run("should handle direct path to json file", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "direct-config.json")

		initialConfig := &Config{
			Language: "en",
			UseEmoji: true,
			PathFile: configPath,
			AIProviders: map[string]AIProviderConfig{
				"gemini": {
					APIKey:      "test-key",
					Model:       "gemini-2.5-flash",
					Temperature: 0.3,
					MaxTokens:   10000,
				},
			},
		}

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
			t.Errorf("LoadConfig() with direct path error = %v", err)
		}

		if providerCfg, exists := loadedConfig.AIProviders["gemini"]; !exists || providerCfg.APIKey != "test-key" {
			t.Errorf("Gemini APIKey not properly loaded")
		}

		if loadedConfig.PathFile != configPath {
			t.Errorf("PathFile = %v, want %v", loadedConfig.PathFile, configPath)
		}
	})

	t.Run("should handle error when getting home directory", func(t *testing.T) {
		t.Setenv("HOME", "")

		config := &Config{
			Language: "en",
		}

		err := SaveConfig(config)
		if err == nil {
			t.Error("expected an error when home directory could not be obtained")
		}
	})

	t.Run("should handle error when writing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		configDir := filepath.Join(tmpDir, ".config")
		if err := os.MkdirAll(configDir, 0000); err != nil {
			t.Fatal(err)
		}

		defer func() {
			_ = os.Chmod(configDir, 0755)
		}()

		config := &Config{
			Language: "en",
		}

		err := SaveConfig(config)
		if err == nil {
			t.Error("expected an error when file could not be written")
		}
	})

	t.Run("should validate configuration before saving", func(t *testing.T) {
		config := &Config{
			Language: "",
		}

		err := SaveConfig(config)
		if err == nil {
			t.Error("expected an error when saving invalid configuration")
		}
	})

	t.Run("should save configuration correctly", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		configDir := filepath.Join(tmpDir, ".config", "matecommit")
		err := os.MkdirAll(configDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		configPath := filepath.Join(configDir, "config.json")
		config := &Config{
			Language: "fr",
			UseEmoji: false,
			PathFile: configPath,
			AIProviders: map[string]AIProviderConfig{
				"gemini": {
					APIKey:      "new-key",
					Model:       "gemini-2.5-flash",
					Temperature: 0.3,
					MaxTokens:   10000,
				},
			},
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

		if providerCfg, exists := savedConfig.AIProviders["gemini"]; !exists || providerCfg.APIKey != "new-key" {
			t.Errorf("Gemini APIKey not properly saved")
		}
		if savedConfig.Language != config.Language {
			t.Errorf("Saved DefaultLang = %v, want %v", savedConfig.Language, config.Language)
		}

		if savedConfig.UseEmoji != config.UseEmoji {
			t.Errorf("Saved UseEmoji = %v, want %v", savedConfig.UseEmoji, config.UseEmoji)
		}
	})
}

func TestCreateDefaultConfig(t *testing.T) {
	t.Run("should create default configuration correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		config, err := createDefaultConfig(configPath)

		if err != nil {
			t.Errorf("createDefaultConfig() error = %v, want nil", err)
		}

		if config.Language != defaultLang {
			t.Errorf("DefaultLang = %v, want %v", config.Language, defaultLang)
		}
		if config.UseEmoji != defaultUseEmoji {
			t.Errorf("UseEmoji = %v, want %v", config.UseEmoji, defaultUseEmoji)
		}
	})

	t.Run("should handle error when creating directory", func(t *testing.T) {
		invalidDir := filepath.Join(string([]byte{0}), "config.json")

		_, err := createDefaultConfig(invalidDir)

		if err == nil {
			t.Error("expected an error when creating the directory, but none occurred")
		}
	})

	t.Run("should handle error when writing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.Chmod(tmpDir, 0000); err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = os.Chmod(tmpDir, 0755)
		}()

		configPath := filepath.Join(tmpDir, "config.json")

		_, err := createDefaultConfig(configPath)
		if err == nil {
			t.Error("expected an error when writing file")
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
			name: "valid configuration",
			config: &Config{
				Language: "en",
			},
			wantErr: false,
		},

		{
			name: "empty language",
			config: &Config{
				Language: "",
			},
			wantErr: true,
		},
		{
			name: "multiple invalid fields",
			config: &Config{
				Language: "",
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
