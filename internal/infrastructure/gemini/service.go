package gemini

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"strings"
)

type GeminiService struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewGeminiService(ctx context.Context, apiKey string) *GeminiService {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		panic(err)
	}

	model := client.GenerativeModel("gemini-2.0-flash-exp")
	return &GeminiService{
		client: client,
		model:  model,
	}
}

func (s *GeminiService) GenerateCommitMessage(ctx context.Context, info models.CommitInfo) (string, error) {
	prompt := fmt.Sprintf(`Genera un mensaje de commit conciso y descriptivo basado en estos cambios:
		Archivos modificados:
		%v
		
		Diff:
		%s
		
		Por favor:
		1. Usa el formato de Conventional Commits (feat/fix/docs/etc)
		2. El mensaje no debe exceder 72 caracteres
		3. Debe ser claro y descriptivo
		4. No incluyas puntos finales ni saltos de línea`,
		formatChanges(info.Files),
		info.Diff)

	resp, err := s.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	return formatResponse(resp), nil
}

func formatChanges(files []string) string {
	var result string
	for _, file := range files {
		result += fmt.Sprintf("- %s\n", file)
	}
	return result
}

func formatResponse(resp *genai.GenerateContentResponse) string {
	var formattedContent strings.Builder
	if resp != nil && resp.Candidates != nil {
		for _, cand := range resp.Candidates {
			if cand.Content != nil {
				for _, part := range cand.Content.Parts {
					formattedContent.WriteString(fmt.Sprintf("%v", part))
				}
			}
		}
	}

	return formattedContent.String()
}
