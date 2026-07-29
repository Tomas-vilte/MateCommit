package release

import (
	"fmt"
	"strings"

	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/models"
)

// FormatReleaseMarkdown generates the full release Markdown with all sections
func FormatReleaseMarkdown(release *models.Release, notes *models.ReleaseNotes, trans *i18n.Translations) string {
	content := fmt.Sprintf("# %s\n\n", notes.Title)
	var md strings.Builder

	content += "![Version](https://img.shields.io/badge/version-" + release.Version + "-blue)\n"
	content += "![Status](https://img.shields.io/badge/status-released-success)\n\n"

	content += fmt.Sprintf("**%s:** %s\n", trans.GetMessage("release.md_version", 0, nil), release.Version)
	content += fmt.Sprintf("**%s:** %s\n\n", trans.GetMessage("release.md_previous", 0, nil), release.PreviousVersion)

	content += fmt.Sprintf("## %s\n\n%s\n\n", trans.GetMessage("release.md_summary", 0, nil), notes.Summary)

	if len(notes.Sections) > 0 {
		for _, section := range notes.Sections {
			content += fmt.Sprintf("## %s\n\n", section.Title)
			for _, item := range section.Items {
				content += fmt.Sprintf("- %s\n", item)
			}
			content += "\n"
		}
	} else if len(notes.Highlights) > 0 {
		content += fmt.Sprintf("## %s\n\n", trans.GetMessage("release.md_highlights", 0, nil))
		for _, h := range notes.Highlights {
			content += fmt.Sprintf("- %s\n", h)
		}
		content += "\n"
	}

	if len(notes.BreakingChanges) > 0 {
		content += fmt.Sprintf("## %s\n\n", trans.GetMessage("release.breaking_changes_title", 0, nil))
		for _, bc := range notes.BreakingChanges {
			content += fmt.Sprintf("- %s\n", bc)
		}
		content += "\n"
	} else {
		content += fmt.Sprintf("## %s\n\n%s\n\n",
			trans.GetMessage("release.breaking_changes_title", 0, nil),
			trans.GetMessage("release.no_breaking_changes", 0, nil))
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
			content += fmt.Sprintf("## %s\n\n", trans.GetMessage("release.resources_title", 0, nil))
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

	if len(release.MergedPRs) > 0 {
		md.WriteString("## ")
		md.WriteString(trans.GetMessage("release.md_pull_requests", 0, nil))
		md.WriteString("\n\n")
		for _, pr := range release.MergedPRs {
			if pr.URL != "" {
				md.WriteString(fmt.Sprintf("- [#%d](%s) %s (by @%s)\n", pr.Number, pr.URL, pr.Title, pr.Author))
			} else {
				md.WriteString(fmt.Sprintf("- #%d %s (by @%s)\n", pr.Number, pr.Title, pr.Author))
			}
		}
		md.WriteString("\n")
	}

	if len(release.Contributors) > 0 {
		md.WriteString("## ")
		md.WriteString(trans.GetMessage("release.md_contributors", 0, nil))
		md.WriteString("\n\n")

		if len(release.NewContributors) > 0 {
			md.WriteString(trans.GetMessage("release.new_contributors", 0, struct{ Count int }{len(release.NewContributors)}))
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
