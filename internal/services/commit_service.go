package services

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"regexp"
)

type CommitService struct {
	git           ports.GitService
	ai            ports.CommitSummarizer
	ticketManager ports.TickerManager
	config        *config.Config
	trans         *i18n.Translations
}

func NewCommitService(git ports.GitService, ai ports.CommitSummarizer, ticketManager ports.TickerManager, cfg *config.Config, trans *i18n.Translations) *CommitService {
	return &CommitService{
		git:           git,
		ai:            ai,
		ticketManager: ticketManager,
		config:        cfg,
		trans:         trans,
	}
}

func (s *CommitService) GenerateSuggestions(ctx context.Context, count int) ([]models.CommitSuggestion, error) {
	var commitInfo models.CommitInfo

	changes, err := s.git.GetChangedFiles()
	if err != nil {
		return nil, err
	}

	if len(changes) == 0 {
		msg := s.trans.GetMessage("commit_service.undetected_changes", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}

	diff, err := s.git.GetDiff()
	if err != nil {
		msg := s.trans.GetMessage("commit_service.error_getting_diff", 0, map[string]interface{}{
			"Error": err,
		})
		return nil, fmt.Errorf("%s", msg)
	}

	if diff == "" {
		msg := s.trans.GetMessage("commit_service.no_differences_detected", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}

	files := make([]string, 0)
	for _, change := range changes {
		files = append(files, change.Path)
	}

	commitInfo = models.CommitInfo{
		Files: files,
		Diff:  diff,
	}

	if s.config.UseTicket {
		ticketID, err := s.getTicketIDFromBranch()
		if err != nil {
			msg := s.trans.GetMessage("commit_service.error_get_id_ticket", 0, map[string]interface{}{
				"Error": err,
			})
			return nil, fmt.Errorf("%s", msg)
		}

		ticketInfo, err := s.ticketManager.GetTicketInfo(ticketID)
		if err != nil {
			msg := s.trans.GetMessage("commit_service.error_get_ticket_info", 0, map[string]interface{}{
				"Error": err,
			})
			return nil, fmt.Errorf("%s", msg)
		}

		commitInfo.TicketInfo = ticketInfo
	}

	return s.ai.GenerateSuggestions(ctx, commitInfo, count)
}

func (s *CommitService) getTicketIDFromBranch() (string, error) {
	branchName, err := s.git.GetCurrentBranch()
	if err != nil {
		msg := s.trans.GetMessage("commit_service.error_get_name_from_branch", 0, map[string]interface{}{
			"Error": err,
		})
		return "", fmt.Errorf("%s", msg)
	}

	re := regexp.MustCompile(`([A-Za-z]+-\d+)`)
	match := re.FindString(branchName)
	if match == "" {
		msg := s.trans.GetMessage("commit_service.ticket_id_not_found_branch", 0, nil)
		return "", fmt.Errorf("%s", msg)
	}

	return match, nil
}
