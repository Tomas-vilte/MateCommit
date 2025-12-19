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
		Language         string `json:"language"`
		UseEmoji         bool   `json:"use_emoji"`
		SuggestionsCount int    `json:"suggestions_count"`
		PathFile         string `json:"path_file"`

		AIProviders map[string]AIProviderConfig `json:"ai_providers,omitempty"`
		AIConfig    AIConfig                    `json:"ai_config"`

		TicketProviders     map[string]TicketProviderConfig `json:"ticket_providers,omitempty"`
		ActiveTicketService string                          `json:"active_ticket_service,omitempty"`
		UseTicket           bool                            `json:"use_ticket,omitempty"`

		VCSConfigs        map[string]VCSConfig `json:"vcs_configs"`
		ActiveVCSProvider string               `json:"active_vcs_provider,omitempty"`
		UpdateChangelog   bool                 `json:"update_changelog"`
		VersionFile       string               `json:"version_file,omitempty"`
		VersionPattern    string               `json:"version_pattern,omitempty"`
	}

	AIProviderConfig struct {
		APIKey      string  `json:"api_key"`
		Model       string  `json:"model,omitempty"`
		Temperature float32 `json:"temperature,omitempty"`
		MaxTokens   int     `json:"max_tokens,omitempty"`
	}

	TicketProviderConfig struct {
		APIKey   string            `json:"api_key"`
		BaseURL  string            `json:"base_url,omitempty"`
		Email    string            `json:"email,omitempty"`
		Username string            `json:"username,omitempty"`
		Extra    map[string]string `json:"extra,omitempty"`
	}

	AIConfig struct {
		ActiveAI    AI           `json:"active_ai"`
		Models      map[AI]Model `json:"models"`
		BudgetDaily *float64     `json:"budget_daily,omitempty"`
	}

	VCSConfig struct {
		Provider string `json:"provider"` // github o gitlab lo que se te cante
		Token    string `json:"token,omitempty"`
		Owner    string `json:"owner,omitempty"`
		Repo     string `json:"repo,omitempty"`
	}
)

const (
	defaultLang             = "en"
	defaultUseEmoji         = true
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

	config.PathFile = configPath
	return &config, nil
}

func createDefaultConfig(path string) (*Config, error) {
	config := &Config{
		Language:         defaultLang,
		UseEmoji:         defaultUseEmoji,
		SuggestionsCount: defaultSuggestionsCount,
		UpdateChangelog:  false,
		VersionFile:      "",
		VersionPattern:   "",
		PathFile:         path,
		AIConfig: AIConfig{
			ActiveAI: AIGemini,
			Models:   make(map[AI]Model),
		},

		AIProviders:     make(map[string]AIProviderConfig),
		TicketProviders: make(map[string]TicketProviderConfig),
		VCSConfigs:      make(map[string]VCSConfig),

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
	if config.Language == "" {
		return errors.New("language no puede estar vacío")
	}

	if config.ActiveTicketService != "" {
		if config.TicketProviders != nil {
			if ticketCfg, exists := config.TicketProviders[config.ActiveTicketService]; exists {
				if ticketCfg.BaseURL == "" {
					return fmt.Errorf("%s base URL no está configurada", config.ActiveTicketService)
				}
				if ticketCfg.APIKey == "" {
					return fmt.Errorf("%s API key no está configurada", config.ActiveTicketService)
				}
			}
		}
	}

	return nil
}
