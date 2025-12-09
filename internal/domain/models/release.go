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
		Title       string
		Summary     string
		Highlights  []string
		Changelog   string
		Recommended VersionBump
	}
)

const (
	MajorBump VersionBump = "major"
	MinorBump VersionBump = "minor"
	PatchBump VersionBump = "patch"
	NoBump    VersionBump = "none"
)
