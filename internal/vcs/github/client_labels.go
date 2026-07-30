package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v80/github"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
)

var allowedLabels = map[string]struct {
	Color string
	Key   string
}{
	"feature":  {"00FF00", "label.feature"},
	"fix":      {"FF0000", "label.fix"},
	"refactor": {"FFA500", "label.refactor"},
	"docs":     {"0075CA", "label.docs"},
	"infra":    {"808080", "label.infra"},
	"test":     {"8A2BE2", "label.test"},
}

var labelDescriptions = map[string]string{
	"feature":  "New feature",
	"fix":      "Bug fix",
	"refactor": "Code refactor",
	"docs":     "Documentation",
	"infra":    "Infrastructure",
	"test":     "Test",
}

var labelAliases = map[string]string{
	"bug":            "fix",
	"enhancement":    "feature",
	"documentation":  "docs",
	"infrastructure": "infra",
	"testing":        "test",
}

func (ghc *GitHubClient) GetRepoLabels(ctx context.Context) ([]string, error) {
	labels, err := ghc.GetRepoLabelsWithDescriptions(ctx)
	if err != nil {
		return nil, err
	}

	labelNames := make([]string, len(labels))
	for i, label := range labels {
		labelNames[i] = label.Name
	}
	return labelNames, nil
}

func (ghc *GitHubClient) GetRepoLabelsWithDescriptions(ctx context.Context) ([]models.RepoLabel, error) {
	labels, _, err := ghc.issuesService.ListLabels(ctx, ghc.owner, ghc.repo, &github.ListOptions{PerPage: 100})
	if err != nil {
		return nil, fmt.Errorf("failed to list repository labels: %w", err)
	}

	result := make([]models.RepoLabel, len(labels))
	for i, label := range labels {
		result[i] = models.RepoLabel{Name: label.GetName(), Description: label.GetDescription()}
	}
	return result, nil
}

func (ghc *GitHubClient) CreateLabel(ctx context.Context, name, color, description string) error {
	_, _, err := ghc.issuesService.CreateLabel(ctx, ghc.owner, ghc.repo, &github.Label{
		Name:        github.Ptr(name),
		Color:       github.Ptr(color),
		Description: github.Ptr(description),
	})
	return err
}

func (ghc *GitHubClient) labelExists(existingLabels []string, target string) bool {
	for _, l := range existingLabels {
		if strings.EqualFold(l, target) {
			return true
		}
	}
	return false
}

func (ghc *GitHubClient) addLabelsToIssue(ctx context.Context, prNumber int, labels []string) error {
	_, _, err := ghc.issuesService.AddLabelsToIssue(ctx, ghc.owner, ghc.repo, prNumber, labels)
	if err != nil {
		return fmt.Errorf("failed to add labels to PR #%d: %w", prNumber, err)
	}
	return nil
}

func (ghc *GitHubClient) ensureLabelsExist(ctx context.Context, existingLabels []string, requiredLabels []string) error {
	log := logger.FromContext(ctx)

	for _, label := range requiredLabels {
		if !ghc.labelExists(existingLabels, label) {
			meta := allowedLabels[label]

			description := labelDescriptions[label]
			if err := ghc.CreateLabel(ctx, label, meta.Color, description); err != nil {
				if !strings.Contains(err.Error(), "already_exists") && !strings.Contains(err.Error(), "422") {
					return fmt.Errorf("failed to create label '%s': %w", label, err)
				}
				log.Debug("label already exists, skipping creation",
					"label", label,
					"owner", ghc.owner,
					"repo", ghc.repo)
			}
		}
	}
	return nil
}

func (ghc *GitHubClient) validateAndFilterLabels(labels []string) []string {
	var validLabels []string
	seen := make(map[string]bool)

	for _, label := range labels {
		cleaned := strings.ToLower(strings.TrimSpace(label))
		if cleaned == "" {
			continue
		}

		if mapped, ok := labelAliases[cleaned]; ok {
			cleaned = mapped
		}

		if ghc.isAllowedLabel(cleaned) && !seen[cleaned] {
			validLabels = append(validLabels, cleaned)
			seen[cleaned] = true
		}
	}
	return validLabels
}

func (ghc *GitHubClient) isAllowedLabel(label string) bool {
	_, exists := allowedLabels[label]
	return exists
}
