package config

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/google/go-github/v80/github"
	"github.com/thomas-vilte/matecommit/internal/ai/gemini"
	"github.com/thomas-vilte/matecommit/internal/commands/completion_helper"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/ui"
	"github.com/urfave/cli/v3"
)

func (c *ConfigCommandFactory) newInitCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: t.GetMessage("config_init_usage", 0, nil),
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "quick",
				Aliases: []string{"q"},
				Usage:   t.GetMessage("config_init_quick_flag", 0, nil),
			},
			&cli.BoolFlag{
				Name:  "full",
				Usage: t.GetMessage("config_init_full_flag", 0, nil),
			},
			&cli.BoolFlag{
				Name:    "local",
				Aliases: []string{"l"},
				Usage:   t.GetMessage("config_init_local_flag", 0, nil),
			},
			&cli.BoolFlag{
				Name:    "global",
				Aliases: []string{"g"},
				Usage:   t.GetMessage("config_init_global_flag", 0, nil),
			},
		},
		ShellComplete: completion_helper.DefaultFlagComplete,
		Action:        initConfigAction(cfg, t),
	}
}

func initConfigAction(cfg *config.Config, t *i18n.Translations) cli.ActionFunc {
	return func(ctx context.Context, command *cli.Command) error {
		isLocalExplicit := command.Bool("local")
		isGlobalExplicit := command.Bool("global")

		useLocal := isLocalExplicit
		if !isGlobalExplicit && !isLocalExplicit {
			localPath := config.GetRepoConfigPath()
			useLocal = localPath != ""
		}

		if useLocal {
			localPath := config.GetRepoConfigPath()
			if localPath == "" {
				return errors.New(t.GetMessage("config_local.not_in_repo", 0, nil))
			}

			localCfg, err := config.LoadConfig(localPath)
			if err != nil {
				if os.IsNotExist(err) {
					localCfg, err = config.CreateDefaultConfig(localPath)
					if err != nil {
						return fmt.Errorf("error creating local config: %w", err)
					}
				} else {
					return fmt.Errorf("error loading local config: %w", err)
				}
			}

			reader := bufio.NewReader(os.Stdin)

			if command.Bool("quick") {
				return runQuickSetupLocal(reader, localCfg, t)
			}

			if command.Bool("full") {
				return runFullSetupLocal(ctx, command, reader, localCfg, t)
			}

			fmt.Println(t.GetMessage("setup_mode.choose_mode", 0, nil))
			fmt.Println(t.GetMessage("setup_mode.quick_option", 0, nil))
			fmt.Println(t.GetMessage("setup_mode.full_option", 0, nil))
			fmt.Print(t.GetMessage("setup_mode.prompt_selection", 0, nil))

			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(choice)

			if choice == "" || choice == "1" {
				return runQuickSetupLocal(reader, localCfg, t)
			}

			return runFullSetupLocal(ctx, command, reader, localCfg, t)
		}

		reader := bufio.NewReader(os.Stdin)

		if command.Bool("quick") {
			return runQuickSetup(ctx, reader, cfg, t)
		}

		if command.Bool("full") {
			return runFullSetup(ctx, command, reader, cfg, t)
		}

		fmt.Println(t.GetMessage("setup_mode.choose_mode", 0, nil))
		fmt.Println(t.GetMessage("setup_mode.quick_option", 0, nil))
		fmt.Println(t.GetMessage("setup_mode.full_option", 0, nil))
		fmt.Print(t.GetMessage("setup_mode.prompt_selection", 0, nil))

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		if choice == "" || choice == "1" {
			return runQuickSetup(ctx, reader, cfg, t)
		}

		return runFullSetup(ctx, command, reader, cfg, t)
	}
}

func runFullSetup(ctx context.Context, command *cli.Command, reader *bufio.Reader, cfg *config.Config, t *i18n.Translations) error {
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
		fmt.Println(t.GetMessage("config_save.error_saving_config", 0, struct{ Error string }{err.Error()}))
		return fmt.Errorf("error saving configuration: %w", err)
	}

	printConfigSummary(cfg, t)

	if ui.AskConfirmation(t.GetMessage("init.prompt_run_again", 0, nil)) {
		return runFullSetup(ctx, command, reader, cfg, t)
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
	fmt.Println(t.GetMessage("init.ai_intro", 0, struct{ Providers string }{aiProvidersStr}))

	ui.PrintInfo(t.GetMessage("config.api_key_instructions", 0, struct{ Provider string }{"Gemini"}))
	ui.PrintInfo(t.GetMessage("config.get_key_at", 0, struct{ URL string }{"https://makersuite.google.com/app/apikey"}))
	fmt.Println()

	fmt.Print(t.GetMessage("init.prompt_ai_api_key", 0, struct{ Provider string }{"Gemini"}))
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading API_KEY: %w", err)
	}
	apiKey = strings.TrimSpace(apiKey)

	if apiKey != "" {
		if !validateGeminiAPIKey(ctx, apiKey, t) {
			ui.PrintWarning(t.GetMessage("config.api_key_saved_unverified", 0, nil))
			if ui.AskConfirmation(t.GetMessage("config.retry_api_key", 0, nil)) {
				return configureWelcome(ctx, reader, cfg, t)
			}
		}
	}

	fmt.Println(t.GetMessage("init.model_hint_supported", 0, struct{ Models string }{geminiModelsStr}))
	fmt.Print(t.GetMessage("init.prompt_model_with_default", 0, struct{ Default string }{geminiDefault}))
	modelInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading model: %w", err)
	}
	modelInput = strings.TrimSpace(modelInput)

	selectedModel := geminiDefault
	if modelInput != "" {
		selectedModel = modelInput
	}

	if cfg.AIProviders == nil {
		cfg.AIProviders = make(map[string]config.AIProviderConfig)
	}

	cfg.AIProviders["gemini"] = config.AIProviderConfig{
		APIKey:      apiKey,
		Model:       selectedModel,
		Temperature: 0.3,
		MaxTokens:   10000,
	}

	cfg.AIConfig.ActiveAI = config.AIGemini
	if cfg.AIConfig.Models == nil {
		cfg.AIConfig.Models = make(map[config.AI]config.Model)
	}
	cfg.AIConfig.Models[config.AIGemini] = config.Model(selectedModel)

	return nil
}

func configureLanguage(reader *bufio.Reader, cfg *config.Config, t *i18n.Translations) error {
	printSection(t.GetMessage("init.section_language", 0, nil))
	fmt.Println(t.GetMessage("init.language_supported_with_current", 0, struct{ Current string }{cfg.Language}))
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

	if !ui.AskConfirmation(t.GetMessage("init.prompt_vcs_enable_blank_no", 0, struct{ Providers string }{vcsProvidersStr})) {
		fmt.Println(t.GetMessage("init.info_vcs_skipped", 0, nil))
		return nil
	}

	provider := "github"

	for {
		ui.PrintInfo(t.GetMessage("config.github_token_instructions", 0, nil))
		ui.PrintInfo(t.GetMessage("config.get_token_at", 0, struct{ URL string }{"https://github.com/settings/tokens/new"}))
		fmt.Println()

		fmt.Print(t.GetMessage("init.prompt_github_token_blank_skip", 0, nil))
		token, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading GitHub token: %w", err)
		}
		token = strings.TrimSpace(token)

		if token == "" {
			fmt.Println(t.GetMessage("init.info_vcs_skipped", 0, nil))
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		isValid := validateGitHubToken(ctx, token, t)
		cancel()

		if !isValid {
			ui.PrintWarning(t.GetMessage("config.token_saved_unverified", 0, nil))
			if ui.AskConfirmation(t.GetMessage("config.retry_token", 0, nil)) {
				continue
			}
		}

		if cfg.VCSConfigs == nil {
			cfg.VCSConfigs = make(map[string]config.VCSConfig)
		}
		cfg.VCSConfigs[provider] = config.VCSConfig{
			Provider: provider,
			Token:    token,
		}
		cfg.ActiveVCSProvider = provider

		break
	}

	return nil
}

func configureTickets(reader *bufio.Reader, cfg *config.Config, t *i18n.Translations) error {
	ticketProviders := config.SupportedTicketServices()
	ticketProvidersStr := strings.Join(ticketProviders, ", ")

	printSection(t.GetMessage("init.section_tickets", 0, nil))

	if !ui.AskConfirmation(t.GetMessage("init.prompt_ticket_enable_blank_no", 0, struct{ Providers string }{ticketProvidersStr})) {
		disableTickets(cfg)
		return nil
	}

	cfg.UseTicket = true
	cfg.ActiveTicketService = "jira"

	for {
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
			ui.PrintWarning(t.GetMessage("init.warning_invalid_url", 0, nil))
			if !ui.AskConfirmation(t.GetMessage("init.confirm_continue_anyway", 0, nil)) {
				continue
			}
		}

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
			ui.PrintWarning(t.GetMessage("init.warning_invalid_email", 0, nil))
		}

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

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		isValid := validateJiraConnection(ctx, jiraURL, jiraEmail, jiraToken, t)
		cancel()

		if !isValid {
			ui.PrintWarning(t.GetMessage("config.jira_saved_unverified", 0, nil))
			if ui.AskConfirmation(t.GetMessage("config.retry_jira", 0, nil)) {
				continue
			}
		}

		if cfg.TicketProviders == nil {
			cfg.TicketProviders = make(map[string]config.TicketProviderConfig)
		}

		cfg.TicketProviders["jira"] = config.TicketProviderConfig{
			APIKey:  jiraToken,
			BaseURL: jiraURL,
			Email:   jiraEmail,
		}

		break
	}

	return nil
}

func printConfigSummary(cfg *config.Config, t *i18n.Translations) {
	printSection(t.GetMessage("init.section_finish", 0, nil))
	fmt.Println(t.GetMessage("init.saved_ok", 0, nil))
	fmt.Println()
	fmt.Println(t.GetMessage("init.summary_header", 0, nil))

	langLabel := t.GetMessage("language_label", 0, struct{ Lang string }{cfg.Language})
	activeAI := string(cfg.AIConfig.ActiveAI)
	fmt.Println(t.GetMessage("config_models.active_ai_label", 0, struct{ IA string }{activeAI}))

	if m, ok := cfg.AIConfig.Models[config.AIGemini]; ok && m != "" {
		fmt.Println(t.GetMessage("init.summary_model", 0, struct {
			AI    string
			Model string
		}{"gemini", string(m)}))
	} else {
		fmt.Println(t.GetMessage("init.summary_model_none", 0, struct{ AI string }{"gemini"}))
	}

	apiMask := "❌"
	if providerCfg, exists := cfg.AIProviders["gemini"]; exists && providerCfg.APIKey != "" {
		apiMask = "✅"
	}
	fmt.Println(t.GetMessage("init.summary_api", 0, struct {
		AI         string
		Configured string
	}{"gemini", apiMask}))

	if cfg.ActiveVCSProvider != "" {
		fmt.Println(t.GetMessage("vcs_summary.config_active_vcs_updated", 0, struct{ Provider string }{cfg.ActiveVCSProvider}))
	} else {
		fmt.Println(t.GetMessage("init.summary_vcs_none", 0, nil))
	}

	if cfg.UseTicket && cfg.ActiveTicketService == "jira" {
		fmt.Println(t.GetMessage("config_models.ticket_service_enabled", 0, struct{ Service string }{"jira"}))
		jiraCfg := cfg.TicketProviders["jira"]
		fmt.Println(t.GetMessage("config_models.jira_config_label", 0, struct {
			BaseURL string
			Email   string
		}{jiraCfg.BaseURL, jiraCfg.Email}))
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
		Language: "en",
		AIProviders: map[string]config.AIProviderConfig{
			"gemini": {
				APIKey:      apiKey,
				Model:       string(config.ModelGeminiV15Flash),
				Temperature: 0.3,
				MaxTokens:   10000,
			},
		},
		AIConfig: config.AIConfig{
			ActiveAI: config.AIGemini,
			Models: map[config.AI]config.Model{
				config.AIGemini: config.ModelGeminiV25Flash,
			},
		},
	}

	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := gemini.NewGeminiCommitSummarizer(testCtx, testCfg, nil)
	if err != nil {
		spinner.Error(t.GetMessage("config.api_key_invalid", 0, nil))
		ui.PrintError(os.Stdout, t.GetMessage("config.check_api_key_error", 0, struct{ Error string }{err.Error()}))
		return false
	}

	spinner.Success(t.GetMessage("config.api_key_valid", 0, nil))
	return true
}

func validateGitHubToken(ctx context.Context, token string, t *i18n.Translations) bool {
	if token == "" {
		return false
	}

	ui.PrintInfo(t.GetMessage("config.validating_github_token", 0, nil))
	spinner := ui.NewSmartSpinner(t.GetMessage("config.testing_github_connection", 0, nil))
	spinner.Start()

	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client := github.NewClient(nil).WithAuthToken(token)

	user, resp, err := client.Users.Get(testCtx, "")
	if err != nil {
		spinner.Error(t.GetMessage("config.github_token_invalid", 0, nil))
		ui.PrintError(os.Stdout, t.GetMessage("config.check_token_error", 0, struct{ Error string }{err.Error()}))
		return false
	}

	if resp.StatusCode != 200 {
		spinner.Error(t.GetMessage("config.github_token_invalid", 0, nil))
		return false
	}

	spinner.Success(t.GetMessage("config.github_token_valid", 0, nil))

	cyan := color.New(color.FgCyan)
	dim := color.New(color.FgHiBlack)
	green := color.New(color.FgGreen)

	fmt.Println()
	_, _ = cyan.Println(t.GetMessage("config.github_token_info", 0, nil))
	if user.Login != nil {
		_, _ = dim.Printf("   %s\n", t.GetMessage("config.github_authenticated_as", 0, struct{ Login string }{green.Sprint(*user.Login)}))
	}
	if user.Name != nil && *user.Name != "" {
		_, _ = dim.Printf("   %s\n", t.GetMessage("config.github_name_label", 0, struct{ Name string }{*user.Name}))
	}

	scopes := resp.Header.Get("X-OAuth-Scopes")
	if scopes != "" {
		_, _ = dim.Printf("\n   %s\n", t.GetMessage("config.github_detected_permissions", 0, nil))

		scopeList := strings.Split(scopes, ", ")
		hasRepo := false
		hasWorkflow := false

		for _, scope := range scopeList {
			scope = strings.TrimSpace(scope)
			switch scope {
			case "repo":
				hasRepo = true
				_, _ = green.Printf("   %s\n", t.GetMessage("config.github_scope_repo", 0, struct{ Scope string }{scope}))
			case "workflow":
				hasWorkflow = true
				_, _ = green.Printf("   %s\n", t.GetMessage("config.github_scope_workflow", 0, struct{ Scope string }{scope}))
			case "admin:org":
				_, _ = green.Printf("   %s\n", t.GetMessage("config.github_scope_admin_org", 0, struct{ Scope string }{scope}))
			case "user":
				_, _ = green.Printf("   %s\n", t.GetMessage("config.github_scope_user", 0, struct{ Scope string }{scope}))
			default:
				_, _ = dim.Printf("   %s\n", t.GetMessage("config.github_scope_other", 0, struct{ Scope string }{scope}))
			}
		}

		if !hasRepo {
			yellow := color.New(color.FgYellow)
			_, _ = yellow.Printf("\n   %s\n", t.GetMessage("config.github_missing_repo", 0, nil))
		}
		if !hasWorkflow {
			yellow := color.New(color.FgYellow)
			_, _ = yellow.Printf("   %s\n", t.GetMessage("config.github_missing_workflow", 0, nil))
		}
	}

	fmt.Println()
	return true
}

func validateJiraConnection(ctx context.Context, baseURL, email, token string, t *i18n.Translations) bool {
	if baseURL == "" || email == "" || token == "" {
		return false
	}

	ui.PrintInfo(t.GetMessage("config.validating_jira_connection", 0, nil))
	spinner := ui.NewSmartSpinner(t.GetMessage("config.testing_jira_connection", 0, nil))
	spinner.Start()

	testCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	testURL := strings.TrimSuffix(baseURL, "/") + "/rest/api/3/myself"

	req, err := http.NewRequestWithContext(testCtx, "GET", testURL, nil)
	if err != nil {
		spinner.Error(t.GetMessage("config.jira_connection_failed", 0, nil))
		ui.PrintError(os.Stdout, t.GetMessage("config.jira_error_creating_request", 0, struct{ Error error }{err}))
		return false
	}

	req.SetBasicAuth(email, token)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		spinner.Error(t.GetMessage("config.jira_connection_failed", 0, nil))
		ui.PrintError(os.Stdout, t.GetMessage("config.check_jira_error", 0, struct{ Error string }{err.Error()}))
		return false
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			return
		}
	}()

	if resp.StatusCode != 200 {
		spinner.Error(t.GetMessage("config.jira_connection_failed", 0, nil))
		var errorMsg string
		switch resp.StatusCode {
		case 401:
			errorMsg = t.GetMessage("config.jira_invalid_credentials", 0, nil)
		case 403:
			errorMsg = t.GetMessage("config.jira_access_denied", 0, nil)
		default:
			errorMsg = t.GetMessage("config.jira_http_error", 0, struct{ Code int }{resp.StatusCode})
		}
		ui.PrintError(os.Stdout, errorMsg)
		return false
	}

	var userInfo struct {
		AccountID    string `json:"accountId"`
		EmailAddress string `json:"emailAddress"`
		DisplayName  string `json:"displayName"`
		Active       bool   `json:"active"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		spinner.Success(t.GetMessage("config.jira_connection_valid", 0, nil))
		return true
	}

	spinner.Success(t.GetMessage("config.jira_connection_valid", 0, nil))

	cyan := color.New(color.FgCyan)
	dim := color.New(color.FgHiBlack)
	green := color.New(color.FgGreen)

	fmt.Println()
	_, _ = cyan.Println(t.GetMessage("config.jira_connection_info", 0, nil))
	if userInfo.DisplayName != "" {
		_, _ = dim.Printf("   %s\n", t.GetMessage("config.jira_user_label", 0, struct{ User string }{green.Sprint(userInfo.DisplayName)}))
	}
	if userInfo.EmailAddress != "" {
		_, _ = dim.Printf("   %s\n", t.GetMessage("config.jira_email_label", 0, struct{ Email string }{userInfo.EmailAddress}))
	}
	if userInfo.Active {
		_, _ = green.Printf("   %s\n", t.GetMessage("config.jira_status_active", 0, nil))
	}
	fmt.Println()

	return true
}

func disableTickets(cfg *config.Config) {
	cfg.UseTicket = false
	cfg.ActiveTicketService = ""
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
