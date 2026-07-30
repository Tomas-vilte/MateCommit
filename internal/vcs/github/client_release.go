package github

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/go-github/v80/github"
	"github.com/thomas-vilte/matecommit/internal/builder"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
)

func (ghc *GitHubClient) CreateRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool, buildBinaries bool, progressCh chan<- models.BuildProgress) error {
	body := notes.Changelog
	if body == "" {
		body = fmt.Sprintf("%s\n\n", notes.Summary)
		if len(notes.Highlights) > 0 {
			body += "## Highlights\n\n"
			for _, h := range notes.Highlights {
				body += fmt.Sprintf("- %s\n", h)
			}
		}
	}

	releaseRequest := &github.RepositoryRelease{
		TagName:    github.Ptr(release.Version),
		Name:       github.Ptr(notes.Title),
		Body:       github.Ptr(body),
		Draft:      github.Ptr(draft),
		Prerelease: github.Ptr(false),
		MakeLatest: github.Ptr("true"),
	}

	createdRelease, resp, err := ghc.releaseService.CreateRelease(ctx, ghc.owner, ghc.repo, releaseRequest)
	if err != nil {
		if resp != nil {
			if resp.StatusCode == http.StatusUnauthorized {
				return domainErrors.ErrGitHubTokenInvalid.
					WithContext("operation", "create release").
					WithContext("version", release.Version)
			}
			if resp.StatusCode == http.StatusUnprocessableEntity {
				return domainErrors.ErrCreateRelease.
					WithContext("version", release.Version).
					WithContext("reason", "release already exists")
			}
			if resp.StatusCode == http.StatusNotFound {
				return domainErrors.ErrRepositoryNotFound.
					WithContext("operation", "create release").
					WithContext("version", release.Version).
					WithContext("repo", fmt.Sprintf("%s/%s", ghc.owner, ghc.repo))
			}
			if resp.StatusCode == http.StatusForbidden {
				return domainErrors.ErrGitHubInsufficientPerms.
					WithContext("operation", "create release").
					WithContext("version", release.Version)
			}
		}
		return domainErrors.ErrCreateRelease.WithError(err).WithContext("version", release.Version)
	}

	if buildBinaries {
		if err := ghc.uploadBinaries(ctx, createdRelease.GetID(), release.Version, progressCh); err != nil {
			return fmt.Errorf("failed to upload binaries: %w", err)
		}
	}

	return nil
}

func (ghc *GitHubClient) GetRelease(ctx context.Context, version string) (*models.VCSRelease, error) {
	release, resp, err := ghc.releaseService.GetReleaseByTag(ctx, ghc.owner, ghc.repo, version)
	if err != nil {
		if resp != nil {
			if resp.StatusCode == http.StatusUnauthorized {
				return nil, domainErrors.ErrGitHubTokenInvalid.
					WithContext("operation", "get release").
					WithContext("version", version)
			}
			if resp.StatusCode == http.StatusNotFound {
				return nil, domainErrors.ErrRepositoryNotFound.
					WithContext("operation", "get release").
					WithContext("version", version).
					WithContext("repo", fmt.Sprintf("%s/%s", ghc.owner, ghc.repo))
			}
			return nil, domainErrors.ErrGetRelease.
				WithContext("version", version).
				WithContext("status_code", resp.StatusCode)
		}
		return nil, domainErrors.ErrGetRelease.WithError(err).WithContext("version", version)
	}

	return &models.VCSRelease{
		TagName: release.GetTagName(),
		Name:    release.GetName(),
		Body:    release.GetBody(),
		Draft:   release.GetDraft(),
		URL:     release.GetHTMLURL(),
	}, nil
}

func (ghc *GitHubClient) UpdateRelease(ctx context.Context, version, body string) error {
	release, resp, err := ghc.releaseService.GetReleaseByTag(ctx, ghc.owner, ghc.repo, version)
	if err != nil {
		if resp != nil {
			if resp.StatusCode == http.StatusUnauthorized {
				return domainErrors.ErrGitHubTokenInvalid.
					WithContext("operation", "update release").
					WithContext("version", version)
			}
			if resp.StatusCode == http.StatusNotFound {
				return domainErrors.ErrRepositoryNotFound.
					WithContext("operation", "update release").
					WithContext("version", version).
					WithContext("repo", fmt.Sprintf("%s/%s", ghc.owner, ghc.repo))
			}
			return domainErrors.ErrUpdateRelease.
				WithContext("version", version).
				WithContext("status_code", resp.StatusCode)
		}
		return domainErrors.ErrUpdateRelease.WithError(err).WithContext("version", version)
	}

	releaseUpdate := &github.RepositoryRelease{
		Body: github.Ptr(body),
	}

	_, _, err = ghc.releaseService.EditRelease(ctx, ghc.owner, ghc.repo, release.GetID(), releaseUpdate)
	if err != nil {
		return domainErrors.ErrUpdateRelease.WithError(err).WithContext("version", version)
	}
	return nil
}

func (ghc *GitHubClient) GetMergedPRsBetweenTags(ctx context.Context, previousTag, _ string) ([]models.PullRequest, error) {
	prevRelease, _, err := ghc.releaseService.GetReleaseByTag(ctx, ghc.owner, ghc.repo, previousTag)
	if err != nil {
		return nil, err
	}

	opts := &github.PullRequestListOptions{
		State:     "closed",
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
	var allPRs []models.PullRequest
	for {
		prs, resp, err := ghc.prService.List(ctx, ghc.owner, ghc.repo, opts)
		if err != nil {
			return nil, err
		}

		for _, pr := range prs {
			// The list endpoint never populates the "merged" boolean field
			// (only the single-PR endpoint does) — merged_at is the only
			// reliable signal here, and a zero value means "not merged".
			if !pr.GetMergedAt().IsZero() && pr.GetMergedAt().After(prevRelease.GetCreatedAt().Time) {
				labels := make([]string, 0, len(pr.Labels))
				for _, label := range pr.Labels {
					labels = append(labels, label.GetName())
				}

				allPRs = append(allPRs, models.PullRequest{
					Number:      pr.GetNumber(),
					Title:       pr.GetTitle(),
					Description: pr.GetBody(),
					Author:      pr.GetUser().GetLogin(),
					Labels:      labels,
					URL:         pr.GetHTMLURL(),
				})
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return allPRs, nil
}

func (ghc *GitHubClient) GetContributorsBetweenTags(ctx context.Context, previousTag, currentTag string) ([]string, error) {
	comparison, _, err := ghc.repoService.CompareCommits(ctx, ghc.owner, ghc.repo, previousTag, currentTag, &github.ListOptions{
		PerPage: 100,
	})
	if err != nil {
		return nil, err
	}

	contributorsMap := make(map[string]struct{})
	for _, commit := range comparison.Commits {
		if author := commit.GetAuthor(); author != nil {
			contributorsMap[author.GetLogin()] = struct{}{}
		}
	}

	contributors := make([]string, 0, len(contributorsMap))
	for contributor := range contributorsMap {
		contributors = append(contributors, contributor)
	}
	return contributors, nil
}

func (ghc *GitHubClient) GetFileStatsBetweenTags(ctx context.Context, previousTag, currentTag string) (*models.FileStatistics, error) {
	comparison, _, err := ghc.repoService.CompareCommits(ctx, ghc.owner, ghc.repo, previousTag, currentTag, &github.ListOptions{
		PerPage: 100,
	})
	if err != nil {
		return nil, err
	}

	stats := &models.FileStatistics{
		FilesChanged: len(comparison.Files),
		Insertions:   0,
		Deletions:    0,
		TopFiles:     make([]models.FileChange, 0),
	}

	fileChanges := make([]models.FileChange, 0, len(comparison.Files))
	for _, file := range comparison.Files {
		stats.Insertions += file.GetAdditions()
		stats.Deletions += file.GetDeletions()

		fileChanges = append(fileChanges, models.FileChange{
			Path:      file.GetFilename(),
			Additions: file.GetAdditions(),
			Deletions: file.GetDeletions(),
		})
	}

	sort.Slice(fileChanges, func(i, j int) bool {
		totalI := fileChanges[i].Additions + fileChanges[i].Deletions
		totalJ := fileChanges[j].Additions + fileChanges[j].Deletions
		return totalI > totalJ
	})

	if len(fileChanges) > 5 {
		stats.TopFiles = fileChanges[:5]
	} else {
		stats.TopFiles = fileChanges
	}
	return stats, nil
}

func (ghc *GitHubClient) uploadBinaries(ctx context.Context, releaseID int64, version string, progressCh chan<- models.BuildProgress) error {
	tempDir, err := os.MkdirTemp("", "matecommit-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory for build: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			return
		}
	}()

	commit, err := ghc.getCommitSHA(ctx)
	if err != nil {
		commit = "unknown"
	}
	date := time.Now().Format(time.RFC3339)

	log := logger.FromContext(ctx)

	log.Debug("creating binary builder",
		"repo", ghc.repo,
		"main_path", ghc.mainPath,
		"version", version)

	builderBinary := ghc.binaryBuilderFactory.NewBuilder(
		ghc.mainPath,
		ghc.repo,
		builder.WithVersion(version),
		builder.WithCommit(commit),
		builder.WithDate(date),
		builder.WithBuildDir(tempDir),
	)

	log.Info("compiling binaries for release",
		"version", version,
		"build_dir", tempDir)

	archives, err := builderBinary.BuildAndPackageAll(ctx, progressCh)
	if err != nil {
		return fmt.Errorf("failed to build binaries: %w", err)
	}

	log.Info("uploading binaries to release",
		"archives_count", len(archives),
		"release_id", releaseID,
		"version", version)

	if progressCh != nil {
		progressCh <- models.BuildProgress{
			Type:  models.UploadProgressStart,
			Total: len(archives),
		}
	}

	for i, archivePath := range archives {
		archiveName := filepath.Base(archivePath)

		if progressCh != nil {
			progressCh <- models.BuildProgress{
				Type:    models.UploadProgressAsset,
				Asset:   archiveName,
				Current: i + 1,
				Total:   len(archives),
			}
		}

		log.Info("uploading asset",
			"asset", archiveName,
			"progress", fmt.Sprintf("%d/%d", i+1, len(archives)))

		file, err := os.Open(archivePath)
		if err != nil {
			return fmt.Errorf("failed to open archive %s: %w", archivePath, err)
		}

		uploadOpts := &github.UploadOptions{
			Name:  archiveName,
			Label: archiveName,
		}

		_, _, err = ghc.releaseService.UploadReleaseAsset(ctx, ghc.owner, ghc.repo, releaseID, uploadOpts, file)
		_ = file.Close()
		if err != nil {
			return domainErrors.ErrUploadAsset.WithError(err).
				WithContext("asset_path", archivePath).
				WithContext("release_id", releaseID)
		}

		log.Info("asset uploaded successfully",
			"asset", archiveName,
			"progress", fmt.Sprintf("%d/%d", i+1, len(archives)))
	}

	if progressCh != nil {
		progressCh <- models.BuildProgress{
			Type:  models.UploadProgressComplete,
			Total: len(archives),
		}
	}

	return nil
}

func (ghc *GitHubClient) getCommitSHA(ctx context.Context) (string, error) {
	ref, _, err := ghc.repoService.GetCommit(ctx, ghc.owner, ghc.repo, "HEAD", nil)
	if err != nil {
		return "", err
	}
	return ref.GetSHA(), nil
}
