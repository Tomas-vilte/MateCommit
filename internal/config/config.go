package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
		AutoFetchTags     bool                 `json:"auto_fetch_tags"`
		MainPath          string               `json:"main_path,omitempty"`
		GitFallback       GitConfig            `json:"git_fallback,omitempty"`
	}

	GitConfig struct {
		UserName  string `json:"user_name,omitempty"`
		UserEmail string `json:"user_email,omitempty"`
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
		Provider string `json:"provider"` // GitHub or gitlab or whatever you want
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

// atomicWriteFile writes data to a file atomically by writing to a temp file
// and then renaming it. This prevents corruption if the process is interrupted.
func atomicWriteFile(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	tmpFile, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()

	defer func() {
		if tmpFile != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	tmpFile = nil

	if err := os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := os.Rename(tmpName, filename); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

func LoadConfig(path string) (*Config, error) {
	var configPath string

	if filepath.Ext(path) == ".json" {
		configPath = path
	} else {
		configDir := filepath.Join(path, ".config", "matecommit")
		configPath = filepath.Join(configDir, "config.json")

		if _, err := os.Stat(configDir); os.IsNotExist(err) {
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return nil, fmt.Errorf("error creating configuration directory: %w", err)
			}
		}
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return CreateDefaultConfig(configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error decoding JSON file: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("the loaded configuration is invalid: %w", err)
	}

	config.PathFile = configPath
	return &config, nil
}

// GetRepoConfigPath returns the path to the repository-local config file
// Returns empty string if not in a git repository
func GetRepoConfigPath() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	repoRoot := strings.TrimSpace(string(output))
	return filepath.Join(repoRoot, ".matecommit", "config.json")
}

// LoadConfigWithHierarchy loads config with repository-local priority
// If in a git repo, uses ONLY local config (.matecommit/config.json)
// If not in a repo, falls back to global config (~/.config/matecommit/config.json)
func LoadConfigWithHierarchy(globalPath string) (*Config, error) {
	// If in a git repo, use local config ONLY
	localPath := GetRepoConfigPath()
	if localPath != "" {
		cfg, err := LoadConfig(localPath)
		if err != nil {
			// If local config doesn't exist, create it
			if os.IsNotExist(err) {
				return CreateDefaultConfig(localPath)
			}
			return nil, err
		}
		return cfg, nil
	}

	// Fallback to global only if not in a repo
	return LoadConfig(globalPath)
}

func CreateDefaultConfig(path string) (*Config, error) {
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
		return nil, fmt.Errorf("error creating configuration directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error encoding default configuration: %w", err)
	}

	if err := atomicWriteFile(path, data, 0644); err != nil {
		return nil, fmt.Errorf("error saving default configuration: %w", err)
	}

	return config, nil
}

// SaveLocalConfig saves config to repository-local location
func SaveLocalConfig(config *Config) error {
	localPath := GetRepoConfigPath()
	if localPath == "" {
		return errors.New("not in a git repository")
	}

	if err := validateConfig(config); err != nil {
		return fmt.Errorf("the configuration to save is invalid: %w", err)
	}

	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating .matecommit directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding configuration: %w", err)
	}

	if err := atomicWriteFile(localPath, data, 0644); err != nil {
		return fmt.Errorf("error saving configuration: %w", err)
	}

	return nil
}

func SaveConfig(config *Config) error {
	if err := validateConfig(config); err != nil {
		return fmt.Errorf("the configuration to save is invalid: %w", err)
	}

	if config.PathFile == "" {
		return errors.New("the configuration file path is not defined")
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding configuration: %w", err)
	}

	if err := atomicWriteFile(config.PathFile, data, 0644); err != nil {
		return fmt.Errorf("error saving configuration: %w", err)
	}

	return nil
}

func validateConfig(config *Config) error {
	if config.Language == "" {
		return errors.New("language cannot be empty")
	}

	if config.ActiveTicketService != "" {
		if config.TicketProviders != nil {
			if ticketCfg, exists := config.TicketProviders[config.ActiveTicketService]; exists {
				if ticketCfg.BaseURL == "" {
					return fmt.Errorf("%s base URL is not configured", config.ActiveTicketService)
				}
				if ticketCfg.APIKey == "" {
					return fmt.Errorf("%s API key is not configured", config.ActiveTicketService)
				}
			}
		}
	}

	return nil
}
