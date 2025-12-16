package release

import (
	"context"
	"fmt"
	"os"

	"github.com/Tomas-vilte/MateCommit/internal/cli/completion_helper"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
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

func generateReleaseAction(releaseService ports.ReleaseService, trans *i18n.Translations) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println(trans.GetMessage("release.generating", 0, nil))
		fmt.Println()

		release, err := releaseService.AnalyzeNextRelease(ctx)
		if err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_analyzing", 0, map[string]interface{}{
				"Error": err.Error(),
			}))
		}

		if err := releaseService.EnrichReleaseContext(ctx, release); err != nil {
			fmt.Printf("⚠️  %s\n", trans.GetMessage("release.warning_enrich_context", 0, map[string]interface{}{
				"Error": err.Error(),
			}))
		}

		notes, err := releaseService.GenerateReleaseNotes(ctx, release)
		if err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_generating_notes", 0,
				map[string]interface{}{
					"Error": err.Error(),
				}))
		}

		content := FormatReleaseMarkdown(release, notes, trans)

		outputFile := cmd.String("output")
		err = os.WriteFile(outputFile, []byte(content), 0644)
		if err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_writing_file", 0, map[string]interface{}{
				"Error": err.Error(),
			}))
		}

		fmt.Println(trans.GetMessage("release.notes_saved", 0, map[string]interface{}{
			"File": outputFile,
		}))
		fmt.Println(trans.GetMessage("release.version_label", 0, map[string]interface{}{
			"Version": release.Version,
		}))
		fmt.Println()

		return nil
	}
}
