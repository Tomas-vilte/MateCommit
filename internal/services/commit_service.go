package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"regexp"
)

type CommitService struct {
	git         ports.GitService
	ai          ports.AIProvider
	jiraService ports.TickerManager
	useTicket   bool
}

func NewCommitService(git ports.GitService, ai ports.AIProvider, jiraService ports.TickerManager, useTicket bool) *CommitService {
	return &CommitService{
		git:         git,
		ai:          ai,
		jiraService: jiraService,
		useTicket:   useTicket,
	}
}

func (s *CommitService) GenerateSuggestions(ctx context.Context, count int) ([]models.CommitSuggestion, error) {
	var commitInfo models.CommitInfo

	// Obtener los cambios en el código
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

	commitInfo = models.CommitInfo{
		Files: files,
		Diff:  diff,
	}

	if s.useTicket {
		ticketID, err := s.getTicketIDFromBranch()
		if err != nil {
			return nil, fmt.Errorf("error al obtener el ID del ticket: %v", err)
		}

		ticketInfo, err := s.jiraService.GetTicketInfo(ticketID)
		if err != nil {
			return nil, fmt.Errorf("error al obtener la información del ticket: %v", err)
		}

		commitInfo.TicketTitle = ticketInfo.Title
		commitInfo.TicketDesc = ticketInfo.Description
		commitInfo.Criteria = ticketInfo.Criteria
	}

	// Generar sugerencias de commit usando la IA
	return s.ai.GenerateSuggestions(ctx, commitInfo, count)
}

func (s *CommitService) getTicketIDFromBranch() (string, error) {
	branchName, err := s.git.GetCurrentBranch()
	if err != nil {
		return "", fmt.Errorf("error al obtener el nombre de la branch: %v", err)
	}

	re := regexp.MustCompile(`([A-Za-z]+-\d+)`)
	match := re.FindString(branchName)
	if match == "" {
		return "", fmt.Errorf("no se encontró un ID de ticket en el nombre de la branch")
	}

	return match, nil
}
