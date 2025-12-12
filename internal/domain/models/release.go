package models

import "time"

type (
	// Release representa un release con toda la info
	Release struct {
		Version         string
		PreviousVersion string
		Title           string
		Summary         string
		Date            time.Time
		Features        []ReleaseItem
		BugFixes        []ReleaseItem
		Breaking        []ReleaseItem
		Documentation   []ReleaseItem
		Improvements    []ReleaseItem
		Other           []ReleaseItem
		AllCommits      []Commit
		VersionBump     VersionBump
		ClosedIssues    []Issue
		MergedPRs       []PullRequest
		Contributors    []string
		NewContributors []string
		Dependencies    []DependencyChange
		FileStats       FileStatistics
	}

	PullRequest struct {
		Number      int
		Title       string
		Description string
		Author      string
		Labels      []string
		URL         string
	}

	DependencyChange struct {
		Name       string
		OldVersion string
		NewVersion string
		Type       string
	}

	FileStatistics struct {
		FilesChanged int
		Insertions   int
		Deletions    int
		TopFiles     []FileChange
	}

	FileChange struct {
		Path      string
		Additions int
		Deletions int
	}

	// ReleaseItem representa un item en el changelog
	ReleaseItem struct {
		Type        string // feat, fix, docs, etc
		Scope       string
		Description string
		Breaking    bool
		CommitHash  string
		PRNumber    string // si tiene pr asociado
	}

	// VersionBump indica el tipo de bump de version
	VersionBump string

	// ReleaseNotes es el resultado generado por la ia
	ReleaseNotes struct {
		Title           string
		Summary         string
		Highlights      []string
		Changelog       string
		Recommended     VersionBump
		QuickStart      string
		Examples        []CodeExample
		BreakingChanges []string
		Comparisons     []Comparison
		Links           map[string]string
	}

	// CodeExample representa un ejemplo de código con descripción
	CodeExample struct {
		Title       string // Título del ejemplo
		Description string // Descripción breve
		Code        string // Código del ejemplo
		Language    string // Lenguaje (bash, go, etc.)
	}

	// Comparison representa una comparación antes/después
	Comparison struct {
		Feature string // Nombre de la feature
		Before  string // Estado anterior
		After   string // Estado nuevo
	}

	VCSRelease struct {
		TagName string
		Name    string
		Body    string
		Draft   bool
		URL     string
	}
)

const (
	MajorBump VersionBump = "major"
	MinorBump VersionBump = "minor"
	PatchBump VersionBump = "patch"
	NoBump    VersionBump = "none"
)
