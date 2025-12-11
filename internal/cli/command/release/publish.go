package release

import (
	"context"
	"fmt"

	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
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
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			service, err := r.createReleaseService(ctx, trans)
			if err != nil {
				return err
			}
			return publishReleaseAction(service, trans)(ctx, cmd)
		},
	}
}

func publishReleaseAction(releaseService ports.ReleaseService, trans *i18n.Translations) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		release, err := releaseService.AnalyzeNextRelease(ctx)
		if err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_analyzing", 0, map[string]interface{}{
				"Error": err.Error(),
			}))
		}

		if version := cmd.String("version"); version != "" {
			release.Version = version
		}

		if err := releaseService.EnrichReleaseContext(ctx, release); err != nil {
			fmt.Printf("⚠️  %s\n", trans.GetMessage("release.warning_enrich_context", 0, map[string]interface{}{
				"Error": err.Error(),
			}))
		}

		notes, err := releaseService.GenerateReleaseNotes(ctx, release)
		if err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_generating_notes", 0, map[string]interface{}{
				"Error": err.Error(),
			}))
		}

		draft := cmd.Bool("draft")
		draftText := ""
		if draft {
			draftText = " " + trans.GetMessage("release.as_draft", 0, nil)
		}

		fmt.Println(trans.GetMessage("release.publishing", 0, map[string]interface{}{
			"Version": release.Version,
			"Draft":   draftText,
		}))

		err = releaseService.PublishRelease(ctx, release, notes, draft)
		if err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_publishing", 0, map[string]interface{}{
				"Error": err.Error(),
			}))
		}

		fmt.Println(trans.GetMessage("release.publish_success", 0, map[string]interface{}{
			"Version": release.Version,
		}))

		return nil
	}
}
