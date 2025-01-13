package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type (
	Config struct {
		GeminiAPIKey string `json:"gemini_api_key"`
		DefaultLang  string `json:"default_lang"`
		UseEmoji     bool   `json:"use_emoji"`
		MaxLength    int    `json:"max_length"`
		Format       string `json:"format"`
	}
)

func LoadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".mate-commit", "config.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return createDefaultConfig(configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func createDefaultConfig(path string) (*Config, error) {
	config := &Config{
		DefaultLang: "en",
		UseEmoji:    true,
		MaxLength:   72,
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, err
	}

	return config, nil
}

func SaveConfig(config *Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".mate-commit", "config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
