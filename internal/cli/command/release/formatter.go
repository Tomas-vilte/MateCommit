package release

import (
	"fmt"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
)

// FormatReleaseMarkdown generates the full release markdown with all sections
func FormatReleaseMarkdown(release *models.Release, notes *models.ReleaseNotes, trans *i18n.Translations) string {
	content := fmt.Sprintf("# %s\n\n", notes.Title)
	var md strings.Builder

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
				if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
					content += fmt.Sprintf("- [%s](%s)\n", key, value)
				} else {
					content += fmt.Sprintf("- **%s:** %s\n", key, value)
				}
			}
			content += "\n"
		}
	}

	if len(release.Contributors) > 0 {
		md.WriteString("## ")
		md.WriteString(trans.GetMessage("release.md_contributors", 0, nil))
		md.WriteString("\n\n")

		if len(release.NewContributors) > 0 {
			md.WriteString(trans.GetMessage("release.new_contributors", 0, map[string]interface{}{
				"Count": len(release.NewContributors),
			}))
			md.WriteString(" ")
			for i, contributor := range release.NewContributors {
				md.WriteString(fmt.Sprintf("@%s", contributor))
				if i < len(release.NewContributors)-1 {
					md.WriteString(", ")
				}
			}
			md.WriteString("\n\n")
		}

		md.WriteString(trans.GetMessage("release.all_contributors", 0, nil))
		md.WriteString("\n")
		for _, contributor := range release.Contributors {
			md.WriteString(fmt.Sprintf("- @%s\n", contributor))
		}
		md.WriteString("\n")
	}

	if release.FileStats.FilesChanged > 0 {
		md.WriteString("## ")
		md.WriteString(trans.GetMessage("release.md_stats", 0, nil))
		md.WriteString("\n\n")
		md.WriteString(fmt.Sprintf("- %s: **%d**\n",
			trans.GetMessage("release.files_changed", 0, nil),
			release.FileStats.FilesChanged))
		md.WriteString(fmt.Sprintf("- %s: **+%d**\n",
			trans.GetMessage("release.insertions", 0, nil),
			release.FileStats.Insertions))
		md.WriteString(fmt.Sprintf("- %s: **-%d**\n",
			trans.GetMessage("release.deletions", 0, nil),
			release.FileStats.Deletions))
		md.WriteString("\n")
	}
	return content + md.String()
}
