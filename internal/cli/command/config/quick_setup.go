package config

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/ui"
)

func runQuickSetup(ctx context.Context, reader *bufio.Reader, cfg *config.Config, t *i18n.Translations) error {
	ui.PrintSectionBanner(t.GetMessage("quick_setup.header", 0, nil))
	fmt.Println(t.GetMessage("quick_setup.intro_message", 0, nil))
	fmt.Println()

	provider := "Gemini"
	providerKey := config.AIGemini
	apiURL := "https://makersuite.google.com/app/apikey"

	fmt.Println(t.GetMessage("quick_setup.ai_provider_intro", 0, map[string]interface{}{
		"Provider": provider,
	}))
	fmt.Println(t.GetMessage("quick_setup.get_api_key_at", 0, map[string]interface{}{
		"URL": apiURL,
	}))
	fmt.Println()

	fmt.Print(t.GetMessage("quick_setup.prompt_api_key", 0, map[string]interface{}{
		"Provider": provider,
	}))
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading API_KEY: %w", err)
	}
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		ui.PrintWarning(t.GetMessage("quick_setup.no_api_key_provided", 0, nil))
		return nil
	}

	fmt.Println(t.GetMessage("quick_setup.validating_key", 0, nil))
	if validateGeminiAPIKey(ctx, apiKey, t) {
		ui.PrintSuccess(os.Stdout, t.GetMessage("quick_setup.key_validated", 0, nil))
	} else {
		ui.PrintWarning(t.GetMessage("quick_setup.key_validation_failed", 0, nil))
	}

	cfg.Language = "en"
	cfg.UseEmoji = true
	cfg.SuggestionsCount = 3

	if cfg.AIProviders == nil {
		cfg.AIProviders = make(map[string]config.AIProviderConfig)
	}

	cfg.AIProviders["gemini"] = config.AIProviderConfig{
		APIKey:      apiKey,
		Model:       string(config.ModelGeminiV25Flash),
		Temperature: 0.3,
		MaxTokens:   10000,
	}

	cfg.AIConfig.ActiveAI = providerKey
	if cfg.AIConfig.Models == nil {
		cfg.AIConfig.Models = make(map[config.AI]config.Model)
	}
	cfg.AIConfig.Models[providerKey] = config.ModelGeminiV25Flash

	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("error saving configuration: %w", err)
	}

	fmt.Println()
	ui.PrintSuccess(os.Stdout, t.GetMessage("quick_setup.config_saved", 0, nil))
	fmt.Println()
	ui.PrintInfo(t.GetMessage("quick_setup.try_now", 0, nil))
	ui.PrintInfo(t.GetMessage("quick_setup.full_setup_hint", 0, nil))

	return nil
}
