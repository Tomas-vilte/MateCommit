package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
)

type CommitService struct {
	git ports.GitService
	ai  ports.AIProvider
}

func NewCommitService(git ports.GitService, ai ports.AIProvider) *CommitService {
	return &CommitService{
		git: git,
		ai:  ai,
	}
}

func (s *CommitService) GenerateSuggestions(ctx context.Context, count int) ([]models.CommitSuggestion, error) {
	changes, err := s.git.GetChangedFiles()
	if err != nil {
		return nil, err
	}

	if len(changes) == 0 {
		return nil, fmt.Errorf("no hay cambios detectados")
	}

	diff, err := s.git.GetDiff()
	if err != nil {
		return nil, err
	}

	if diff == "" {
		return nil, errors.New("no se detectaron diferencias en los archivos")
	}

	files := make([]string, 0)
	for _, change := range changes {
		files = append(files, change.Path)
	}

	info := models.CommitInfo{
		Files: files,
		Diff:  diff,
	}

	return s.ai.GenerateSuggestions(ctx, info, count)
}
