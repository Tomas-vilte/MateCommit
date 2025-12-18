package config

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai/gemini"
	"github.com/Tomas-vilte/MateCommit/internal/ui"
	"github.com/urfave/cli/v3"
)

type DoctorCommand struct{}

func NewDoctorCommand() *DoctorCommand {
	return &DoctorCommand{}
}

func (d *DoctorCommand) CreateCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:    "doctor",
		Aliases: []string{"dr"},
		Usage:   t.GetMessage("doctor.command_usage", 0, nil),
		Action: func(ctx context.Context, command *cli.Command) error {
			return d.runHealthCheck(ctx, t, cfg)
		},
	}
}

func (d *DoctorCommand) runHealthCheck(ctx context.Context, t *i18n.Translations, cfg *config.Config) error {
	ui.PrintSectionBanner(t.GetMessage("doctor.running_checks", 0, nil))

	checks := []healthCheck{
		{name: "doctor.check_config_file", fn: d.checkConfigFile},
		{name: "doctor.check_git_repo", fn: d.checkGitRepo},
		{name: "doctor.check_git_installed", fn: d.checkGitInstalled},
		{name: "doctor.check_gemini_key", fn: func(ctx context.Context, t *i18n.Translations, cfg *config.Config) checkResult {
			return d.checkGeminiAPIKey(ctx, t, cfg)
		}},
		{name: "doctor.check_github_token", fn: d.checkGitHubToken},
		{name: "doctor.check_editor", fn: d.checkEditor},
	}

	var warnings []string
	var errors []string
	allPassed := true

	for _, check := range checks {
		checkName := t.GetMessage(check.name, 0, nil)
		spinner := ui.NewSmartSpinner(checkName)
		spinner.Start()

		time.Sleep(100 * time.Millisecond)

		result := check.fn(ctx, t, cfg)

		switch result.status {
		case checkStatusOK:
			spinner.Success(checkName)
			if result.message != "" {
				ui.PrintInfo("  " + result.message)
			}
		case checkStatusWarning:
			spinner.Warning(checkName)
			warnings = append(warnings, result.message)
			if result.suggestion != "" {
				ui.PrintInfo("  → " + result.suggestion)
			}
		case checkStatusError:
			spinner.Error(checkName)
			errors = append(errors, result.message)
			allPassed = false
			if result.suggestion != "" {
				ui.PrintInfo("  → " + result.suggestion)
			}
		}
	}

	fmt.Println()
	ui.PrintSectionBanner(t.GetMessage("doctor.summary", 0, nil))

	if allPassed && len(warnings) == 0 {
		ui.PrintSuccess(t.GetMessage("doctor.all_good", 0, nil))
	} else if len(errors) == 0 {
		ui.PrintWarning(t.GetMessage("doctor.has_warnings", 0, nil))
	} else {
		ui.PrintError(t.GetMessage("doctor.has_errors", 0, nil))
	}

	fmt.Println()
	ui.PrintInfo(t.GetMessage("doctor.available_commands", 0, nil))

	hasGemini := false
	if providerCfg, exists := cfg.AIProviders["gemini"]; exists && providerCfg.APIKey != "" {
		hasGemini = true
	}
	hasGitHub := false
	if cfg.VCSConfigs != nil {
		if githubConfig, ok := cfg.VCSConfigs["github"]; ok && githubConfig.Token != "" {
			hasGitHub = true
		}
	}

	d.printCommandStatus("suggest", hasGemini, t)
	d.printCommandStatus("summarize-pr", hasGitHub, t)
	d.printCommandStatus("config", true, t)

	return nil
}

type healthCheck struct {
	name string
	fn   func(context.Context, *i18n.Translations, *config.Config) checkResult
}

type checkStatus int

const (
	checkStatusOK checkStatus = iota
	checkStatusWarning
	checkStatusError
)

type checkResult struct {
	status     checkStatus
	message    string
	suggestion string
}

func (d *DoctorCommand) checkConfigFile(_ context.Context, t *i18n.Translations, cfg *config.Config) checkResult {
	configPath := cfg.PathFile
	if configPath == "" || !fileExists(configPath) {
		return checkResult{
			status:     checkStatusError,
			message:    t.GetMessage("doctor.config_not_found", 0, nil),
			suggestion: t.GetMessage("doctor.run_config_init", 0, nil),
		}
	}
	return checkResult{
		status:  checkStatusOK,
		message: fmt.Sprintf("(%s)", configPath),
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (d *DoctorCommand) checkGitRepo(ctx context.Context, t *i18n.Translations, _ *config.Config) checkResult {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return checkResult{
			status:     checkStatusWarning,
			message:    t.GetMessage("doctor.not_in_git_repo", 0, nil),
			suggestion: t.GetMessage("doctor.git_init_suggestion", 0, nil),
		}
	}

	cmd = exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err == nil {
		repoPath := strings.TrimSpace(string(output))
		return checkResult{
			status:  checkStatusOK,
			message: fmt.Sprintf("(%s)", repoPath),
		}
	}

	return checkResult{status: checkStatusOK}
}

func (d *DoctorCommand) checkGitInstalled(ctx context.Context, t *i18n.Translations, _ *config.Config) checkResult {
	cmd := exec.CommandContext(ctx, "git", "--version")
	output, err := cmd.Output()
	if err != nil {
		return checkResult{
			status:     checkStatusError,
			message:    t.GetMessage("doctor.git_not_installed", 0, nil),
			suggestion: t.GetMessage("doctor.install_git_suggestion", 0, nil),
		}
	}

	version := strings.TrimSpace(string(output))
	return checkResult{
		status:  checkStatusOK,
		message: version,
	}
}

func (d *DoctorCommand) checkGeminiAPIKey(ctx context.Context, t *i18n.Translations, cfg *config.Config) checkResult {
	providerCfg, exists := cfg.AIProviders["gemini"]
	if !exists || providerCfg.APIKey == "" {
		return checkResult{
			status:     checkStatusWarning,
			message:    t.GetMessage("doctor.gemini_not_configured", 0, nil),
			suggestion: t.GetMessage("doctor.run_config_init", 0, nil),
		}
	}

	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	service, err := gemini.NewGeminiCommitSummarizer(testCtx, cfg, t)
	if err != nil {
		return checkResult{
			status:     checkStatusError,
			message:    t.GetMessage("doctor.gemini_key_invalid", 0, nil),
			suggestion: t.GetMessage("doctor.check_api_key", 0, nil),
		}
	}
	_ = service

	return checkResult{
		status:  checkStatusOK,
		message: t.GetMessage("doctor.api_key_valid", 0, nil),
	}
}

func (d *DoctorCommand) checkGitHubToken(_ context.Context, t *i18n.Translations, cfg *config.Config) checkResult {
	hasGitHub := false
	if cfg.VCSConfigs != nil {
		if githubConfig, ok := cfg.VCSConfigs["github"]; ok && githubConfig.Token != "" {
			hasGitHub = true
		}
	}

	if !hasGitHub {
		return checkResult{
			status:     checkStatusWarning,
			message:    t.GetMessage("doctor.github_not_configured", 0, nil),
			suggestion: t.GetMessage("doctor.github_optional", 0, nil),
		}
	}

	return checkResult{
		status:  checkStatusOK,
		message: t.GetMessage("doctor.github_configured", 0, nil),
	}
}

func (d *DoctorCommand) checkEditor(_ context.Context, t *i18n.Translations, _ *config.Config) checkResult {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editors := []string{"nano", "vim", "vi", "code", "emacs"}
		for _, ed := range editors {
			if _, err := exec.LookPath(ed); err == nil {
				return checkResult{
					status:     checkStatusWarning,
					message:    t.GetMessage("doctor.editor_not_set", 0, map[string]interface{}{"Editor": ed}),
					suggestion: fmt.Sprintf("export EDITOR=%s", ed),
				}
			}
		}

		return checkResult{
			status:     checkStatusError,
			message:    t.GetMessage("doctor.no_editor_found", 0, nil),
			suggestion: t.GetMessage("doctor.install_editor", 0, nil),
		}
	}

	if _, err := exec.LookPath(editor); err != nil {
		return checkResult{
			status:     checkStatusError,
			message:    t.GetMessage("doctor.editor_not_found", 0, map[string]interface{}{"Editor": editor}),
			suggestion: t.GetMessage("doctor.set_valid_editor", 0, nil),
		}
	}

	return checkResult{
		status:  checkStatusOK,
		message: fmt.Sprintf("(%s)", editor),
	}
}

func (d *DoctorCommand) printCommandStatus(command string, available bool, t *i18n.Translations) {
	status := "✗"
	statusMsg := t.GetMessage("doctor.command_unavailable", 0, nil)
	if available {
		status = "✓"
		statusMsg = t.GetMessage("doctor.command_ready", 0, nil)
	}

	fmt.Printf("  %s matecommit %-15s %s\n", status, command, statusMsg)
}
