package services

import (
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"strings"
)

type (
	GitService interface {
		GetChangedFiles() ([]models.GitChange, error)
		GetDiff() (string, error)
		CreateCommit(message string) error
	}
)

type CommitService struct {
	git GitService
	ai  ports.AIProvider
}

func NewCommitService(git GitService, ai ports.AIProvider) *CommitService {
	return &CommitService{
		git: git,
		ai:  ai,
	}
}

func (s *CommitService) GenerateAndCommit(ctx context.Context) (string, error) {
	changes, err := s.git.GetChangedFiles()
	if err != nil {
		return "", err
	}

	diff, err := s.git.GetDiff()
	if err != nil {
		return "", err
	}

	files := make([]string, 0)
	for _, change := range changes {
		files = append(files, change.Path)
	}

	info := models.CommitInfo{
		Files: files,
		Diff:  diff,
	}

	message, err := s.ai.GenerateCommitMessage(ctx, info)
	if err != nil {
		return "", err
	}

	message = strings.TrimSpace(message)

	err = s.git.CreateCommit(message)
	if err != nil {
		return "", err
	}
	return message, nil
}
