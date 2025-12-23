package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-vilte/matecommit/internal/models"
)

func TestRenderPrompt(t *testing.T) {
	t.Run("Success - Render commit prompt with all fields", func(t *testing.T) {
		data := PromptData{
			Count:        3,
			Files:        "main.go, utils.go",
			Diff:         "diff --git a/main.go...",
			Ticket:       "JIRA-123",
			History:      "feat: previous commit",
			Instructions: "Follow conventional commits",
			IssueNumber:  42,
		}

		result, err := RenderPrompt("test", promptTemplateWithTicketEN, data)

		require.NoError(t, err)
		assert.Contains(t, result, "3 commit message suggestions")
		assert.Contains(t, result, "main.go, utils.go")
		assert.Contains(t, result, "JIRA-123")
		assert.Contains(t, result, "feat: previous commit")
		assert.Contains(t, result, "Follow conventional commits")
	})

	t.Run("Success - Render PR prompt", func(t *testing.T) {
		data := PromptData{
			PRContent: "## Changes\n- Added authentication\n- Fixed bug in login",
		}

		result, err := RenderPrompt("pr_test", prPromptTemplateEN, data)

		require.NoError(t, err)
		assert.Contains(t, result, "Added authentication")
		assert.Contains(t, result, "Fixed bug in login")
		assert.Contains(t, result, "Pull Request summary")
	})

	t.Run("Success - Render release notes prompt", func(t *testing.T) {
		data := PromptData{
			RepoOwner:      "thomas-vilte",
			RepoName:       "matecommit",
			CurrentVersion: "v1.0.0",
			LatestVersion:  "v1.1.0",
			ReleaseDate:    "2024-01-15",
			Changelog:      "feat: add new feature\nfix: resolve bug",
		}

		result, err := RenderPrompt("release_test", releasePromptTemplateEN, data)

		require.NoError(t, err)
		assert.Contains(t, result, "thomas-vilte/matecommit")
		assert.Contains(t, result, "v1.0.0")
		assert.Contains(t, result, "v1.1.0")
		assert.Contains(t, result, "2024-01-15")
		assert.Contains(t, result, "feat: add new feature")
	})

	t.Run("Error - Invalid template syntax", func(t *testing.T) {
		invalidTemplate := "Hello {{.Name"
		data := PromptData{}

		result, err := RenderPrompt("invalid", invalidTemplate, data)

		assert.Error(t, err)
		assert.Empty(t, result)
		assert.Contains(t, err.Error(), "error parsing template")
	})

	t.Run("Error - Missing field in data", func(t *testing.T) {
		template := "Count: {{.Count}}, Missing: {{.NonExistent}}"
		data := PromptData{Count: 5}

		result, err := RenderPrompt("missing_field", template, data)

		assert.Error(t, err)
		assert.Empty(t, result)
		assert.Contains(t, err.Error(), "error executing template")
	})

	t.Run("Success - Empty optional fields", func(t *testing.T) {
		data := PromptData{
			Count: 1,
			Files: "",
			Diff:  "",
		}

		result, err := RenderPrompt("empty_fields", promptTemplateWithoutTicketEN, data)

		require.NoError(t, err)
		assert.Contains(t, result, "1 commit message suggestions")
	})
}

func TestGetPRPromptTemplate(t *testing.T) {
	t.Run("English template contains key instructions", func(t *testing.T) {
		result := GetPRPromptTemplate("en")

		assert.Contains(t, result, "Senior Tech Lead")
		assert.Contains(t, result, "No Hallucinations")
		assert.Contains(t, result, "first person")
		assert.Contains(t, result, "valid JSON")
		assert.NotContains(t, result, "español")
	})

	t.Run("Spanish template contains key instructions in Spanish", func(t *testing.T) {
		result := GetPRPromptTemplate("es")

		assert.Contains(t, result, "Desarrollador Senior")
		assert.Contains(t, result, "Cero alucinaciones")
		assert.Contains(t, result, "primera persona")
		assert.Contains(t, result, "JSON crudo")
		assert.Contains(t, result, "ESPAÑOL")
	})

	t.Run("Unknown language defaults to English", func(t *testing.T) {
		result := GetPRPromptTemplate("fr")
		expected := GetPRPromptTemplate("en")

		assert.Equal(t, expected, result)
	})
}

func TestGetCommitPromptTemplate(t *testing.T) {
	t.Run("English with ticket contains ticket validation", func(t *testing.T) {
		result := GetCommitPromptTemplate("en", true)

		assert.Contains(t, result, "Ticket/Issue")
		assert.Contains(t, result, "Requirements Validation")
		assert.Contains(t, result, "completed_indices")
		assert.Contains(t, result, "Conventional Commits")
	})

	t.Run("English without ticket does not mention requirements", func(t *testing.T) {
		result := GetCommitPromptTemplate("en", false)

		assert.NotContains(t, result, "Requirements Validation")
		assert.NotContains(t, result, "completed_indices")
		assert.Contains(t, result, "Git Specialist")
	})

	t.Run("Spanish with ticket contains Spanish instructions", func(t *testing.T) {
		result := GetCommitPromptTemplate("es", true)

		assert.Contains(t, result, "Ticket/Issue")
		assert.Contains(t, result, "Validación de Requerimientos")
		assert.Contains(t, result, "ESPAÑOL")
		assert.Contains(t, result, "completed_indices")
	})

	t.Run("Spanish without ticket is in Spanish", func(t *testing.T) {
		result := GetCommitPromptTemplate("es", false)

		assert.Contains(t, result, "especialista en Git")
		assert.Contains(t, result, "ESPAÑOL")
		assert.NotContains(t, result, "Ticket/Issue:")
	})
}

func TestGetReleasePromptTemplate(t *testing.T) {
	t.Run("English template has noise filtering instructions", func(t *testing.T) {
		result := GetReleasePromptTemplate("en")

		assert.Contains(t, result, "TECHNICAL NOISE FILTERING")
		assert.Contains(t, result, "INTELLIGENT GROUPING")
		assert.Contains(t, result, "Keep a Changelog")
		assert.Contains(t, result, "highlights")
		assert.Contains(t, result, "breaking_changes")
	})

	t.Run("Spanish template has Spanish instructions", func(t *testing.T) {
		result := GetReleasePromptTemplate("es")

		assert.Contains(t, result, "FILTRADO DE RUIDO TÉCNICO")
		assert.Contains(t, result, "AGRUPACIÓN INTELIGENTE")
		assert.Contains(t, result, "Keep a Changelog")
		assert.Contains(t, result, "ESPAÑOL ARGENTINO")
		assert.Contains(t, result, "highlights")
	})
}

func TestGetIssuePromptTemplate(t *testing.T) {
	t.Run("English template has proper structure", func(t *testing.T) {
		result := GetIssuePromptTemplate("en")

		assert.Contains(t, result, "Senior Tech Lead")
		assert.Contains(t, result, "Context (Motivation)")
		assert.Contains(t, result, "Technical Details")
		assert.Contains(t, result, "Impact")
		assert.Contains(t, result, "No Emojis")
	})

	t.Run("Spanish template is in Spanish", func(t *testing.T) {
		result := GetIssuePromptTemplate("es")

		assert.Contains(t, result, "Tech Lead")
		assert.Contains(t, result, "Contexto")
		assert.Contains(t, result, "Detalles Técnicos")
		assert.Contains(t, result, "Impacto")
		assert.Contains(t, result, "ESPAÑOL")
	})
}

func TestGetIssueReferenceInstructions(t *testing.T) {
	t.Run("English instructions contain examples", func(t *testing.T) {
		result := GetIssueReferenceInstructions("en")

		assert.Contains(t, result, "feat: add dark mode support")
		assert.Contains(t, result, "fix: resolve authentication error")
		assert.Contains(t, result, "MUST include the reference")
		assert.Contains(t, result, "{{.IssueNumber}}")
	})

	t.Run("Spanish instructions are in Spanish", func(t *testing.T) {
		result := GetIssueReferenceInstructions("es")

		assert.Contains(t, result, "DEBES incluir la referencia")
		assert.Contains(t, result, "{{.IssueNumber}}")
		assert.Contains(t, result, "feat:")
		assert.Contains(t, result, "fix:")
	})

	t.Run("Can be rendered with issue number", func(t *testing.T) {
		instructions := GetIssueReferenceInstructions("en")
		data := struct{ IssueNumber int }{IssueNumber: 42}

		result, err := RenderPrompt("issue_ref", instructions, data)

		require.NoError(t, err)
		assert.Contains(t, result, "#42")
		assert.NotContains(t, result, "{{.IssueNumber}}")
	})
}

func TestFormatTemplateForPrompt(t *testing.T) {
	t.Run("Formats issue template with body content in English", func(t *testing.T) {
		template := &models.IssueTemplate{
			Name:        "bug_report",
			BodyContent: "## Bug Description\n\nDescribe the bug...",
		}

		result := FormatTemplateForPrompt(template, "en", "issue")

		assert.Contains(t, result, "Project Issue Template")
		assert.Contains(t, result, "Template Name: bug_report")
		assert.Contains(t, result, "Template Structure:")
		assert.Contains(t, result, "## Bug Description")
		assert.Contains(t, result, "```markdown")
		assert.Contains(t, result, "MUST follow its structure")
	})

	t.Run("Formats PR template in Spanish", func(t *testing.T) {
		template := &models.IssueTemplate{
			Name:        "pull_request",
			BodyContent: "## Cambios\n\nDescribe los cambios...",
		}

		result := FormatTemplateForPrompt(template, "es", "pr")

		assert.Contains(t, result, "Template de PR del Proyecto")
		assert.Contains(t, result, "Nombre del Template: pull_request")
		assert.Contains(t, result, "Estructura del Template:")
		assert.Contains(t, result, "## Cambios")
		assert.Contains(t, result, "DEBES seguir su estructura")
	})

	t.Run("Handles template with About field", func(t *testing.T) {
		template := &models.IssueTemplate{
			Name:        "feature_request",
			About:       "Template for feature requests",
			BodyContent: "## Feature Description",
		}

		result := FormatTemplateForPrompt(template, "en", "issue")

		assert.Contains(t, result, "Template Description: Template for feature requests")
		assert.Contains(t, result, "## Feature Description")
	})

	t.Run("Returns empty string for nil template", func(t *testing.T) {
		result := FormatTemplateForPrompt(nil, "en", "issue")

		assert.Empty(t, result)
	})

	t.Run("Defaults to English for empty language", func(t *testing.T) {
		template := &models.IssueTemplate{
			Name:        "test",
			BodyContent: "Test content",
		}

		result := FormatTemplateForPrompt(template, "", "issue")

		assert.Contains(t, result, "Project Issue Template")
		assert.NotContains(t, result, "Template de Issue")
	})

	t.Run("Handles YAML template without body content", func(t *testing.T) {
		template := &models.IssueTemplate{
			Name:  "yaml_template",
			About: "YAML based template",
			Body:  map[string]interface{}{"type": "markdown"},
		}

		result := FormatTemplateForPrompt(template, "en", "issue")

		assert.Contains(t, result, "GitHub Issue Form (YAML)")
		assert.Contains(t, result, "defines specific fields")
	})
}

func TestFormatIssuesForPrompt(t *testing.T) {
	t.Run("Formats multiple issues in English", func(t *testing.T) {
		issues := []models.Issue{
			{
				Number:      42,
				Title:       "Add authentication",
				Description: "We need OAuth2 support for third-party integrations",
			},
			{
				Number:      43,
				Title:       "Fix memory leak",
				Description: "Application crashes after 24 hours of runtime",
			},
		}

		result := FormatIssuesForPrompt(issues, "en")

		assert.Contains(t, result, "Issue #42: Add authentication")
		assert.Contains(t, result, "Description: We need OAuth2 support")
		assert.Contains(t, result, "Issue #43: Fix memory leak")
		assert.Contains(t, result, "Description: Application crashes")
	})

	t.Run("Formats issues in Spanish", func(t *testing.T) {
		issues := []models.Issue{
			{
				Number:      10,
				Title:       "Agregar modo oscuro",
				Description: "Los usuarios quieren modo oscuro",
			},
		}

		result := FormatIssuesForPrompt(issues, "es")

		assert.Contains(t, result, "Issue #10: Agregar modo oscuro")
		assert.Contains(t, result, "Descripción: Los usuarios quieren modo oscuro")
	})

	t.Run("Truncates long descriptions", func(t *testing.T) {
		longDesc := strings.Repeat("a", 250)
		issues := []models.Issue{
			{
				Number:      1,
				Title:       "Test",
				Description: longDesc,
			},
		}

		result := FormatIssuesForPrompt(issues, "en")

		assert.Contains(t, result, "Issue #1: Test")
		assert.Contains(t, result, "...")
		assert.Less(t, len(result), len(longDesc)+100)
	})

	t.Run("Handles issues without description", func(t *testing.T) {
		issues := []models.Issue{
			{
				Number:      5,
				Title:       "No description issue",
				Description: "",
			},
		}

		result := FormatIssuesForPrompt(issues, "en")

		assert.Contains(t, result, "Issue #5: No description issue")
		assert.NotContains(t, result, "Description:")
	})

	t.Run("Returns empty string for empty issues list", func(t *testing.T) {
		result := FormatIssuesForPrompt([]models.Issue{}, "en")

		assert.Empty(t, result)
	})

	t.Run("Returns empty string for nil issues list", func(t *testing.T) {
		result := FormatIssuesForPrompt(nil, "en")

		assert.Empty(t, result)
	})
}

func TestGetPRIssueContextInstructions(t *testing.T) {
	t.Run("English instructions contain closing keywords", func(t *testing.T) {
		result := GetPRIssueContextInstructions("en")

		assert.Contains(t, result, "Fixes #N")
		assert.Contains(t, result, "Closes #N")
		assert.Contains(t, result, "Relates to #N")
		assert.Contains(t, result, "MUST include at the BEGINNING")
		assert.Contains(t, result, "{{.RelatedIssues}}")
	})

	t.Run("Spanish instructions are in Spanish", func(t *testing.T) {
		result := GetPRIssueContextInstructions("es")

		assert.Contains(t, result, "Fixes #N")
		assert.Contains(t, result, "Closes #N")
		assert.Contains(t, result, "DEBES incluir AL INICIO")
		assert.Contains(t, result, "{{.RelatedIssues}}")
	})

	t.Run("Can be rendered with related issues", func(t *testing.T) {
		instructions := GetPRIssueContextInstructions("en")
		data := struct{ RelatedIssues string }{
			RelatedIssues: "- Issue #42: Add feature\n- Issue #43: Fix bug",
		}

		result, err := RenderPrompt("pr_issues", instructions, data)

		require.NoError(t, err)
		assert.Contains(t, result, "Issue #42: Add feature")
		assert.Contains(t, result, "Issue #43: Fix bug")
	})
}

func TestGetTemplateInstructions(t *testing.T) {
	t.Run("English instructions are clear", func(t *testing.T) {
		result := GetTemplateInstructions("en")

		assert.Contains(t, result, "Project Template")
		assert.Contains(t, result, "MUST follow")
		assert.Contains(t, result, "structure and format")
	})

	t.Run("Spanish instructions are in Spanish", func(t *testing.T) {
		result := GetTemplateInstructions("es")

		assert.Contains(t, result, "Template del Proyecto")
		assert.Contains(t, result, "DEBES seguir")
		assert.Contains(t, result, "estructura y formato")
	})
}

func TestGetPRTemplateInstructions(t *testing.T) {
	t.Run("English PR instructions", func(t *testing.T) {
		result := GetPRTemplateInstructions("en")

		assert.Contains(t, result, "PR template")
		assert.Contains(t, result, "MUST follow")
	})

	t.Run("Spanish PR instructions", func(t *testing.T) {
		result := GetPRTemplateInstructions("es")

		assert.Contains(t, result, "Template de PR")
		assert.Contains(t, result, "DEBES seguir")
	})
}

func TestGetTechnicalAnalysisInstruction(t *testing.T) {
	t.Run("English includes key analysis points", func(t *testing.T) {
		result := GetTechnicalAnalysisInstruction("en")

		assert.Contains(t, result, "best practices")
		assert.Contains(t, result, "performance")
		assert.Contains(t, result, "maintainability")
		assert.Contains(t, result, "security")
	})

	t.Run("Spanish is in Spanish", func(t *testing.T) {
		result := GetTechnicalAnalysisInstruction("es")

		assert.Contains(t, result, "buenas prácticas")
		assert.Contains(t, result, "rendimiento")
		assert.Contains(t, result, "mantenibilidad")
		assert.Contains(t, result, "seguridad")
	})
}

func TestGetNoIssueReferenceInstruction(t *testing.T) {
	t.Run("English instruction is clear", func(t *testing.T) {
		result := GetNoIssueReferenceInstruction("en")

		assert.Contains(t, result, "Do not include issue references")
	})

	t.Run("Spanish instruction", func(t *testing.T) {
		result := GetNoIssueReferenceInstruction("es")

		assert.Contains(t, result, "No incluyas referencias de issues")
	})
}

func TestGetReleaseNotesSectionHeaders(t *testing.T) {
	t.Run("English headers are complete", func(t *testing.T) {
		headers := GetReleaseNotesSectionHeaders("en")

		assert.Equal(t, "BREAKING CHANGES:", headers["breaking"])
		assert.Equal(t, "NEW FEATURES:", headers["features"])
		assert.Equal(t, "BUG FIXES:", headers["fixes"])
		assert.Equal(t, "IMPROVEMENTS:", headers["improvements"])
		assert.Equal(t, "CONTRIBUTORS", headers["contributors"])
		assert.Len(t, headers, 9)
	})

	t.Run("Spanish headers are in Spanish", func(t *testing.T) {
		headers := GetReleaseNotesSectionHeaders("es")

		assert.Equal(t, "CAMBIOS QUE ROMPEN:", headers["breaking"])
		assert.Equal(t, "NUEVAS CARACTERÍSTICAS:", headers["features"])
		assert.Equal(t, "CORRECCIONES DE BUGS:", headers["fixes"])
		assert.Equal(t, "MEJORAS:", headers["improvements"])
		assert.Equal(t, "CONTRIBUIDORES", headers["contributors"])
		assert.Len(t, headers, 9)
	})

	t.Run("Unknown language defaults to English", func(t *testing.T) {
		headers := GetReleaseNotesSectionHeaders("fr")
		englishHeaders := GetReleaseNotesSectionHeaders("en")

		assert.Equal(t, englishHeaders, headers)
	})
}

func TestPromptRenderingWorkflow(t *testing.T) {
	t.Run("Complete commit prompt with issue reference", func(t *testing.T) {
		template := GetCommitPromptTemplate("en", true)

		issueInstructions := GetIssueReferenceInstructions("en")
		issueData := struct{ IssueNumber int }{IssueNumber: 42}
		renderedInstructions, err := RenderPrompt("issue_inst", issueInstructions, issueData)
		require.NoError(t, err)

		data := PromptData{
			Count:        3,
			Files:        "auth.go, middleware.go",
			Diff:         "diff --git a/auth.go...",
			Ticket:       "AUTH-42",
			History:      "feat: add login endpoint",
			Instructions: renderedInstructions,
			IssueNumber:  42,
		}

		result, err := RenderPrompt("full_commit", template, data)

		require.NoError(t, err)
		assert.Contains(t, result, "3 commit message suggestions")
		assert.Contains(t, result, "auth.go, middleware.go")
		assert.Contains(t, result, "AUTH-42")
		assert.Contains(t, result, "#42")
		assert.Contains(t, result, "Conventional Commits")
	})

	t.Run("Complete PR prompt with issues and template", func(t *testing.T) {
		issues := []models.Issue{
			{Number: 10, Title: "Add feature", Description: "Feature description"},
			{Number: 11, Title: "Fix bug", Description: "Bug description"},
		}
		formattedIssues := FormatIssuesForPrompt(issues, "en")

		prTemplate := &models.IssueTemplate{
			Name:        "pr_template",
			BodyContent: "## Changes\n## Testing",
		}
		formattedTemplate := FormatTemplateForPrompt(prTemplate, "en", "pr")

		issueContext := GetPRIssueContextInstructions("en")

		prContent := formattedTemplate + "\n\n" + issueContext + "\n\n" + formattedIssues

		template := GetPRPromptTemplate("en")
		data := PromptData{PRContent: prContent}
		result, err := RenderPrompt("full_pr", template, data)

		require.NoError(t, err)
		assert.Contains(t, result, "Issue #10: Add feature")
		assert.Contains(t, result, "Issue #11: Fix bug")
		assert.Contains(t, result, "## Changes")
		assert.Contains(t, result, "Closes #N")
	})
}
