package factory

import (
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/vcs/github"
	"github.com/Tomas-vilte/MateCommit/internal/services"
)

type PRServiceFactory struct {
	config    *config.Config
	aiService ports.PRSummarizer
	trans     *i18n.Translations
}

func NewPrServiceFactory(cfg *config.Config, trans *i18n.Translations, aiService ports.PRSummarizer) *PRServiceFactory {
	return &PRServiceFactory{
		config:    cfg,
		trans:     trans,
		aiService: aiService,
	}
}

func (f *PRServiceFactory) CreatePRService() (ports.PRService, error) {
	if f.config.ActiveVCSProvider == "" {
		return nil, fmt.Errorf("provedor vcs no configurado")
	}

	vcsConfig, exists := f.config.VCSConfigs[f.config.ActiveVCSProvider]
	if !exists {
		return nil, fmt.Errorf("configuración para el proveedor de VCS '%s' no encontrada", f.config.ActiveVCSProvider)
	}

	var vcsClient ports.VCSClient

	switch vcsConfig.Provider {
	case "github":
		vcsClient = github.NewGitHubClient(vcsConfig.Owner, vcsConfig.Repo, vcsConfig.Token, f.trans)
	case "?": // bueno aca iria los otros provedores como gitlab o bitbucket, mas adelante se agregara
	default:
		return nil, fmt.Errorf("proveedor de VCS no compatible: %s", vcsConfig.Provider)
	}

	return services.NewPRService(vcsClient, f.aiService), nil
}
