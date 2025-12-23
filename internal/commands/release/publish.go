package release

import (
	"context"
	"fmt"
	"time"

	"github.com/thomas-vilte/matecommit/internal/commands/completion_helper"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/ui"
	"github.com/urfave/cli/v3"
)

func (r *ReleaseCommandFactory) newPublishCommand(trans *i18n.Translations) *cli.Command {
	return &cli.Command{
		Name:    "publish",
		Aliases: []string{"pub"},
		Usage:   trans.GetMessage("release.publish_usage", 0, nil),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   trans.GetMessage("release.version_flag", 0, nil),
			},
			&cli.BoolFlag{
				Name:    "draft",
				Aliases: []string{"d"},
				Usage:   trans.GetMessage("release.draft_flag", 0, nil),
			},
			&cli.BoolFlag{
				Name:    "build-binaries",
				Aliases: []string{"b"},
				Usage:   trans.GetMessage("release.build_binaries_flag", 0, nil),
				Value:   true,
			},
		},
		ShellComplete: completion_helper.DefaultFlagComplete,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			service, err := r.createReleaseService(ctx, trans)
			if err != nil {
				return err
			}
			return publishReleaseAction(service, trans)(ctx, cmd)
		},
	}
}

func publishReleaseAction(releaseSvc releaseService,
	trans *i18n.Translations) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		log := logger.FromContext(ctx)
		start := time.Now()

		version := cmd.String("version")
		draft := cmd.Bool("draft")
		buildBinaries := cmd.Bool("build-binaries")

		log.Info("executing release publish command",
			"version", version,
			"draft", draft,
			"build_binaries", buildBinaries,
		)

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
			return fmt.Errorf("%s", trans.GetMessage("release.error_analyzing", 0, struct{ Error string }{err.Error()}))
		}

		log.Debug("release analyzed",
			"version", release.Version)

		if version := cmd.String("version"); version != "" {
			release.Version = version
		}

		if err := releaseSvc.EnrichReleaseContext(ctx, release); err != nil {
			log.Warn("failed to enrich release context", "error", err)
			fmt.Printf("⚠️  %s\n", trans.GetMessage("release.warning_enrich_context", 0, struct{ Error string }{err.Error()}))
		}

		notes, err := releaseSvc.GenerateReleaseNotes(ctx, release)
		if err != nil {
			log.Error("failed to generate release notes",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds())
			ui.HandleAppError(err, trans)
			return fmt.Errorf("%s", trans.GetMessage("release.error_generating_notes", 0, struct{ Error string }{err.Error()}))
		}

		log.Debug("release notes generated",
			"title", notes.Title)

		draftText := ""
		if draft {
			draftText = " " + trans.GetMessage("release.as_draft", 0, nil)
		}

		fmt.Println(trans.GetMessage("release.publishing", 0, struct {
			Version string
			Draft   string
		}{release.Version, draftText}))

		err = releaseSvc.PublishRelease(ctx, release, notes, draft, buildBinaries)
		if err != nil {
			log.Error("failed to publish release",
				"error", err,
				"version", release.Version,
				"duration_ms", time.Since(start).Milliseconds())
			return fmt.Errorf("%s", trans.GetMessage("release.error_publishing", 0, struct{ Error string }{err.Error()}))
		}

		log.Info("release published successfully",
			"version", release.Version,
			"draft", draft)

		fmt.Println(trans.GetMessage("release.publish_success", 0, struct{ Version string }{release.Version}))

		if notes.Usage != nil {
			fmt.Println()
			ui.PrintTokenUsage(notes.Usage, trans)
		}

		log.Info("release publish command completed successfully",
			"duration_ms", time.Since(start).Milliseconds())

		return nil
	}
}
