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

func (r *ReleaseCommandFactory) newPreviewCommand(trans *i18n.Translations) *cli.Command {
	return &cli.Command{
		Name:          "preview",
		Aliases:       []string{"p"},
		Usage:         trans.GetMessage("release.preview_usage", 0, nil),
		ShellComplete: completion_helper.DefaultFlagComplete,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			service, err := r.createReleaseService(ctx, trans)
			if err != nil {
				return err
			}
			return previewReleaseAction(service, trans)(ctx, cmd)
		},
	}
}

func previewReleaseAction(releaseSvc releaseService, trans *i18n.Translations) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		log := logger.FromContext(ctx)
		start := time.Now()

		log.Info("executing release preview command")

		fmt.Println(trans.GetMessage("release.analyzing", 0, nil))
		fmt.Println()

		release, err := releaseSvc.AnalyzeNextRelease(ctx)
		if err != nil {
			log.Error("failed to analyze next release",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds())
			return fmt.Errorf("%s", trans.GetMessage("release.error_analyzing", 0, struct{ Error string }{err.Error()}))
		}

		log.Debug("release analyzed",
			"version", release.Version,
			"previous_version", release.PreviousVersion,
			"features_count", len(release.Features),
			"bugfixes_count", len(release.BugFixes),
			"breaking_count", len(release.Breaking))

		fmt.Println(trans.GetMessage("release.previous_version", 0, struct{ Version string }{release.PreviousVersion}))
		fmt.Println(trans.GetMessage("release.next_version", 0, struct {
			Version string
			Bump    string
		}{release.Version, string(release.VersionBump)}))
		fmt.Println()

		fmt.Println(trans.GetMessage("release.changes_summary", 0, nil))
		if len(release.Breaking) > 0 {
			fmt.Println(trans.GetMessage("release.breaking_changes", 0, struct{ Count int }{len(release.Breaking)}))
		}
		fmt.Println(trans.GetMessage("release.new_features", 0, struct{ Count int }{len(release.Features)}))
		fmt.Println(trans.GetMessage("release.bug_fixes", 0, struct{ Count int }{len(release.BugFixes)}))
		fmt.Println(trans.GetMessage("release.improvements", 0, struct{ Count int }{len(release.Improvements)}))
		fmt.Println(trans.GetMessage("release.documentation", 0, struct{ Count int }{len(release.Documentation)}))
		fmt.Println(trans.GetMessage("release.total_commits", 0, struct{ Count int }{len(release.AllCommits)}))
		fmt.Println()

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
			"title", notes.Title,
			"highlights_count", len(notes.Highlights))

		fmt.Println(trans.GetMessage("release.separator", 0, nil))
		fmt.Printf("## %s\n\n", notes.Title)
		fmt.Printf("%s\n\n", notes.Summary)

		if len(notes.Highlights) > 0 {
			fmt.Println(trans.GetMessage("release.highlights_section", 0, nil))
			for _, h := range notes.Highlights {
				fmt.Printf("- %s\n", h)
			}
			fmt.Println()
		}

		fmt.Println(notes.Changelog)
		fmt.Println(trans.GetMessage("release.separator", 0, nil))
		fmt.Println()

		fmt.Println(trans.GetMessage("release.next_steps", 0, nil))
		fmt.Println(trans.GetMessage("release.next_steps_cmd", 0, nil))
		fmt.Println()

		if notes.Usage != nil {
			ui.PrintTokenUsage(notes.Usage, trans)
			fmt.Println()
		}

		log.Info("release preview command completed successfully",
			"version", release.Version,
			"duration_ms", time.Since(start).Milliseconds())

		return nil
	}
}
