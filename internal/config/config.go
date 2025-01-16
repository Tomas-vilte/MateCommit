package config

import (
	"encoding/json"
	"errors"
	"fmt"
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

const (
	defaultLang      = "en"
	defaultUseEmoji  = true
	defaultMaxLength = 72
)

func LoadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("no se pudo obtener el directorio del usuario: %w", err)
	}

	configPath := filepath.Join(homeDir, ".mate-commit", "config.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return createDefaultConfig(configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error al leer el archivo de configuración: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error al decodificar el archivo JSON: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("la configuración cargada no es válida: %w", err)
	}

	return &config, nil
}

func createDefaultConfig(path string) (*Config, error) {
	config := &Config{
		DefaultLang: defaultLang,
		UseEmoji:    defaultUseEmoji,
		MaxLength:   defaultMaxLength,
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("error al crear el directorio de configuración: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error al codificar la configuración por defecto: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, fmt.Errorf("error al guardar la configuración por defecto: %w", err)
	}

	return config, nil
}

func SaveConfig(config *Config) error {
	if err := validateConfig(config); err != nil {
		return fmt.Errorf("la configuración a guardar no es válida: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("no se pudo obtener el directorio del usuario: %w", err)
	}

	configPath := filepath.Join(homeDir, ".mate-commit", "config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error al codificar la configuración: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("error al guardar la configuración: %w", err)
	}

	return nil
}

func validateConfig(config *Config) error {
	if config.MaxLength <= 0 {
		return errors.New("MaxLength debe ser mayor que 0")
	}
	if config.DefaultLang == "" {
		return errors.New("DefaultLang no puede estar vacío")
	}
	return nil
}
