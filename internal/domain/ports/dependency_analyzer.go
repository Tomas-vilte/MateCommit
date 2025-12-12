package ports

import (
	"context"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
)

// DependencyAnalyzer define la interfaz para analizar dependencias de diferentes lenguajes
type DependencyAnalyzer interface {
	// CanHandle detecta si este analyzer puede manejar el proyecto
	CanHandle(ctx context.Context, vcsClient VCSClient, previousTag, currentTag string) bool

	// AnalyzeChanges analiza cambios de dependencias entre dos versiones
	AnalyzeChanges(ctx context.Context, vcsClient VCSClient, previousTag, currentTag string) ([]models.DependencyChange, error)

	// Name retorna el nombre del gestor de dependencias
	Name() string
}
