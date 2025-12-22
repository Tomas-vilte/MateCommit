package issues

import (
	"context"
	"fmt"
	"os"

	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/ui"
	"github.com/urfave/cli/v3"
)

// newTemplateCommand creates the 'template' subcommand for template management.
func (f *IssuesCommandFactory) newTemplateCommand(t *i18n.Translations, _ *config.Config) *cli.Command {
	templateService := f.templateService

	return &cli.Command{
		Name:    "template",
		Aliases: []string{"t"},
		Usage:   t.GetMessage("issue.template_usage", 0, nil),
		Commands: []*cli.Command{
			{
				Name:    "init",
				Aliases: []string{"i"},
				Usage:   t.GetMessage("issue.template_init_usage", 0, nil),
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "force",
						Usage: t.GetMessage("issue.template_force_flag", 0, nil),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					force := cmd.Bool("force")

					ui.PrintSectionBanner(t.GetMessage("issue.template_init_banner", 0, nil))
					ui.PrintInfo(t.GetMessage("issue.template_init_info", 0, nil))

					if err := templateService.InitializeTemplates(ctx, force); err != nil {
						ui.PrintError(os.Stdout, fmt.Sprintf("%s: %v", t.GetMessage("issue.template_init_error", 0, nil), err))
						return err
					}

					templatesDir, _ := templateService.GetTemplatesDir(ctx)
					ui.PrintSuccess(os.Stdout, t.GetMessage("issue.template_init_success", 0, struct{ Dir string }{templatesDir}))

					return nil
				},
			},
			{
				Name:    "list",
				Aliases: []string{"ls", "l"},
				Usage:   t.GetMessage("issue.template_list_usage", 0, nil),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					templates, err := templateService.ListTemplates(ctx)
					if err != nil {
						ui.PrintError(os.Stdout, fmt.Sprintf("%s: %v", t.GetMessage("issue.template_list_error", 0, nil), err))
						return err
					}

					if len(templates) == 0 {
						ui.PrintWarning(t.GetMessage("issue.template_list_empty", 0, nil))
						ui.PrintInfo(t.GetMessage("issue.template_list_hint", 0, nil))
						return nil
					}

					ui.PrintSectionBanner(t.GetMessage("issue.template_list_banner", 0, nil))
					fmt.Println()

					for _, tmpl := range templates {
						ui.PrintKeyValue(tmpl.FilePath, fmt.Sprintf("%s - %s", tmpl.Name, tmpl.About))
					}

					fmt.Println()
					ui.PrintInfo(t.GetMessage("issue.template_list_usage_hint", 0, nil))

					return nil
				},
			},
		},
	}
}
