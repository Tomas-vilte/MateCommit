package release

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/thomas-vilte/matecommit/internal/commands/completion_helper"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/ui"
	"github.com/urfave/cli/v3"
)

func (r *ReleaseCommandFactory) newGenerateCommand(trans *i18n.Translations) *cli.Command {
	return &cli.Command{
		Name:    "generate",
		Aliases: []string{"g"},
		Usage:   trans.GetMessage("release.generate_usage", 0, nil),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   trans.GetMessage("release.output_flag", 0, nil),
				Value:   "RELEASE_NOTES.md",
			},
		},
		ShellComplete: completion_helper.DefaultFlagComplete,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			service, err := r.createReleaseService(ctx, trans)
			if err != nil {
				return err
			}
			return generateReleaseAction(service, trans)(ctx, cmd)
		},
	}
}

func generateReleaseAction(releaseSvc releaseService, trans *i18n.Translations) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		log := logger.FromContext(ctx)
		start := time.Now()

		outputFile := cmd.String("output")

		log.Info("executing release generate command",
			"output_file", outputFile)

		fmt.Println(trans.GetMessage("release.generating", 0, nil))
		fmt.Println()

		if err := releaseSvc.ValidateMainBranch(ctx); err != nil {
			log.Error("branch validation failed",
				"error", err)
			return fmt.Errorf("%s", trans.GetMessage("release.error_invalid_branch", 0, struct{ Error string }{err.Error()}))
		}

		release, err := releaseSvc.AnalyzeNextRelease(ctx)
		if err != nil {
			log.Error("failed to analyze next release",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds())
			ui.HandleAppError(err)
			return fmt.Errorf("%s", trans.GetMessage("release.error_analyzing", 0, struct{ Error string }{err.Error()}))
		}

		log.Debug("release analyzed",
			"version", release.Version)

		if err := releaseSvc.EnrichReleaseContext(ctx, release); err != nil {
			log.Warn("failed to enrich release context", "error", err)
			fmt.Printf("⚠️  %s\n", trans.GetMessage("release.warning_enrich_context", 0, struct{ Error string }{err.Error()}))
		}

		notes, err := releaseSvc.GenerateReleaseNotes(ctx, release)
		if err != nil {
			log.Error("failed to generate release notes",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds())
			ui.HandleAppError(err)
			return fmt.Errorf("%s", trans.GetMessage("release.error_generating_notes", 0, struct{ Error string }{err.Error()}))
		}

		log.Debug("release notes generated",
			"title", notes.Title)

		content := FormatReleaseMarkdown(release, notes, trans)

		err = os.WriteFile(outputFile, []byte(content), 0644)
		if err != nil {
			log.Error("failed to write release notes file",
				"error", err,
				"output_file", outputFile,
				"duration_ms", time.Since(start).Milliseconds())
			return fmt.Errorf("%s", trans.GetMessage("release.error_writing_file", 0, struct{ Error string }{err.Error()}))
		}

		log.Info("release notes file written successfully",
			"output_file", outputFile,
			"version", release.Version,
			"content_size", len(content))

		fmt.Println(trans.GetMessage("release.notes_saved", 0, struct{ File string }{outputFile}))
		fmt.Println(trans.GetMessage("release.version_label", 0, struct{ Version string }{release.Version}))

		if notes.Usage != nil {
			fmt.Println()
			ui.PrintTokenUsage(notes.Usage, trans)
		}

		log.Info("release generate command completed successfully",
			"duration_ms", time.Since(start).Milliseconds())

		fmt.Println()

		return nil
	}
}
