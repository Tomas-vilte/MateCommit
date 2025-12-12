package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai"
	"google.golang.org/genai"
)

type GeminiService struct {
	client *genai.Client
	config *config.Config
	trans  *i18n.Translations
}

func NewGeminiService(ctx context.Context, cfg *config.Config, trans *i18n.Translations) (*GeminiService, error) {
	if cfg.GeminiAPIKey == "" {
		msg := trans.GetMessage("error_missing_api_key", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.GeminiAPIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		msg := trans.GetMessage("error_gemini_client", 0, map[string]interface{}{
			"Error": err,
		})
		return nil, fmt.Errorf("%s", msg)
	}

	return &GeminiService{
		client: client,
		config: cfg,
		trans:  trans,
	}, nil
}

func (s *GeminiService) GenerateSuggestions(ctx context.Context, info models.CommitInfo, count int) ([]models.CommitSuggestion, error) {
	if count <= 0 {
		msg := s.trans.GetMessage("error_invalid_suggestion_count", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}

	if len(info.Files) == 0 {
		msg := s.trans.GetMessage("error_no_files", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}

	prompt := s.generatePrompt(s.config.Language, info, count)
	modelName := string(s.config.AIConfig.Models[config.AIGemini])

	resp, err := s.client.Models.GenerateContent(ctx, modelName, genai.Text(prompt), &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
	})
	if err != nil {
		msg := s.trans.GetMessage("error_generating_content", 0, map[string]interface{}{
			"Error": err.Error(),
		})
		return nil, fmt.Errorf("%s", msg)
	}

	suggestions := s.parseSuggestions(resp)
	if len(suggestions) == 0 {
		msg := s.trans.GetMessage("error_no_suggestions", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}

	if info.IssueInfo != nil && info.IssueInfo.Number > 0 {
		suggestions = s.ensureIssueReference(suggestions, info.IssueInfo.Number)
	}

	return suggestions, nil
}

func (s *GeminiService) generatePrompt(locale string, info models.CommitInfo, count int) string {
	promptTemplate := ai.GetCommitPromptTemplate(locale, info.TicketInfo != nil && info.TicketInfo.TicketTitle != "")

	ticketInfo := ""
	if info.TicketInfo != nil && info.TicketInfo.TicketTitle != "" {
		ticketInfo = fmt.Sprintf("\nTicket Title: %s\nTicket Description: %s\nAcceptance Criteria: %s",
			info.TicketInfo.TicketTitle,
			info.TicketInfo.TitleDesc,
			strings.Join(info.TicketInfo.Criteria, ", "))
	}

	issueInstructions := ""
	if info.IssueInfo != nil && info.IssueInfo.Number > 0 {
		num := info.IssueInfo.Number
		issueInstructions = fmt.Sprintf(ai.GetIssueReferenceInstructions(locale), num, num, num, num, num, num, num, num)
	} else {
		issueInstructions = "No hay issue asociado, no incluyas referencias de issues en el título."
	}

	if info.TicketInfo != nil && info.TicketInfo.TicketTitle != "" {
		return fmt.Sprintf(promptTemplate,
			count,
			issueInstructions,
			formatChanges(info.Files),
			info.Diff,
			ticketInfo,
		)
	}

	return fmt.Sprintf(promptTemplate,
		count,
		issueInstructions,
		formatChanges(info.Files),
		info.Diff,
	)
}

func formatChanges(files []string) string {
	if len(files) == 0 {
		return ""
	}
	formattedFiles := make([]string, len(files))
	for i, file := range files {
		formattedFiles[i] = fmt.Sprintf("- %s", file)
	}
	return strings.Join(formattedFiles, "\n")
}

// formatResponse formatea la respuesta de la API de Gemini en una cadena.
func formatResponse(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 {
		return ""
	}

	var formattedContent strings.Builder
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				if part.Text != "" {
					formattedContent.WriteString(part.Text)
				}
			}
		}
	}
	return formattedContent.String()
}

func (s *GeminiService) parseSuggestions(resp *genai.GenerateContentResponse) []models.CommitSuggestion {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil
	}

	responseText := formatResponse(resp)
	if responseText == "" {
		return nil
	}

	// Clean up markdown code blocks if present (Gemini might wrap JSON in ```json ... ```)
	responseText = strings.TrimSpace(responseText)
	if strings.HasPrefix(responseText, "```json") {
		responseText = strings.TrimPrefix(responseText, "```json")
		responseText = strings.TrimSuffix(responseText, "```")
	}
	if strings.HasPrefix(responseText, "```") {
		responseText = strings.TrimPrefix(responseText, "```")
		responseText = strings.TrimSuffix(responseText, "```")
	}

	var suggestions []models.CommitSuggestion
	if err := json.Unmarshal([]byte(responseText), &suggestions); err != nil {
		// Fallback or log error? For now, return empty or try to parse what we can.
		// Since we can't easily log here without a logger instance in the method signature (only have it in struct but it is not a logger, it is translations),
		// we might return nil and let the caller handle "no suggestions".
		fmt.Printf("Error unmarshalling JSON from Gemini: %v\nInput: %s\n", err, responseText) // Temporary debug
		return nil
	}

	return suggestions
}

// ensureIssueReference asegura que todas las sugerencias incluyan la referencia al issue correcta
func (s *GeminiService) ensureIssueReference(suggestions []models.CommitSuggestion, issueNumber int) []models.CommitSuggestion {
	issuePattern := regexp.MustCompile(`\(#\d+\)`)

	for i := range suggestions {
		title := suggestions[i].CommitTitle
		title = strings.TrimSpace(title)

		if strings.Contains(title, fmt.Sprintf("(#%d)", issueNumber)) ||
			strings.Contains(title, fmt.Sprintf("fixes #%d", issueNumber)) ||
			strings.Contains(title, fmt.Sprintf("closes #%d", issueNumber)) {
			continue
		}

		if issuePattern.MatchString(title) {
			title = issuePattern.ReplaceAllString(title, fmt.Sprintf("(#%d)", issueNumber))
			suggestions[i].CommitTitle = title
			continue
		}

		suggestions[i].CommitTitle = fmt.Sprintf("%s (#%d)", title, issueNumber)
	}

	return suggestions
}
