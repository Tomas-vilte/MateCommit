package release

import (
	"context"
	"fmt"

	"github.com/Tomas-vilte/MateCommit/internal/cli/completion_helper"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/ui"
	"github.com/urfave/cli/v3"
)

func (r *ReleaseCommandFactory) newPushCommand(trans *i18n.Translations) *cli.Command {
	return &cli.Command{
		Name:  "push",
		Usage: trans.GetMessage("release.push_usage", 0, nil),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   trans.GetMessage("release.push_version_flag", 0, nil),
			},
		},
		ShellComplete: completion_helper.DefaultFlagComplete,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			service, err := r.createReleaseService(ctx, trans)
			if err != nil {
				return err
			}
			return pushReleaseAction(service, trans)(ctx, cmd)
		},
	}
}

func pushReleaseAction(releaseSvc releaseService, trans *i18n.Translations) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		version := cmd.String("version")

		if version == "" {
			release, err := releaseSvc.AnalyzeNextRelease(ctx)
			if err != nil {
				return fmt.Errorf("%s", trans.GetMessage("release.error_analyzing", 0, map[string]interface{}{"Error": err.Error()}))
			}
			version = release.Version
		}

		fmt.Println(trans.GetMessage("release.pushing_tag", 0, map[string]interface{}{"Version": version}))

		err := releaseSvc.PushTag(ctx, version)
		if err != nil {
			ui.HandleAppError(err, trans)
			return fmt.Errorf("%s", trans.GetMessage("release.error_pushing_tag", 0, map[string]interface{}{"Error": err.Error()}))
		}

		fmt.Println(trans.GetMessage("release.push_success", 0, map[string]interface{}{"Version": version}))
		return nil
	}
}
