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
	config     *config.Config
	aiService  ports.PRSummarizer
	trans      *i18n.Translations
	gitService ports.GitService
}

func NewPrServiceFactory(cfg *config.Config, trans *i18n.Translations, aiService ports.PRSummarizer, gitService ports.GitService) *PRServiceFactory {
	return &PRServiceFactory{
		config:     cfg,
		trans:      trans,
		aiService:  aiService,
		gitService: gitService,
	}
}

func (f *PRServiceFactory) CreatePRService() (ports.PRService, error) {
	owner, repo, provider, err := f.gitService.GetRepoInfo()
	if err != nil {
		return nil, fmt.Errorf("error al obtener la informacion del repositorio: %w", err)
	}

	vcsConfig, exists := f.config.VCSConfigs[provider]
	if !exists {
		if f.config.ActiveVCSProvider != "" {
			vcsConfig, exists = f.config.VCSConfigs[f.config.ActiveVCSProvider]
			if !exists {
				return nil, fmt.Errorf("configuración para el proveedor de VCS '%s' no encontrada", f.config.ActiveVCSProvider)
			}
			provider = f.config.ActiveVCSProvider
		} else {
			return nil, fmt.Errorf("proveedor de VCS '%s' detectado automáticamente pero no configurado. Use 'matecommit config set-vcs --provider %s --token <token>' para configurarlo", provider, provider)
		}
	}

	var vcsClient ports.VCSClient

	switch provider {
	case "github":
		vcsClient = github.NewGitHubClient(owner, repo, vcsConfig.Token, f.trans)
	default:
		return nil, fmt.Errorf("proveedor de VCS no compatible: %s", provider)
	}

	return services.NewPRService(vcsClient, f.aiService, f.trans), nil
}
