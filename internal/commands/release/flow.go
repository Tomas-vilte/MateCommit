package release

import (
	"context"
	"fmt"
	"time"

	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/ui"
)

// analyzeNextRelease validates the current branch and computes the next
// release. It's the first step shared by every release subcommand
// (create, generate, preview, publish).
func analyzeNextRelease(ctx context.Context, releaseSvc releaseService, trans *i18n.Translations, start time.Time) (*models.Release, error) {
	log := logger.FromContext(ctx)

	if err := releaseSvc.ValidateMainBranch(ctx); err != nil {
		log.Error("branch validation failed", "error", err)
		return nil, fmt.Errorf("%s", trans.GetMessage("release.error_invalid_branch", 0, struct{ Error string }{err.Error()}))
	}

	release, err := releaseSvc.AnalyzeNextRelease(ctx)
	if err != nil {
		log.Error("failed to analyze next release",
			"error", err,
			"duration_ms", time.Since(start).Milliseconds())
		ui.HandleAppError(err)
		return nil, fmt.Errorf("%s", trans.GetMessage("release.error_analyzing", 0, struct{ Error string }{err.Error()}))
	}

	log.Debug("release analyzed",
		"version", release.Version,
		"previous_version", release.PreviousVersion,
		"version_bump", release.VersionBump)

	return release, nil
}

// generateReleaseNotes enriches the release context (best-effort) and
// generates the release notes. It's the step shared by every release
// subcommand right after the release version is finalized.
func generateReleaseNotes(ctx context.Context, releaseSvc releaseService, trans *i18n.Translations, start time.Time, release *models.Release) (*models.ReleaseNotes, error) {
	log := logger.FromContext(ctx)

	if err := releaseSvc.EnrichReleaseContext(ctx, release); err != nil {
		log.Warn("failed to enrich release context", "error", err)
		fmt.Printf("⚠️  %s\n", trans.GetMessage("release.warning_enrich_context", 0, struct{ Error string }{err.Error()}))
	}

	notes, err := releaseSvc.GenerateReleaseNotes(ctx, release)
	if err != nil {
		log.Error("failed to generate release notes",
			"error", err,
			"duration_ms", time.Since(start).Milliseconds())
		ui.HandleAppError(err)
		return nil, fmt.Errorf("%s", trans.GetMessage("release.error_generating_notes", 0, struct{ Error string }{err.Error()}))
	}

	log.Debug("release notes generated",
		"title", notes.Title,
		"highlights_count", len(notes.Highlights))

	return notes, nil
}
