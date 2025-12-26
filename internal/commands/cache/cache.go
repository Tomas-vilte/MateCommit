package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/thomas-vilte/matecommit/internal/cache"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

type CacheCommand struct{}

func NewCacheCommand() *CacheCommand {
	return &CacheCommand{}
}

func (c *CacheCommand) CreateCommand(t *i18n.Translations, _ *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "cache",
		Usage: t.GetMessage("cache.usage", 0, nil),
		Commands: []*cli.Command{
			{
				Name:  "clean",
				Usage: t.GetMessage("cache.clean_usage", 0, nil),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					cacheService, err := cache.NewCache(24 * time.Hour)
					if err != nil {
						return fmt.Errorf(t.GetMessage("cache.error_init", 0, nil)+": %w", err)
					}

					if err := cacheService.Clean(); err != nil {
						return fmt.Errorf(t.GetMessage("cache.error_clean", 0, nil)+": %w", err)
					}

					green := color.New(color.FgGreen, color.Bold)
					_, _ = green.Printf("✓ %s\n", t.GetMessage("cache.cleaned", 0, nil))
					return nil
				},
			},
		},
	}
}
