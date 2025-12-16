package release

import (
	"context"
	"fmt"

	"github.com/Tomas-vilte/MateCommit/internal/cli/completion_helper"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
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

func previewReleaseAction(releaseService ports.ReleaseService, trans *i18n.Translations) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println(trans.GetMessage("release.analyzing", 0, nil))
		fmt.Println()

		release, err := releaseService.AnalyzeNextRelease(ctx)
		if err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_analyzing", 0, map[string]interface{}{
				"Error": err.Error(),
			}))
		}

		fmt.Println(trans.GetMessage("release.previous_version", 0, map[string]interface{}{
			"Version": release.PreviousVersion,
		}))
		fmt.Println(trans.GetMessage("release.next_version", 0, map[string]interface{}{
			"Version": release.Version,
			"Bump":    release.VersionBump,
		}))
		fmt.Println()

		fmt.Println(trans.GetMessage("release.changes_summary", 0, nil))
		if len(release.Breaking) > 0 {
			fmt.Println(trans.GetMessage("release.breaking_changes", 0, map[string]interface{}{
				"Count": len(release.Breaking),
			}))
		}
		fmt.Println(trans.GetMessage("release.new_features", 0, map[string]interface{}{
			"Count": len(release.Features),
		}))
		fmt.Println(trans.GetMessage("release.bug_fixes", 0, map[string]interface{}{
			"Count": len(release.BugFixes),
		}))
		fmt.Println(trans.GetMessage("release.improvements", 0, map[string]interface{}{
			"Count": len(release.Improvements),
		}))
		fmt.Println(trans.GetMessage("release.documentation", 0, map[string]interface{}{
			"Count": len(release.Documentation),
		}))
		fmt.Println(trans.GetMessage("release.total_commits", 0, map[string]interface{}{
			"Count": len(release.AllCommits),
		}))
		fmt.Println()

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

		return nil
	}
}
