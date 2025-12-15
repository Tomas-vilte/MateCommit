package config

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai/gemini"
	"github.com/Tomas-vilte/MateCommit/internal/ui"
	"github.com/urfave/cli/v3"
)

func (c *ConfigCommandFactory) newInitCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:   "init",
		Usage:  t.GetMessage("config_init_usage", 0, nil),
		Action: initConfigAction(cfg, t),
	}
}

func initConfigAction(cfg *config.Config, t *i18n.Translations) cli.ActionFunc {
	return func(ctx context.Context, command *cli.Command) error {
		reader := bufio.NewReader(os.Stdin)
		return runInitProcess(ctx, command, reader, cfg, t)
	}
}

func runInitProcess(ctx context.Context, command *cli.Command, reader *bufio.Reader, cfg *config.Config, t *i18n.Translations) error {
	if err := configureWelcome(ctx, reader, cfg, t); err != nil {
		return err
	}
	if err := configureLanguage(reader, cfg, t); err != nil {
		return err
	}
	if err := configureVCS(reader, cfg, t); err != nil {
		return err
	}
	if err := configureTickets(reader, cfg, t); err != nil {
		return err
	}
	if err := config.SaveConfig(cfg); err != nil {
		fmt.Println(t.GetMessage("config_save.error_saving_config", 0, map[string]interface{}{"Error": err.Error()}))
		return fmt.Errorf("error saving configuration: %w", err)
	}

	printConfigSummary(cfg, t)

	fmt.Println()
	fmt.Print(t.GetMessage("init.prompt_run_again", 0, nil))
	runAgain, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	if isYes(runAgain) {
		return runInitProcess(ctx, command, reader, cfg, t)
	}

	return nil
}

func configureWelcome(ctx context.Context, reader *bufio.Reader, cfg *config.Config, t *i18n.Translations) error {
	aiProviders := config.SupportedAIs()
	aiProvidersStr := strings.Join(toStrings(aiProviders), ", ")

	geminiModels := config.ModelsForAI(config.AIGemini)
	geminiModelsStr := strings.Join(toStrings(geminiModels), ", ")
	geminiDefault := string(config.DefaultModelForAI(config.AIGemini))

	printSection(t.GetMessage("init.section_welcome", 0, nil))
	fmt.Println(t.GetMessage("init.welcome", 0, nil))
	fmt.Println(t.GetMessage("init.ai_intro", 0, map[string]interface{}{"Providers": aiProvidersStr}))

	ui.PrintInfo(t.GetMessage("config.api_key_instructions", 0, nil))
	ui.PrintInfo(t.GetMessage("config.get_key_at", 0, nil) + " https://makersuite.google.com/app/apikey")
	fmt.Println()

	fmt.Print(t.GetMessage("init.prompt_gemini_api_key", 0, nil))
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading API key: %w", err)
	}
	apiKey = strings.TrimSpace(apiKey)

	if apiKey != "" {
		if !validateGeminiAPIKey(ctx, apiKey, t) {
			ui.PrintWarning(t.GetMessage("config.api_key_saved_unverified", 0, nil))
			fmt.Print(t.GetMessage("config.retry_api_key", 0, nil) + " (y/n): ")
			retry, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(retry)) == "y" {
				return configureWelcome(ctx, reader, cfg, t)
			}
		}
	}
	cfg.GeminiAPIKey = apiKey

	fmt.Println(t.GetMessage("init.model_hint_supported", 0, map[string]interface{}{"Models": geminiModelsStr}))
	fmt.Print(t.GetMessage("init.prompt_model_with_default", 0, map[string]interface{}{"Default": geminiDefault}))
	modelInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading model: %w", err)
	}
	modelInput = strings.TrimSpace(modelInput)

	cfg.AIConfig.ActiveAI = config.AIGemini
	if cfg.AIConfig.Models == nil {
		cfg.AIConfig.Models = make(map[config.AI]config.Model)
	}
	if modelInput == "" {
		cfg.AIConfig.Models[config.AIGemini] = config.Model(geminiDefault)
	} else {
		cfg.AIConfig.Models[config.AIGemini] = config.Model(modelInput)
	}

	return nil
}

func configureLanguage(reader *bufio.Reader, cfg *config.Config, t *i18n.Translations) error {
	printSection(t.GetMessage("init.section_language", 0, nil))
	fmt.Println(t.GetMessage("init.language_supported_with_current", 0, map[string]interface{}{"Current": cfg.Language}))
	fmt.Print(t.GetMessage("init.prompt_language_blank_keeps", 0, nil))

	lang, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading language: %w", err)
	}
	lang = strings.TrimSpace(strings.ToLower(lang))

	if lang != "" {
		if isValidLanguage(lang) {
			cfg.Language = lang
		} else {
			fmt.Println(t.GetMessage("init.error_invalid_language", 0, nil))
		}
	}

	return nil
}

func configureVCS(reader *bufio.Reader, cfg *config.Config, t *i18n.Translations) error {
	vcsProviders := config.SupportedVCSProviders()
	vcsProvidersStr := strings.Join(vcsProviders, ", ")

	printSection(t.GetMessage("init.section_vcs", 0, nil))
	fmt.Print(t.GetMessage("init.prompt_vcs_enable_blank_no", 0, map[string]interface{}{"Providers": vcsProvidersStr}))

	ansVCS, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading VCS answer: %w", err)
	}
	ansVCS = strings.TrimSpace(strings.ToLower(ansVCS))

	if isYes(ansVCS) {
		provider := "github"
		fmt.Print(t.GetMessage("init.prompt_github_token_blank_skip", 0, nil))
		token, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading GitHub token: %w", err)
		}
		token = strings.TrimSpace(token)

		if token != "" {
			if cfg.VCSConfigs == nil {
				cfg.VCSConfigs = make(map[string]config.VCSConfig)
			}
			cfg.VCSConfigs[provider] = config.VCSConfig{
				Provider: provider,
				Token:    token,
			}
			cfg.ActiveVCSProvider = provider
		} else {
			fmt.Println(t.GetMessage("init.info_vcs_skipped", 0, nil))
		}
	}

	return nil
}

func configureTickets(reader *bufio.Reader, cfg *config.Config, t *i18n.Translations) error {
	ticketProviders := config.SupportedTicketServices()
	ticketProvidersStr := strings.Join(ticketProviders, ", ")
	printSection(t.GetMessage("init.section_tickets", 0, nil))
	fmt.Print(t.GetMessage("init.prompt_ticket_enable_blank_no", 0, map[string]interface{}{"Providers": ticketProvidersStr}))

	ansJira, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading Jira answer: %w", err)
	}
	ansJira = strings.TrimSpace(strings.ToLower(ansJira))

	if !isYes(ansJira) {
		disableTickets(cfg)
		return nil
	}

	cfg.UseTicket = true
	cfg.ActiveTicketService = "jira"

	fmt.Print(t.GetMessage("init.prompt_jira_base_url_blank_cancel", 0, nil))
	jiraURL, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading Jira URL: %w", err)
	}
	jiraURL = strings.TrimSpace(jiraURL)

	if jiraURL == "" {
		fmt.Println(t.GetMessage("init.info_jira_canceled", 0, nil))
		disableTickets(cfg)
		return nil
	}

	if !isValidURL(jiraURL) {
		fmt.Println(t.GetMessage("init.warning_invalid_url", 0, nil))
	}
	cfg.JiraConfig.BaseURL = jiraURL

	fmt.Print(t.GetMessage("init.prompt_jira_email_blank_cancel", 0, nil))
	jiraEmail, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading Jira email: %w", err)
	}
	jiraEmail = strings.TrimSpace(jiraEmail)

	if jiraEmail == "" {
		fmt.Println(t.GetMessage("init.info_jira_canceled", 0, nil))
		disableTickets(cfg)
		return nil
	}

	if !isValidEmail(jiraEmail) {
		fmt.Println(t.GetMessage("init.warning_invalid_email", 0, nil))
	}
	cfg.JiraConfig.Email = jiraEmail

	fmt.Print(t.GetMessage("init.prompt_jira_api_token_blank_cancel", 0, nil))
	jiraToken, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading Jira token: %w", err)
	}
	jiraToken = strings.TrimSpace(jiraToken)

	if jiraToken == "" {
		fmt.Println(t.GetMessage("init.info_jira_canceled", 0, nil))
		disableTickets(cfg)
		return nil
	}
	cfg.JiraConfig.APIKey = jiraToken

	return nil
}

func printConfigSummary(cfg *config.Config, t *i18n.Translations) {
	printSection(t.GetMessage("init.section_finish", 0, nil))
	fmt.Println(t.GetMessage("init.saved_ok", 0, nil))
	fmt.Println()
	fmt.Println(t.GetMessage("init.summary_header", 0, nil))

	langLabel := t.GetMessage("language_label", 0, map[string]interface{}{"Lang": cfg.Language})
	activeAI := string(cfg.AIConfig.ActiveAI)
	fmt.Println(t.GetMessage("config_models.active_ai_label", 0, map[string]interface{}{"IA": activeAI}))

	if m, ok := cfg.AIConfig.Models[config.AIGemini]; ok && m != "" {
		fmt.Println(t.GetMessage("init.summary_model", 0, map[string]interface{}{"AI": "gemini", "Model": string(m)}))
	} else {
		fmt.Println(t.GetMessage("init.summary_model_none", 0, map[string]interface{}{"AI": "gemini"}))
	}

	apiMask := "❌"
	if cfg.GeminiAPIKey != "" {
		apiMask = "✅"
	}
	fmt.Println(t.GetMessage("init.summary_api", 0, map[string]interface{}{"AI": "gemini", "Configured": apiMask}))

	if cfg.ActiveVCSProvider != "" {
		fmt.Println(t.GetMessage("vcs_summary.config_active_vcs_updated", 0, map[string]interface{}{"Provider": cfg.ActiveVCSProvider}))
	} else {
		fmt.Println(t.GetMessage("init.summary_vcs_none", 0, nil))
	}

	if cfg.UseTicket && cfg.ActiveTicketService == "jira" {
		fmt.Println(t.GetMessage("config_models.ticket_service_enabled", 0, map[string]interface{}{"Service": "jira"}))
		fmt.Println(t.GetMessage("config_models.jira_config_label", 0, map[string]interface{}{"BaseURL": cfg.JiraConfig.BaseURL, "Email": cfg.JiraConfig.Email}))
	} else {
		fmt.Println(t.GetMessage("config_models.ticket_service_disabled", 0, nil))
	}

	fmt.Println(langLabel)
}

func validateGeminiAPIKey(ctx context.Context, apiKey string, t *i18n.Translations) bool {
	if apiKey == "" {
		return false
	}

	ui.PrintInfo(t.GetMessage("config.validating_api_key", 0, nil))
	spinner := ui.NewSmartSpinner(t.GetMessage("config.testing_connection", 0, nil))
	spinner.Start()

	testCfg := &config.Config{
		GeminiAPIKey: apiKey,
		Language:     "en",
		AIConfig: config.AIConfig{
			ActiveAI: config.AIGemini,
			Models: map[config.AI]config.Model{
				config.AIGemini: config.ModelGeminiV25Flash,
			},
		},
	}

	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := gemini.NewGeminiService(testCtx, testCfg, t)
	if err != nil {
		spinner.Error(t.GetMessage("config.api_key_invalid", 0, nil))
		ui.PrintError(t.GetMessage("config.check_api_key_error", 0, map[string]interface{}{
			"Error": err.Error(),
		}))
		return false
	}

	spinner.Success(t.GetMessage("config.api_key_valid", 0, nil))
	return true
}

func disableTickets(cfg *config.Config) {
	cfg.UseTicket = false
	cfg.ActiveTicketService = ""
}

func isYes(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "y", "yes", "s", "si", "sí":
		return true
	default:
		return false
	}
}

func isValidLanguage(lang string) bool {
	validLangs := map[string]bool{
		"en": true,
		"es": true,
	}
	return validLangs[strings.ToLower(lang)]
}

func isValidURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func isValidEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

func printSection(title string) {
	fmt.Println()
	fmt.Println(title)
}

func toStrings[T ~string](vals []T) []string {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		out = append(out, string(v))
	}
	return out
}
