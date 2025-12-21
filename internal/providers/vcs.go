package providers

import (
	"context"
	"fmt"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/git"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/vcs/github"
)

// NewVCSClient creates a VCSClient based on the configuration and automatic detection of the remote
func NewVCSClient(ctx context.Context, gitService *git.GitService, cfg *config.Config) (ports.VCSClient, error) {
	owner, repo, provider, err := gitService.GetRepoInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting repo info: %w", err)
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
