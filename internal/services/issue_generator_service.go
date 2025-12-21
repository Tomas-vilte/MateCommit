package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	domainErrors "github.com/Tomas-vilte/MateCommit/internal/domain/errors"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
)

// issueGitService defines only the methods needed by IssueGeneratorService.
type issueGitService interface {
	GetDiff(ctx context.Context) (string, error)
	GetChangedFiles(ctx context.Context) ([]string, error)
}

// issueTemplateService is a minimal interface for testing purposes
type issueTemplateService interface {
	GetTemplateByName(name string) (*models.IssueTemplate, error)
	MergeWithGeneratedContent(template *models.IssueTemplate, generated *models.IssueGenerationResult) *models.IssueGenerationResult
}

type IssueGeneratorService struct {
	git             issueGitService
	ai              ports.IssueContentGenerator
	vcsClient       ports.VCSClient
	templateService issueTemplateService
	config          *config.Config
}

type IssueGeneratorOption func(*IssueGeneratorService)

func WithIssueVCSClient(vcs ports.VCSClient) IssueGeneratorOption {
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
	ai ports.IssueContentGenerator,
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
func (s *IssueGeneratorService) GenerateFromDiff(ctx context.Context, hint string, skipLabels bool) (*models.IssueGenerationResult, error) {
	if s.ai == nil {
		return nil, domainErrors.ErrAPIKeyMissing
	}

	diff, err := s.git.GetDiff(ctx)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeGit, "failed to get diff", err)
	}

	if diff == "" {
		return nil, domainErrors.ErrNoChanges
	}

	changedFiles, err := s.git.GetChangedFiles(ctx)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeGit, "failed to get changed files", err)
	}

	request := models.IssueGenerationRequest{
		Diff:         diff,
		ChangedFiles: changedFiles,
		Hint:         hint,
		Language:     s.config.Language,
	}

	result, err := s.ai.GenerateIssueContent(ctx, request)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeAI, "failed to generate issue content", err)
	}

	if !skipLabels {
		smartLabels := s.inferSmartLabels(diff, changedFiles)
		result.Labels = s.mergeLabels(result.Labels, smartLabels)
	}

	return result, nil
}

// GenerateFromDescription generates issue content based on a manual description.
// Useful when the user wants to create an issue without having local changes.
func (s *IssueGeneratorService) GenerateFromDescription(ctx context.Context, description string, skipLabels bool) (*models.IssueGenerationResult, error) {
	if s.ai == nil {
		return nil, domainErrors.ErrAPIKeyMissing
	}

	if description == "" {
		return nil, domainErrors.NewAppError(domainErrors.TypeConfiguration, "description is required", nil)
	}

	request := models.IssueGenerationRequest{
		Description: description,
	}
	if s.config != nil {
		request.Language = s.config.Language
	}

	result, err := s.ai.GenerateIssueContent(ctx, request)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeAI, "failed to generate issue content", err)
	}

	if skipLabels {
		result.Labels = []string{}
	}

	return result, nil
}

func (s *IssueGeneratorService) GenerateFromPR(ctx context.Context, prNumber int, hint string, skipLabels bool) (*models.IssueGenerationResult, error) {
	if s.ai == nil {
		return nil, domainErrors.ErrAPIKeyMissing
	}

	if s.vcsClient == nil {
		return nil, domainErrors.ErrConfigMissing
	}

	prData, err := s.vcsClient.GetPR(ctx, prNumber)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeVCS, "failed to get PR", err)
	}

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

	request := models.IssueGenerationRequest{
		Description:  contextBuilder.String(),
		Diff:         prData.Diff,
		ChangedFiles: changedFiles,
		Hint:         hint,
		Language:     s.config.Language,
	}

	result, err := s.ai.GenerateIssueContent(ctx, request)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeAI, "failed to generate issue content", err)
	}

	result.Description = fmt.Sprintf("%s\n\n---\n*Related PR: #%d*", result.Description, prNumber)

	if !skipLabels {
		smartLabels := s.inferSmartLabels(prData.Diff, changedFiles)
		result.Labels = s.mergeLabels(result.Labels, smartLabels)
	}
	return result, nil
}

func (s *IssueGeneratorService) GenerateWithTemplate(ctx context.Context, templateName string, hint string, fromDiff bool, description string, skipLabels bool) (*models.IssueGenerationResult, error) {
	template, err := s.templateService.GetTemplateByName(templateName)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeConfiguration, fmt.Sprintf("failed to load template: %s", templateName), err)
	}

	var baseResult *models.IssueGenerationResult
	if fromDiff {
		baseResult, err = s.GenerateFromDiff(ctx, hint, skipLabels)
	} else if description != "" {
		baseResult, err = s.GenerateFromDescription(ctx, description, skipLabels)
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

	fixPattern := regexp.MustCompile(`(?i)(fix|bug|resolve|close)`)
	featPattern := regexp.MustCompile(`(?i)(feat|feature|add|implement)`)
	refactorPattern := regexp.MustCompile(`(?i)(refactor|restructure|reorganize)`)

	if fixPattern.MatchString(diff) {
		analysis.Keywords["fix"] = true
	}
	if featPattern.MatchString(diff) {
		analysis.Keywords["feat"] = true
	}
	if refactorPattern.MatchString(diff) {
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
