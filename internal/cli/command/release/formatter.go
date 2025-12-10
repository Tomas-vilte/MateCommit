package release

import (
	"fmt"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
)

// FormatReleaseMarkdown genera el markdown completo de una release con todas las secciones
func FormatReleaseMarkdown(release *models.Release, notes *models.ReleaseNotes, trans *i18n.Translations) string {
	content := fmt.Sprintf("# %s\n\n", notes.Title)

	content += "![Version](https://img.shields.io/badge/version-" + release.Version + "-blue)\n"
	content += "![Status](https://img.shields.io/badge/status-released-success)\n\n"

	content += fmt.Sprintf("**%s:** %s\n", trans.GetMessage("release.md_version", 0, nil), release.Version)
	content += fmt.Sprintf("**%s:** %s\n\n", trans.GetMessage("release.md_previous", 0, nil), release.PreviousVersion)

	content += fmt.Sprintf("## %s\n\n%s\n\n", trans.GetMessage("release.md_summary", 0, nil), notes.Summary)

	if len(notes.Highlights) > 0 {
		content += fmt.Sprintf("## %s\n\n", trans.GetMessage("release.md_highlights", 0, nil))
		for _, h := range notes.Highlights {
			content += fmt.Sprintf("- %s\n", h)
		}
		content += "\n"
	}

	if notes.QuickStart != "" {
		content += fmt.Sprintf("## 🚀 %s\n\n", trans.GetMessage("release.quick_start_title", 0, nil))
		content += notes.QuickStart + "\n\n"
	}

	if len(notes.Examples) > 0 {
		content += fmt.Sprintf("## 💡 %s\n\n", trans.GetMessage("release.examples_title", 0, nil))
		for _, example := range notes.Examples {
			if example.Title != "" {
				content += fmt.Sprintf("### %s\n\n", example.Title)
			}
			if example.Description != "" {
				content += fmt.Sprintf("%s\n\n", example.Description)
			}
			if example.Code != "" {
				lang := example.Language
				if lang == "" {
					lang = "bash"
				}
				content += fmt.Sprintf("```%s\n%s\n```\n\n", lang, example.Code)
			}
		}
	}

	if len(notes.BreakingChanges) > 0 {
		content += fmt.Sprintf("## ⚠️ %s\n\n", trans.GetMessage("release.breaking_changes_title", 0, nil))
		for _, bc := range notes.BreakingChanges {
			content += fmt.Sprintf("- %s\n", bc)
		}
		content += "\n"
	} else {
		content += fmt.Sprintf("## ⚠️ %s\n\n%s 🎉\n\n",
			trans.GetMessage("release.breaking_changes_title", 0, nil),
			trans.GetMessage("release.no_breaking_changes", 0, nil))
	}

	if len(notes.Comparisons) > 0 {
		content += fmt.Sprintf("## 📊 %s\n\n", trans.GetMessage("release.comparison_title", 0, nil))
		content += "| Feature | Before | After |\n"
		content += "|---------|--------|-------|\n"
		for _, comp := range notes.Comparisons {
			content += fmt.Sprintf("| %s | %s | %s |\n", comp.Feature, comp.Before, comp.After)
		}
		content += "\n"
	}

	if notes.Changelog != "" {
		content += notes.Changelog + "\n"
	}

	if len(notes.Links) > 0 {
		validLinks := make(map[string]string)
		for key, value := range notes.Links {
			if value != "" && value != "N/A" && key != "" && key != "N/A" {
				validLinks[key] = value
			}
		}

		if len(validLinks) > 0 {
			content += fmt.Sprintf("## 📚 %s\n\n", trans.GetMessage("release.resources_title", 0, nil))
			for key, value := range validLinks {
				content += fmt.Sprintf("- [%s](%s)\n", key, value)
			}
			content += "\n"
		}
	}

	return content
}
