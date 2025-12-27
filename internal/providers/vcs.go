package providers

import (
	"context"
	"fmt"

	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/git"
	"github.com/thomas-vilte/matecommit/internal/github"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/ports"
)

// NewVCSClient creates a VCSClient based on the configuration and automatic detection of the remote
// Returns nil, nil when not in a git repository (this is not an error condition)
func NewVCSClient(ctx context.Context, gitService *git.GitService, cfg *config.Config) (ports.VCSClient, error) {
	owner, repo, provider, err := gitService.GetRepoInfo(ctx)
	if err != nil {
		// Not in a git repository or no remote configured - this is OK
		logger.Debug(ctx, "VCS client not available", "reason", "not in a git repository or no remote configured")
		return nil, nil
	}

	vcsConfig, exists := cfg.VCSConfigs[provider]
	if !exists {
		vcsConfig, exists = cfg.VCSConfigs[cfg.ActiveVCSProvider]
		if !exists {
			return nil, fmt.Errorf("VCS provider '%s' not configured", provider)
		}
		provider = cfg.ActiveVCSProvider
	}

	switch provider {
	case "github":
		return github.NewGitHubClient(owner, repo, vcsConfig.Token), nil
	default:
		return nil, fmt.Errorf("VCS provider '%s' not supported", provider)
	}
}
