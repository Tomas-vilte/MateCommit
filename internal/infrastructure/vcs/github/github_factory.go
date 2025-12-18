package github

import (
	"context"
	"fmt"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
)

// GitHubProviderFactory implementa VCSProviderFactory para GitHub
type GitHubProviderFactory struct{}

// NewGitHubProviderFactory crea una nueva factory para GitHub
func NewGitHubProviderFactory() *GitHubProviderFactory {
	return &GitHubProviderFactory{}
}

// CreateClient crea un cliente GitHub
func (f *GitHubProviderFactory) CreateClient(
	_ context.Context,
	owner, repo, token string,
	trans *i18n.Translations,
) (ports.VCSClient, error) {
	return NewGitHubClient(owner, repo, token, trans), nil
}

// ValidateConfig valida la configuración de GitHub
func (f *GitHubProviderFactory) ValidateConfig(cfg *config.VCSConfig) error {
	if cfg.Token == "" {
		return fmt.Errorf("token de github necesario")
	}
	return nil
}

// Name retorna el nombre del proveedor
func (f *GitHubProviderFactory) Name() string {
	return "github"
}
