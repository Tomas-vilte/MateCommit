package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/thomas-vilte/matecommit/internal/ai"
	"github.com/thomas-vilte/matecommit/internal/config"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/regex"
	"github.com/thomas-vilte/matecommit/internal/vcs"
)

// issueGitService defines only the methods needed by IssueGeneratorService.
type issueGitService interface {
	GetDiff(ctx context.Context) (string, error)
	GetChangedFiles(ctx context.Context) ([]string, error)
}

// issueTemplateService is a minimal interface for testing purposes
type issueTemplateService interface {
	GetTemplateByName(ctx context.Context, name string) (*models.IssueTemplate, error)
	MergeWithGeneratedContent(template *models.IssueTemplate, generated *models.IssueGenerationResult) *models.IssueGenerationResult
	ListTemplates(ctx context.Context) ([]models.TemplateMetadata, error)
}

type IssueGeneratorService struct {
	git             issueGitService
	ai              ai.IssueContentGenerator
	vcsClient       vcs.VCSClient
	templateService issueTemplateService
	config          *config.Config
}

type IssueGeneratorOption func(*IssueGeneratorService)

func WithIssueVCSClient(vcs vcs.VCSClient) IssueGeneratorOption {
	return func(s *IssueGeneratorService) {
		s.vcsClient = vcs
	}
}

func WithIssueTemplateService(ts issueTemplateService) IssueGeneratorOption {
	return func(s *IssueGeneratorService) {
		s.templateService = ts
	}
}

func WithIssueConfig(cfg *config.Config) IssueGeneratorOption {
	return func(s *IssueGeneratorService) {
		s.config = cfg
	}
}

func NewIssueGeneratorService(
	gitSvc issueGitService,
	ai ai.IssueContentGenerator,
	opts ...IssueGeneratorOption,
) *IssueGeneratorService {
	s := &IssueGeneratorService{
		git: gitSvc,
		ai:  ai,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// GenerateFromDiff generates issue content based on the current git diff.
// It analyzes local changes (staged and unstaged) to create an appropriate title, description, and labels.
func (s *IssueGeneratorService) GenerateFromDiff(ctx context.Context, hint string, skipLabels bool, autoTemplate bool) (*models.IssueGenerationResult, error) {
	logger.Info(ctx, "generating issue from diff",
		"has_hint", hint != "",
		"skip_labels", skipLabels)

	if s.ai == nil {
		logger.Error(ctx, "AI service not configured", nil)
		return nil, domainErrors.ErrAPIKeyMissing
	}

	diff, err := s.git.GetDiff(ctx)
	if err != nil {
		logger.Error(ctx, "failed to get diff", err)
		return nil, domainErrors.NewAppError(domainErrors.TypeGit, "failed to get diff", err)
	}

	if diff == "" {
		logger.Warn(ctx, "no changes to generate issue from")
		return nil, domainErrors.ErrNoChanges
	}

	changedFiles, err := s.git.GetChangedFiles(ctx)
	if err != nil {
		logger.Error(ctx, "failed to get changed files", err)
		return nil, domainErrors.NewAppError(domainErrors.TypeGit, "failed to get changed files", err)
	}

	logger.Debug(ctx, "git data retrieved",
		"diff_size", len(diff),
		"files_count", len(changedFiles))

	var template *models.IssueTemplate
	if autoTemplate {
		template, _ = s.SelectTemplateWithAI(ctx, "", hint, changedFiles, nil)
	}

	var availableLabels []string
	if s.vcsClient != nil {
		availableLabels, err = s.fetchAvailableLabels(ctx)
		if err != nil {
			logger.Warn(ctx, "failed to fetch repo labels, proceeding without them", err)
		}
	}

	request := models.IssueGenerationRequest{
		Diff:            diff,
		ChangedFiles:    changedFiles,
		Hint:            hint,
		Language:        s.config.Language,
		Template:        template,
		AvailableLabels: availableLabels,
	}

	logger.Debug(ctx, "calling AI for issue generation from diff",
		"has_template", template != nil,
		"available_labels_count", len(availableLabels))

	result, err := s.ai.GenerateIssueContent(ctx, request)
	if err != nil {
		logger.Error(ctx, "failed to generate issue content", err)
		return nil, domainErrors.NewAppError(domainErrors.TypeAI, "failed to generate issue content", err)
	}

	if template != nil {
		result = s.templateService.MergeWithGeneratedContent(template, result)
		logger.Debug(ctx, "merged template with generated content",
			"template_name", template.Name)
	}

	if !skipLabels {
		smartLabels := s.inferSmartLabels(diff, changedFiles)
		result.Labels = s.mergeLabels(result.Labels, smartLabels)
		logger.Debug(ctx, "labels inferred",
			"total_labels", len(result.Labels))
	}

	logger.Info(ctx, "issue generated from diff successfully",
		"title", result.Title)

	return result, nil
}

func (s *IssueGeneratorService) fetchAvailableLabels(ctx context.Context) ([]string, error) {
	if s.vcsClient == nil {
		return nil, nil
	}
	return s.vcsClient.GetRepoLabels(ctx)
}

// GenerateFromDescription generates issue content based on a manual description.
// Useful when the user wants to create an issue without having local changes.
func (s *IssueGeneratorService) GenerateFromDescription(ctx context.Context, description string, skipLabels bool, autoTemplate bool) (*models.IssueGenerationResult, error) {
	logger.Info(ctx, "generating issue from description",
		"description_length", len(description),
		"skip_labels", skipLabels)

	if s.ai == nil {
		logger.Error(ctx, "AI service not configured", nil)
		return nil, domainErrors.ErrAPIKeyMissing
	}

	if description == "" {
		logger.Warn(ctx, "empty description provided")
		return nil, domainErrors.NewAppError(domainErrors.TypeConfiguration, "description is required", nil)
	}

	var template *models.IssueTemplate
	if autoTemplate {
		var err error
		template, err = s.SelectTemplateWithAI(ctx, "", description, nil, nil)
		if err != nil {
			logger.Warn(ctx, "failed to auto-select template via AI", err)
		}
	}

	var availableLabels []string
	if s.vcsClient != nil {
		var err error
		availableLabels, err = s.fetchAvailableLabels(ctx)
		if err != nil {
			logger.Warn(ctx, "failed to fetch repo labels, proceeding without them", err)
		}
	}

	request := models.IssueGenerationRequest{
		Description:     description,
		Template:        template,
		AvailableLabels: availableLabels,
	}
	if s.config != nil {
		request.Language = s.config.Language
	}

	logger.Debug(ctx, "calling AI for issue generation from description",
		"has_template", template != nil,
		"available_labels_count", len(availableLabels))

	result, err := s.ai.GenerateIssueContent(ctx, request)
	if err != nil {
		logger.Error(ctx, "failed to generate issue content", err)
		return nil, domainErrors.NewAppError(domainErrors.TypeAI, "failed to generate issue content", err)
	}

	if template != nil {
		result = s.templateService.MergeWithGeneratedContent(template, result)
		logger.Debug(ctx, "merged template with generated content",
			"template_name", template.Name)
	}

	if skipLabels {
		result.Labels = []string{}
	}

	logger.Info(ctx, "issue generated from description successfully",
		"title", result.Title)

	return result, nil
}

func (s *IssueGeneratorService) GenerateFromPR(ctx context.Context, prNumber int, hint string, skipLabels bool, autoTemplate bool) (*models.IssueGenerationResult, error) {
	logger.Info(ctx, "generating issue from PR",
		"pr_number", prNumber,
		"has_hint", hint != "",
		"skip_labels", skipLabels)

	if s.ai == nil {
		logger.Error(ctx, "AI service not configured", nil)
		return nil, domainErrors.ErrAPIKeyMissing
	}

	if s.vcsClient == nil {
		logger.Error(ctx, "VCS client not configured", nil)
		return nil, domainErrors.ErrConfigMissing
	}

	prData, err := s.vcsClient.GetPR(ctx, prNumber)
	if err != nil {
		logger.Error(ctx, "failed to get PR data", err,
			"pr_number", prNumber)
		return nil, domainErrors.NewAppError(domainErrors.TypeVCS, "failed to get PR", err)
	}

	logger.Debug(ctx, "PR data fetched for issue generation",
		"pr_number", prNumber,
		"diff_size", len(prData.Diff))

	var contextBuilder strings.Builder
	contextBuilder.WriteString(fmt.Sprintf("Pull Request #%d: %s\n\n", prNumber, prData.Title))

	if prData.Description != "" {
		contextBuilder.WriteString("PR Description:\n")
		contextBuilder.WriteString(prData.Description)
		contextBuilder.WriteString("\n\n")
	}

	if len(prData.Commits) > 0 {
		contextBuilder.WriteString("Commits:\n")
		for _, commit := range prData.Commits {
			contextBuilder.WriteString(fmt.Sprintf("- %s\n", commit))
		}
		contextBuilder.WriteString("\n")
	}

	changedFiles := s.extractFilesFromDiff(prData.Diff)

	var template *models.IssueTemplate
	if autoTemplate {
		template, _ = s.SelectTemplateWithAI(ctx, prData.Title, prData.Description, changedFiles, prData.Labels)
	}

	var availableLabels []string
	if s.vcsClient != nil {
		var err error
		availableLabels, err = s.fetchAvailableLabels(ctx)
		if err != nil {
			logger.Warn(ctx, "failed to fetch repo labels, proceeding without them", err)
		}
	}

	request := models.IssueGenerationRequest{
		Description:     contextBuilder.String(),
		Diff:            prData.Diff,
		ChangedFiles:    changedFiles,
		Hint:            hint,
		Language:        s.config.Language,
		Template:        template,
		AvailableLabels: availableLabels,
	}

	logger.Debug(ctx, "calling AI for issue generation from PR",
		"has_template", template != nil)

	result, err := s.ai.GenerateIssueContent(ctx, request)
	if err != nil {
		logger.Error(ctx, "failed to generate issue content from PR", err,
			"pr_number", prNumber)
		return nil, domainErrors.NewAppError(domainErrors.TypeAI, "failed to generate issue content", err)
	}

	if template != nil {
		result = s.templateService.MergeWithGeneratedContent(template, result)
		logger.Debug(ctx, "merged template with generated content",
			"template_name", template.Name)
	}

	result.Description = fmt.Sprintf("%s\n\n---\n*Related PR: #%d*", result.Description, prNumber)

	if !skipLabels {
		smartLabels := s.inferSmartLabels(prData.Diff, changedFiles)
		result.Labels = s.mergeLabels(result.Labels, smartLabels)
		logger.Debug(ctx, "labels inferred from PR",
			"total_labels", len(result.Labels))
	}

	logger.Info(ctx, "issue generated from PR successfully",
		"pr_number", prNumber,
		"title", result.Title)

	return result, nil
}

func (s *IssueGeneratorService) GenerateWithTemplate(ctx context.Context, templateName string, hint string, fromDiff bool, description string, skipLabels bool) (*models.IssueGenerationResult, error) {
	template, err := s.templateService.GetTemplateByName(ctx, templateName)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeConfiguration, fmt.Sprintf("failed to load template: %s", templateName), err)
	}

	var baseResult *models.IssueGenerationResult
	if fromDiff {
		baseResult, err = s.GenerateFromDiff(ctx, hint, skipLabels, false)
	} else if description != "" {
		baseResult, err = s.GenerateFromDescription(ctx, description, skipLabels, false)
	} else {
		return nil, domainErrors.NewAppError(domainErrors.TypeConfiguration, "no input provided", nil)
	}
	if err != nil {
		return nil, err
	}

	result := s.templateService.MergeWithGeneratedContent(template, baseResult)

	if len(template.Assignees) > 0 {
		result.Assignees = s.mergeAssignees(result.Assignees, template.Assignees)
	}

	return result, nil
}

func (s *IssueGeneratorService) SuggestTemplates(ctx context.Context) ([]models.TemplateMetadata, error) {
	if s.templateService == nil {
		return []models.TemplateMetadata{}, nil
	}

	templates, err := s.templateService.ListTemplates(ctx)
	if err != nil {
		return []models.TemplateMetadata{}, nil
	}
	return templates, nil
}

func (s *IssueGeneratorService) mergeAssignees(genAssignees, templateAssignees []string) []string {
	assigneeMap := make(map[string]bool)
	for _, a := range genAssignees {
		if a != "" {
			assigneeMap[a] = true
		}
	}
	for _, a := range templateAssignees {
		if a != "" {
			assigneeMap[a] = true
		}
	}

	result := make([]string, 0, len(assigneeMap))
	for a := range assigneeMap {
		result = append(result, a)
	}
	return result
}

// SelectTemplateWithAI uses AI to analyze the context (diff/description) and select the best template
func (s *IssueGeneratorService) SelectTemplateWithAI(ctx context.Context, title, description string, changedFiles, labels []string) (*models.IssueTemplate, error) {
	if s.ai == nil || s.templateService == nil {
		return nil, nil
	}

	templates, err := s.templateService.ListTemplates(ctx)
	if err != nil || len(templates) == 0 {
		return nil, nil
	}

	var templateListBuilder strings.Builder
	for _, t := range templates {
		templateListBuilder.WriteString(fmt.Sprintf("- %s: %s\n", t.Name, t.About))
	}

	var contextBuilder strings.Builder
	if title != "" {
		contextBuilder.WriteString(fmt.Sprintf("Title: %s\n", title))
	}
	if description != "" {
		contextBuilder.WriteString(fmt.Sprintf("Description: %s\n", description))
	}
	if len(changedFiles) > 0 {
		contextBuilder.WriteString(fmt.Sprintf("Changed files: %s\n", strings.Join(changedFiles, ", ")))
	}
	if len(labels) > 0 {
		contextBuilder.WriteString(fmt.Sprintf("Labels: %s\n", strings.Join(labels, ", ")))
	}

	prompt := fmt.Sprintf(`You are an intelligent assistant helping to select the correct issue template for a software project.
Available templates:
%s
Context (Metadata):
%s
Based on the context, select the SINGLE most appropriate template name from the list above.
Respond ONLY with valid JSON in this exact format:
{
  "title": "Template Name",
  "description": "",
  "labels": []
}
The title field must contain ONLY the template name exactly as it appears in the list (e.g., "Bug Report", "Feature Request").
If no template fits perfectly, choose "Custom Issue" or the most generic one.`, templateListBuilder.String(), contextBuilder.String())

	request := models.IssueGenerationRequest{
		Description: prompt,
		Language:    "en",
	}

	result, err := s.ai.GenerateIssueContent(ctx, request)
	if err != nil {
		logger.Warn(ctx, "failed to auto-select template via AI", err)
		return nil, nil
	}

	selectedName := strings.TrimSpace(result.Title)

	logger.Info(ctx, "AI template selection response",
		"selectedName", selectedName,
		"templates_count", len(templates),
	)

	var bestMatch *models.TemplateMetadata
	for i, t := range templates {
		logger.Debug(ctx, "checking template match",
			"index", i,
			"template_name", t.Name,
			"selected_name", selectedName,
			"exact_match", strings.EqualFold(t.Name, selectedName),
			"contains_match", strings.Contains(strings.ToLower(selectedName), strings.ToLower(t.Name)))
		if strings.EqualFold(t.Name, selectedName) || strings.Contains(strings.ToLower(selectedName), strings.ToLower(t.Name)) {
			bestMatch = &t
			break
		}
	}

	logger.Info(ctx, "template matching result", "found_match", bestMatch != nil)

	if bestMatch != nil {
		logger.Info(ctx, "AI auto-selected template", "template", bestMatch.Name)
		templateName := strings.TrimSuffix(bestMatch.FilePath, ".yml")
		templateName = strings.TrimSuffix(templateName, ".yaml")
		templateName = strings.TrimSuffix(templateName, ".md")
		return s.templateService.GetTemplateByName(ctx, templateName)
	}
	return nil, nil
}

func (s *IssueGeneratorService) extractFilesFromDiff(diff string) []string {
	files := make([]string, 0)
	seen := make(map[string]bool)

	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		var file string

		if strings.HasPrefix(line, "diff --git") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				file = strings.TrimPrefix(parts[2], "a/")
			}
		} else if strings.HasPrefix(line, "--- a/") {
			file = strings.TrimPrefix(line, "--- a/")
		} else if strings.HasPrefix(line, "+++ b/") {
			file = strings.TrimPrefix(line, "+++ b/")
		}

		if file != "" && file != "/dev/null" && !seen[file] {
			seen[file] = true
			files = append(files, file)
		}
	}

	return files
}

// CreateIssue creates a new issue in the repository using the configured VCS client.
// It returns the created issue with its assigned number and URL.
func (s *IssueGeneratorService) CreateIssue(ctx context.Context, result *models.IssueGenerationResult, assignees []string) (*models.Issue, error) {
	return s.vcsClient.CreateIssue(ctx, result.Title, result.Description, result.Labels, assignees)
}

// GetAuthenticatedUser gets the username of the current authenticated user.
func (s *IssueGeneratorService) GetAuthenticatedUser(ctx context.Context) (string, error) {
	return s.vcsClient.GetAuthenticatedUser(ctx)
}

func (s *IssueGeneratorService) InferBranchName(issueNumber int, labels []string) string {
	prefix := "feature"

	labelPriority := map[string]int{
		"fix":      1,
		"refactor": 2,
		"docs":     3,
		"test":     4,
		"infra":    5,
		"feature":  6,
	}
	highestPriority := 999
	for _, label := range labels {
		if priority, exists := labelPriority[label]; exists {
			if priority < highestPriority {
				highestPriority = priority
				prefix = label
			}
		}
	}

	return fmt.Sprintf("%s/issue-%d", prefix, issueNumber)
}

// LinkIssueToPR links an issue to a Pull Request by adding "Closes #issueNumber" to the PR description.
func (s *IssueGeneratorService) LinkIssueToPR(ctx context.Context, prNumber int, issueNumber int) error {
	if s.vcsClient == nil {
		return domainErrors.ErrConfigMissing
	}

	prData, err := s.vcsClient.GetPR(ctx, prNumber)
	if err != nil {
		return domainErrors.NewAppError(domainErrors.TypeVCS, "failed to get PR", err)
	}

	linkText := fmt.Sprintf("\n\nCloses #%d", issueNumber)
	newDescription := prData.Description + linkText

	summary := models.PRSummary{
		Title: prData.Title,
		Body:  newDescription,
	}

	if err := s.vcsClient.UpdatePR(ctx, prNumber, summary); err != nil {
		return domainErrors.NewAppError(domainErrors.TypeVCS, "failed to update PR", err)
	}

	return nil
}

func (s *IssueGeneratorService) inferSmartLabels(diff string, changedFiles []string) []string {
	analysis := s.analyzeDiff(diff, changedFiles)
	labels := make([]string, 0)

	if analysis.HasTestFiles {
		labels = append(labels, "test")
	}
	if analysis.HasDocFiles {
		labels = append(labels, "docs")
	}
	if analysis.HasConfigFiles || strings.Contains(diff, "Dockerfile") || strings.Contains(diff, ".github") {
		labels = append(labels, "infra")
	}

	return labels
}

func (s *IssueGeneratorService) analyzeDiff(diff string, changedFiles []string) models.DiffAnalysis {
	analysis := models.DiffAnalysis{
		Keywords: make(map[string]bool),
	}

	diffLower := strings.ToLower(diff)

	for _, file := range changedFiles {
		fileLower := strings.ToLower(file)
		if strings.HasSuffix(fileLower, ".go") {
			analysis.HasGoFiles = true
		}
		if strings.Contains(fileLower, "test") || strings.HasSuffix(fileLower, "_test.go") {
			analysis.HasTestFiles = true
		}
		if strings.HasSuffix(fileLower, ".md") || strings.Contains(fileLower, "doc") {
			analysis.HasDocFiles = true
		}
		if strings.Contains(fileLower, "config") || strings.HasSuffix(fileLower, ".yaml") || strings.HasSuffix(fileLower, ".yml") || strings.HasSuffix(fileLower, ".json") {
			analysis.HasConfigFiles = true
		}
		if strings.HasSuffix(fileLower, ".css") || strings.HasSuffix(fileLower, ".html") || strings.HasSuffix(fileLower, ".jsx") || strings.HasSuffix(fileLower, ".tsx") {
			analysis.HasUIFiles = true
		}
	}

	keywords := []string{"fix", "bug", "feat", "feature", "add", "refactor", "test", "doc"}
	for _, kw := range keywords {
		if strings.Contains(diffLower, kw) {
			analysis.Keywords[kw] = true
		}
	}

	if regex.FixKeywords.MatchString(diff) {
		analysis.Keywords["fix"] = true
	}
	if regex.FeatKeywords.MatchString(diff) {
		analysis.Keywords["feat"] = true
	}
	if regex.RefactorKeywords.MatchString(diff) {
		analysis.Keywords["refactor"] = true
	}

	return analysis
}

func (s *IssueGeneratorService) mergeLabels(aiLabels, smartLabels []string) []string {
	labelMap := make(map[string]bool)

	for _, label := range aiLabels {
		if label != "" {
			labelMap[label] = true
		}
	}

	for _, label := range smartLabels {
		if label != "" {
			labelMap[label] = true
		}
	}

	if len(labelMap) == 0 {
		return []string{"feature"}
	}

	result := make([]string, 0, len(labelMap))
	for label := range labelMap {
		result = append(result, label)
	}

	return result
}
