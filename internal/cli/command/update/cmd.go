package update

import (
	"context"
	"fmt"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/services"
	"github.com/urfave/cli/v3"
)

type UpdateCommandFactory struct {
	currentVersion string
}

func NewUpdateCommandFactory(currentVersion string) *UpdateCommandFactory {
	return &UpdateCommandFactory{
		currentVersion: currentVersion,
	}
}

func (f *UpdateCommandFactory) CreateCommand(trans *i18n.Translations, _ *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: trans.GetMessage("update.usage", 0, nil),
		Action: func(ctx context.Context, command *cli.Command) error {
			updater := services.NewVersionUpdater(f.currentVersion, trans)

			fmt.Println(trans.GetMessage("update.updating", 0, nil))
			if err := updater.UpdateCLI(ctx); err != nil {
				return err
			}

			fmt.Println(trans.GetMessage("update.success", 0, nil))
			return nil
		},
	}
}
