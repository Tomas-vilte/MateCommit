package release

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

func (r *ReleaseCommandFactory) newEditCommand(trans *i18n.Translations) *cli.Command {
	return &cli.Command{
		Name:    "edit",
		Aliases: []string{"e"},
		Usage:   trans.GetMessage("release.edit_usage", 0, nil),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "version",
				Aliases:  []string{"v"},
				Usage:    trans.GetMessage("release.edit_version_flag", 0, nil),
				Required: true,
			},
			&cli.StringFlag{
				Name:    "editor",
				Aliases: []string{"e"},
				Usage:   trans.GetMessage("release.edit_editor_flag", 0, nil),
				Value:   getDefaultEditor(),
			},
			&cli.BoolFlag{
				Name:    "ai",
				Aliases: []string{"a"},
				Usage:   trans.GetMessage("release.edit_ai_flag", 0, nil),
				Value:   false,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			service, err := r.createReleaseService(ctx, trans)
			if err != nil {
				return err
			}
			return editReleaseAction(service, r.gitService, trans)(ctx, cmd)
		},
	}
}

func editReleaseAction(releaseService ports.ReleaseService, gitService ports.GitService, trans *i18n.Translations) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		version := cmd.String("version")
		editor := cmd.String("editor")
		useAI := cmd.Bool("ai")

		fmt.Println(trans.GetMessage("release.fetching_release", 0, map[string]interface{}{
			"Version": version,
		}))

		existingRelease, err := releaseService.GetRelease(ctx, version)
		if err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_fetching_release", 0, map[string]interface{}{
				"Error": err.Error(),
			}))
		}

		content := existingRelease.Body

		if content == "" || useAI {
			if content == "" {
				fmt.Println(trans.GetMessage("release.empty_body_regenerating", 0, nil))
			} else {
				fmt.Println(trans.GetMessage("release.regenerating_with_ai", 0, nil))
			}

			previousVersion, err := getPreviousVersion(version)
			if err != nil {
				fmt.Printf("⚠️  %s\n", trans.GetMessage("release.error_calculating_previous", 0, map[string]interface{}{
					"Error": err.Error(),
				}))
				if content == "" {
					content = generateReleaseTemplate(existingRelease.Name, trans)
				}
			} else {
				commits, err := gitService.GetCommitsSinceTag(ctx, previousVersion)
				if err != nil {
					fmt.Printf("⚠️  %s\n", trans.GetMessage("release.error_getting_commits", 0, map[string]interface{}{
						"Error": err.Error(),
					}))
					if content == "" {
						content = generateReleaseTemplate(existingRelease.Name, trans)
					}
				} else {
					release := &models.Release{
						Version:         version,
						PreviousVersion: previousVersion,
						AllCommits:      commits,
					}

					notes, err := releaseService.GenerateReleaseNotes(ctx, release)
					if err != nil {
						fmt.Printf("⚠️  %s\n", trans.GetMessage("release.error_generating_for_regen", 0, map[string]interface{}{
							"Error": err.Error(),
						}))
						if content == "" {
							content = generateReleaseTemplate(existingRelease.Name, trans)
						}
					} else {
						content = FormatReleaseMarkdown(release, notes, trans)
						fmt.Println(trans.GetMessage("release.notes_regenerated", 0, nil))
					}
				}
			}
		}
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("release-%s-*.md", version))
		if err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_creating_temp", 0, map[string]interface{}{
				"Error": err.Error(),
			}))
		}
		defer func() {
			if err := os.Remove(tmpFile.Name()); err != nil {
				return
			}
		}()
		_, err = tmpFile.WriteString(content)
		if err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_writing_temp", 0, map[string]interface{}{
				"Error": err.Error(),
			}))
		}
		_ = tmpFile.Close()
		fmt.Println(trans.GetMessage("release.opening_editor", 0, map[string]interface{}{
			"Editor": editor,
		}))
		editorCmd := exec.CommandContext(ctx, editor, tmpFile.Name())
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr
		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_running_editor", 0, map[string]interface{}{
				"Error": err.Error(),
			}))
		}
		editedContent, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_reading_temp", 0, map[string]interface{}{
				"Error": err.Error(),
			}))
		}
		fmt.Println(trans.GetMessage("release.updating_release", 0, map[string]interface{}{
			"Version": version,
		}))
		err = releaseService.UpdateRelease(ctx, version, string(editedContent))
		if err != nil {
			return fmt.Errorf("%s", trans.GetMessage("release.error_updating_release", 0, map[string]interface{}{
				"Error": err.Error(),
			}))
		}
		fmt.Println(trans.GetMessage("release.edit_success", 0, map[string]interface{}{
			"Version": version,
		}))
		return nil
	}
}
func generateReleaseTemplate(name string, trans *i18n.Translations) string {
	return fmt.Sprintf(`# %s
<!-- ⚠️  %s -->
<!-- 💡 %s -->
## %s
## %s
- 
- 
## %s
`,
		name,
		trans.GetMessage("release.template_warning", 0, nil),
		trans.GetMessage("release.template_tip", 0, nil),
		trans.GetMessage("release.md_summary", 0, nil),
		trans.GetMessage("release.md_highlights", 0, nil),
		trans.GetMessage("release.template_changes", 0, nil),
	)
}
func getDefaultEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if _, err := exec.LookPath("nano"); err == nil {
		return "nano"
	}
	if _, err := exec.LookPath("vim"); err == nil {
		return "vim"
	}
	return "vi"
}
