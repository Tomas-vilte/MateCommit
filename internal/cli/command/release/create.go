package release

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
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
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			service, err := r.createReleaseService(ctx, t)
			if err != nil {
				return err
			}
			reader := bufio.NewReader(os.Stdin)
			return createReleaseAction(service, t, reader)(ctx, cmd)
		},
	}
}

func createReleaseAction(releaseService ports.ReleaseService, trans *i18n.Translations, reader *bufio.Reader) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println(trans.GetMessage("release.creating", 0, nil))
		fmt.Println()

		release, err := releaseService.AnalyzeNextRelease(ctx)
		if err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_analyzing", 0, map[string]interface{}{
				"Error": err.Error(),
			}))
		}

		if version := cmd.String("version"); version != "" {
			release.Version = version
		}

		notes, err := releaseService.GenerateReleaseNotes(ctx, release)
		if err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_generating_notes", 0,
				map[string]interface{}{
					"Error": err.Error(),
				}))
		}

		fmt.Println(trans.GetMessage("release.create_preview", 0, map[string]interface{}{
			"Version": release.Version,
			"Bump":    release.VersionBump,
		}))
		fmt.Println(trans.GetMessage("release.create_title", 0, map[string]interface{}{
			"Title": notes.Title,
		}))
		fmt.Println(trans.GetMessage("release.create_stats", 0, map[string]interface{}{
			"Features": len(release.Features),
			"Fixes":    len(release.BugFixes),
			"Breaking": len(release.Breaking),
		}))
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
		err = releaseService.CreateTag(ctx, release.Version, message)
		if err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_creating_tag", 0, map[string]interface{}{
				"Error": err.Error(),
			}))
		}

		fmt.Println(trans.GetMessage("release.tag_created", 0, map[string]interface{}{
			"Version": release.Version,
		}))

		if cmd.Bool("publish") {
			notes.Changelog = FormatReleaseMarkdown(release, notes, trans)

			fmt.Println(trans.GetMessage("release.publishing_release", 0, nil))
			err := releaseService.PublishRelease(ctx, release, notes, cmd.Bool("draft"))
			if err != nil {
				return fmt.Errorf("%s", trans.GetMessage("release.error_publishing_release", 0, map[string]interface{}{"Error": err.Error()}))
			}
			fmt.Println(trans.GetMessage("release.release_published", 0, nil))
		} else {
			fmt.Println()
			fmt.Println(trans.GetMessage("release.create_next_steps", 0, nil))
			fmt.Println(trans.GetMessage("release.create_review", 0, map[string]interface{}{
				"Version": release.Version,
			}))
			fmt.Println(trans.GetMessage("release.create_push", 0, map[string]interface{}{
				"Version": release.Version,
			}))
			fmt.Println(trans.GetMessage("release.create_push_help", 0, nil))
		}

		fmt.Println()

		return nil
	}
}
