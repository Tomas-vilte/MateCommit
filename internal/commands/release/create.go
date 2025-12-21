package release

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/thomas-vilte/matecommit/internal/commands/completion_helper"
	cfg "github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/ui"
	"github.com/urfave/cli/v3"
)

func (r *ReleaseCommandFactory) newCreateCommand(t *i18n.Translations) *cli.Command {
	return &cli.Command{
		Name:    "create",
		Aliases: []string{"c"},
		Usage:   t.GetMessage("release.create_usage", 0, nil),
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "auto",
				Aliases: []string{"y"},
				Usage:   t.GetMessage("release.auto_flag", 0, nil),
			},
			&cli.StringFlag{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   t.GetMessage("release.version_flag", 0, nil),
			},
			&cli.BoolFlag{
				Name:  "publish",
				Usage: t.GetMessage("release.flag_publish_usage", 0, nil),
			},
			&cli.BoolFlag{
				Name:  "draft",
				Usage: t.GetMessage("release.flag_draft_usage", 0, nil),
			},
			&cli.BoolFlag{
				Name:  "changelog",
				Usage: t.GetMessage("release.flag_changelog_usage", 0, nil),
			},
			&cli.BoolFlag{
				Name:    "build-binaries",
				Aliases: []string{"b"},
				Usage:   t.GetMessage("release.build_binaries_flag", 0, nil),
				Value:   true,
			},
		},
		ShellComplete: completion_helper.DefaultFlagComplete,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			service, err := r.createReleaseService(ctx, t)
			if err != nil {
				return err
			}
			reader := bufio.NewReader(os.Stdin)
			return createReleaseAction(service, t, reader, r.config)(ctx, cmd)
		},
	}
}

func createReleaseAction(releaseSvc releaseService, trans *i18n.Translations, reader *bufio.Reader, config *cfg.Config) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println(trans.GetMessage("release.creating", 0, nil))
		fmt.Println()

		release, err := releaseSvc.AnalyzeNextRelease(ctx)
		if err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_analyzing", 0, struct{ Error string }{err.Error()}))
		}

		if version := cmd.String("version"); version != "" {
			release.Version = version
		}

		if err := releaseSvc.EnrichReleaseContext(ctx, release); err != nil {
			fmt.Printf("⚠️  %s\n", trans.GetMessage("release.warning_enrich_context", 0, struct{ Error string }{err.Error()}))
		}

		notes, err := releaseSvc.GenerateReleaseNotes(ctx, release)
		if err != nil {
			ui.HandleAppError(err, trans)
			return fmt.Errorf("%s", trans.GetMessage("release.error_generating_notes", 0, struct{ Error string }{err.Error()}))
		}

		updateChangelog := cmd.Bool("changelog")
		if config != nil && config.UpdateChangelog {
			updateChangelog = true
		}

		if updateChangelog {
			s := ui.NewSmartSpinner(trans.GetMessage("release.changelog_update_started", 0, nil))
			s.Start()

			if err := releaseSvc.UpdateLocalChangelog(release, notes); err != nil {
				s.Error(trans.GetMessage("release.error_updating_changelog", 0, struct{ Error string }{err.Error()}))
				return fmt.Errorf("%w", err)
			}
			s.Success(trans.GetMessage("release.changelog_updated", 0, nil))
			fmt.Println()

			sVersion := ui.NewSmartSpinner(trans.GetMessage("release.app_version_update_started", 0, struct{ Version string }{release.Version}))
			sVersion.Start()
			if err := releaseSvc.UpdateAppVersion(release.Version); err != nil {
				sVersion.Error(trans.GetMessage("release.error_updating_app_version", 0, struct{ Error string }{err.Error()}))
				return fmt.Errorf("error updating app version: %w", err)
			}
			sVersion.Success(trans.GetMessage("release.app_version_updated", 0, struct{ Version string }{release.Version}))
			fmt.Println()

			sCommit := ui.NewSmartSpinner(trans.GetMessage("release.committing_changelog", 0, nil))
			sCommit.Start()
			if err := releaseSvc.CommitChangelog(ctx, release.Version); err != nil {
				sCommit.Error(trans.GetMessage("release.error_committing_changelog", 0, struct{ Error string }{err.Error()}))
				return fmt.Errorf("error committing changelog: %w", err)
			}
			sCommit.Success(trans.GetMessage("release.changelog_committed", 0, nil))
			fmt.Println()

			sPush := ui.NewSmartSpinner(trans.GetMessage("release.pushing_changes", 0, nil))
			sPush.Start()
			if err := releaseSvc.PushChanges(ctx); err != nil {
				sPush.Error(trans.GetMessage("release.error_pushing_changes", 0, struct{ Error string }{err.Error()}))
				return fmt.Errorf("error pushing changes: %w", err)
			}
			sPush.Success(trans.GetMessage("release.changes_pushed", 0, nil))
			fmt.Println()
		}

		fmt.Println(trans.GetMessage("release.create_preview", 0, struct {
			Version string
			Bump    string
		}{release.Version, string(release.VersionBump)}))
		fmt.Println(trans.GetMessage("release.create_title", 0, struct{ Title string }{notes.Title}))
		fmt.Println(trans.GetMessage("release.create_stats", 0, struct {
			Features int
			Fixes    int
			Breaking int
		}{len(release.Features), len(release.BugFixes), len(release.Breaking)}))
		fmt.Println()

		if !cmd.Bool("auto") {
			fmt.Print(trans.GetMessage("release.create_confirm", 0, nil))
			response, _ := reader.ReadString('\n')
			response = strings.ToLower(strings.TrimSpace(response))

			if response != "y" && response != "yes" && response != "s" && response != "si" {
				fmt.Println(trans.GetMessage("release.create_cancelled", 0, nil))
				return nil
			}
		}

		message := fmt.Sprintf("%s\n\n%s", notes.Title, notes.Summary)
		err = releaseSvc.CreateTag(ctx, release.Version, message)
		if err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_creating_tag", 0, struct{ Error string }{err.Error()}))
		}

		fmt.Println(trans.GetMessage("release.tag_created", 0, struct{ Version string }{release.Version}))

		if cmd.Bool("publish") {
			notes.Changelog = FormatReleaseMarkdown(release, notes, trans)

			fmt.Println(trans.GetMessage("release.publishing_release", 0, nil))
			buildBinaries := cmd.Bool("build-binaries")
			err := releaseSvc.PublishRelease(ctx, release, notes, cmd.Bool("draft"), buildBinaries)
			if err != nil {
				return fmt.Errorf("%s", trans.GetMessage("release.error_publishing_release", 0, struct{ Error string }{err.Error()}))
			}
			fmt.Println(trans.GetMessage("release.release_published", 0, nil))
		} else {
			fmt.Println()
			fmt.Println(trans.GetMessage("release.create_next_steps", 0, nil))
			fmt.Println(trans.GetMessage("release.create_review", 0, struct{ Version string }{release.Version}))
			fmt.Println(trans.GetMessage("release.create_push", 0, struct{ Version string }{release.Version}))
			fmt.Println(trans.GetMessage("release.create_push_help", 0, nil))
		}

		if notes.Usage != nil {
			fmt.Println()
			ui.PrintTokenUsage(notes.Usage, trans)
		}

		fmt.Println()

		return nil
	}
}
