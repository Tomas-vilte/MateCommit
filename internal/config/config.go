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
		GeminiAPIKey     string `json:"gemini_api_key"`
		Language         string `json:"language"`
		UseEmoji         bool   `json:"use_emoji"`
		MaxLength        int    `json:"max_length"`
		SuggestionsCount int    `json:"suggestions_count"`
		PathFile         string `json:"path_file"`

		ActiveTicketService string     `json:"active_ticket_service,omitempty"` // "jira", "trello", "github", etc.
		JiraConfig          JiraConfig `json:"jira_config"`
		UseTicket           bool       `json:"use_ticket,omitempty"`
	}

	JiraConfig struct {
		APIKey  string `json:"api_key,omitempty"`
		BaseURL string `json:"base_url,omitempty"`
		Email   string `json:"email,omitempty"`
	}
)

const (
	defaultLang             = "en"
	defaultUseEmoji         = true
	defaultMaxLength        = 72
	defaultSuggestionsCount = 3
)

func LoadConfig(path string) (*Config, error) {
	var configPath string

	if filepath.Ext(path) == ".json" {
		configPath = path
	} else {
		configDir := filepath.Join(path, ".mate-commit")
		configPath = filepath.Join(configDir, "config.json")

		if _, err := os.Stat(configDir); os.IsNotExist(err) {
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return nil, fmt.Errorf("error al crear el directorio de configuración: %w", err)
			}
		}
	}

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
		Language:         defaultLang,
		UseEmoji:         defaultUseEmoji,
		MaxLength:        defaultMaxLength,
		SuggestionsCount: defaultSuggestionsCount,
		PathFile:         path,

		JiraConfig: JiraConfig{
			APIKey:  "",
			BaseURL: "",
			Email:   "",
		},
		ActiveTicketService: "",
		UseTicket:           false,
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

	if config.PathFile == "" {
		return errors.New("la ruta del archivo de configuración no está definida")
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error al codificar la configuración: %w", err)
	}

	if err := os.WriteFile(config.PathFile, data, 0644); err != nil {
		return fmt.Errorf("error al guardar la configuración: %w", err)
	}

	return nil
}

func validateConfig(config *Config) error {
	if config.MaxLength <= 0 {
		return errors.New("MaxLength debe ser mayor que 0")
	}
	if config.Language == "" {
		return errors.New("DefaultLang no puede estar vacío")
	}

	if config.ActiveTicketService != "" {
		switch config.ActiveTicketService {
		case "jira":
			if config.JiraConfig.BaseURL == "" {
				return errors.New("jira base URL no está configurada")
			}
			if config.JiraConfig.Email == "" {
				return errors.New("jira username no está configurado")
			}
			if config.JiraConfig.APIKey == "" {
				return errors.New("jira API key no está configurada")
			}
		default:
			return fmt.Errorf("servicio de tickets no soportado: %s", config.ActiveTicketService)
		}
	}
	return nil
}
