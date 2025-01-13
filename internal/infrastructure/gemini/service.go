package gemini

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"strings"
)

type GeminiService struct {
	client *genai.Client
	model  *genai.GenerativeModel
	config *config.CommitConfig
}

func NewGeminiService(ctx context.Context, apiKey string, config *config.CommitConfig) (*GeminiService, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("error creating Gemini client: %w", err)
	}

	model := client.GenerativeModel("gemini-1.5-flash")
	return &GeminiService{
		client: client,
		model:  model,
		config: config,
	}, nil
}

func (s *GeminiService) GenerateSuggestions(ctx context.Context, info models.CommitInfo, count int) ([]string, error) {
	prompt := s.generatePrompt(s.config.Locale, info, count)

	resp, err := s.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("error generating content: %w", err)
	}

	responseText := formatResponse(resp)

	return []string{responseText}, nil
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
		info.Diff)
}
