package gemini

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"strings"
)

type GeminiService struct {
	client *genai.Client
	model  *genai.GenerativeModel
	config *config.CommitConfig
	trans  *i18n.Translations
}

func NewGeminiService(ctx context.Context, apiKey string, config *config.CommitConfig, trans *i18n.Translations) (*GeminiService, error) {
	if apiKey == "" {
		msg := trans.GetMessage("error_missing_api_key", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	model := client.GenerativeModel("gemini-1.5-flash")
	return &GeminiService{
		client: client,
		model:  model,
		config: config,
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

	prompt := s.generatePrompt(s.config.Locale, info, count)
	resp, err := s.model.GenerateContent(ctx, genai.Text(prompt))
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

	return suggestions, nil
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

func formatResponse(resp *genai.GenerateContentResponse) string {
	if resp == nil || resp.Candidates == nil {
		return ""
	}

	var formattedContent strings.Builder
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				formattedContent.WriteString(fmt.Sprintf("%v", part))
			}
		}
	}
	return formattedContent.String()
}

func (s *GeminiService) getSuggestionDelimiter() string {
	return s.trans.GetMessage("suggestion_delimiter", 0, nil)
}

func (s *GeminiService) getFilesPrefix() string {
	return s.trans.GetMessage("files_prefix", 0, nil)
}

func (s *GeminiService) parseSuggestions(resp *genai.GenerateContentResponse) []models.CommitSuggestion {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil
	}

	responseText := formatResponse(resp)
	if responseText == "" {
		return nil
	}

	suggestions := make([]models.CommitSuggestion, 0)
	delimiter := s.getSuggestionDelimiter()
	parts := strings.Split(responseText, delimiter)

	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}

		suggestion := s.parseSuggestionPart(part)
		if suggestion != nil {
			suggestions = append(suggestions, *suggestion)
		}
	}

	return suggestions
}

func (s *GeminiService) parseSuggestionPart(part string) *models.CommitSuggestion {
	lines := strings.Split(strings.TrimSpace(part), "\n")
	if len(lines) < 3 {
		return nil
	}

	suggestion := &models.CommitSuggestion{}
	//suggestion.CommitTitle = strings.TrimPrefix(lines[1], "Commit: ")
	suggestion.CommitTitle = strings.TrimSpace(lines[1])

	prefixFiles := s.getFilesPrefix()
	if filesPart := strings.TrimPrefix(lines[2], prefixFiles); filesPart != "" {
		files := strings.Split(filesPart, ",")
		suggestion.Files = make([]string, 0, len(files))
		for _, file := range files {
			if trimmed := strings.TrimSpace(file); trimmed != "" {
				suggestion.Files = append(suggestion.Files, trimmed)
			}
		}
	}

	var explanation strings.Builder
	for _, line := range lines[3:] {
		explanation.WriteString(line)
		explanation.WriteString("\n")
	}
	suggestion.Explanation = strings.TrimSpace(explanation.String())

	return suggestion
}

func (s *GeminiService) generatePrompt(locale config.CommitLocale, info models.CommitInfo, count int) string {
	var promptTemplate string
	switch locale.Lang {
	case "es":
		promptTemplate = promptTemplateES
		if s.config.UseEmoji {
			promptTemplate = strings.Replace(promptTemplate, "Commit: [tipo]: [mensaje]", "Commit: [emoji] [tipo]: [mensaje]", 1)
		}
	case "en":
		promptTemplate = promptTemplateEN
		if s.config.UseEmoji {
			promptTemplate = strings.Replace(promptTemplate, "Commit: [type]: [message]", "Commit: [emoji] [type]: [message]", 1)
		}
	default:
		promptTemplate = promptTemplateEN
		if s.config.UseEmoji {
			promptTemplate = strings.Replace(promptTemplate, "Commit: [type]: [message]", "Commit: [emoji] [type]: [message]", 1)
		}
	}

	return fmt.Sprintf(promptTemplate,
		count,
		count,
		formatChanges(info.Files),
		info.Diff,
	)
}
