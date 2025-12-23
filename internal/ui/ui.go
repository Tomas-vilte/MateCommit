package ui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/i18n"
)

var (
	// Colors for different message types
	Success = color.New(color.FgGreen, color.Bold)
	Error   = color.New(color.FgRed, color.Bold)
	Warning = color.New(color.FgYellow, color.Bold)
	Info    = color.New(color.FgCyan, color.Bold)
	Accent  = color.New(color.FgMagenta, color.Bold)
	Dim     = color.New(color.FgHiBlack)

	// Emojis with colors
	SuccessEmoji = Success.Sprint("✓")
	WarningEmoji = Warning.Sprint("⚠")
	InfoEmoji    = Info.Sprint("*")
	RocketEmoji  = Accent.Sprint("🚀")
	StatsEmoji   = Accent.Sprint("📊")
)

var activeSpinner *SmartSpinner
var suspendedSpinner *SmartSpinner

// SmartSpinner is a spinner with enhanced capabilities
type SmartSpinner struct {
	spinner *spinner.Spinner
}

// NewSmartSpinner creates a new spinner with an initial message
func NewSmartSpinner(initialMessage string) *SmartSpinner {
	s := spinner.New(
		spinner.CharSets[14],
		100*time.Millisecond,
		spinner.WithColor("cyan"),
		spinner.WithSuffix(" "+initialMessage),
	)
	return &SmartSpinner{spinner: s}
}

// Start starts the spinner and registers it as the globally active spinner.
func (s *SmartSpinner) Start() {
	activeSpinner = s
	s.spinner.Start()
}

// Stop stops the spinner and clears the active spinner record.
func (s *SmartSpinner) Stop() {
	s.spinner.Stop()
	if activeSpinner == s {
		activeSpinner = nil
	}
	if suspendedSpinner == s {
		suspendedSpinner = nil
	}
}

// StopActiveSpinner stops the currently active spinner in the terminal session.
func StopActiveSpinner() {
	if activeSpinner != nil {
		activeSpinner.Stop()
	}
}

// SuspendActiveSpinner temporarily stops the active spinner without deleting its reference,
// allowing it to be resumed after user interaction.
func SuspendActiveSpinner() {
	if activeSpinner != nil {
		suspendedSpinner = activeSpinner
		activeSpinner.spinner.Stop()
		activeSpinner = nil
	}
}

// ResumeSuspendedSpinner resumes the previously suspended spinner.
func ResumeSuspendedSpinner() {
	if suspendedSpinner != nil {
		activeSpinner = suspendedSpinner
		activeSpinner.spinner.Start()
		suspendedSpinner = nil
	}
}

func (s *SmartSpinner) UpdateMessage(msg string) {
	s.spinner.Suffix = " " + msg
}

func (s *SmartSpinner) Success(msg string) {
	s.Stop()
	PrintSuccess(os.Stdout, msg)
}

func (s *SmartSpinner) Error(msg string) {
	s.Stop()
	PrintError(os.Stdout, msg)
}

func (s *SmartSpinner) Warning(msg string) {
	s.Stop()
	PrintWarning(msg)
}

func (s *SmartSpinner) Log(msg string) {
	s.Stop()
	fmt.Println(msg)
	s.Start()
}

// SpinnerBuilder allows building spinners with flexible configuration
type SpinnerBuilder struct {
	message string
	charset int
	color   string
	speed   time.Duration
}

// NewSpinner creates a new spinner builder
func NewSpinner() *SpinnerBuilder {
	return &SpinnerBuilder{
		charset: 14,
		color:   "cyan",
		speed:   100 * time.Millisecond,
	}
}

// WithMessage sets the spinner message
func (b *SpinnerBuilder) WithMessage(msg string) *SpinnerBuilder {
	b.message = msg
	return b
}

// WithColor sets the spinner color
func (b *SpinnerBuilder) WithColor(color string) *SpinnerBuilder {
	b.color = color
	return b
}

// WithSpeed sets the spinner speed
func (b *SpinnerBuilder) WithSpeed(speed time.Duration) *SpinnerBuilder {
	b.speed = speed
	return b
}

// WithCharset sets the spinner charset
func (b *SpinnerBuilder) WithCharset(charset int) *SpinnerBuilder {
	b.charset = charset
	return b
}

// Build constructs the SmartSpinner with the specified configuration
func (b *SpinnerBuilder) Build() *SmartSpinner {
	s := spinner.New(
		spinner.CharSets[b.charset],
		b.speed,
		spinner.WithColor(b.color),
		spinner.WithSuffix(" "+b.message),
	)
	return &SmartSpinner{spinner: s}
}

func PrintSuccess(w io.Writer, msg string) {
	_, _ = fmt.Fprintf(w, "%s ✓ %s\n", color.GreenString("Success:"), msg)
}

func PrintError(w io.Writer, msg string) {
	_, _ = fmt.Fprintf(w, "%s ❌ %s\n", color.RedString("Error:"), msg)
}

func PrintWarning(msg string) {
	fmt.Printf("%s %s\n", WarningEmoji, Warning.Sprint(msg))
}

func PrintInfo(msg string) {
	fmt.Printf("%s %s\n", InfoEmoji, Info.Sprint(msg))
}

func PrintSectionBanner(title string) {
	separator := color.New(color.FgCyan).Sprint("━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("\n%s\n", separator)
	fmt.Printf("%s %s\n", RocketEmoji, Accent.Sprint(title))
	fmt.Printf("%s\n\n", separator)
}

func PrintDuration(msg string, duration time.Duration) {
	durationStr := Dim.Sprintf("(%s)", duration.Round(10*time.Millisecond))
	fmt.Printf("%s %s %s\n", SuccessEmoji, Success.Sprint(msg), durationStr)
}

func PrintErrorWithSuggestion(errMsg, suggestion string) {
	PrintError(os.Stdout, errMsg)
	if suggestion != "" {
		fmt.Printf("\n%s %s\n", Info.Sprint("💡"), suggestion)
	}
}

// HandleAppError handles an application error and displays it in a friendly way.
func HandleAppError(err error, t *i18n.Translations) {
	if err == nil {
		return
	}

	var appErr *domainErrors.AppError
	ok := errors.As(err, &appErr)
	if !ok {
		PrintError(os.Stdout, err.Error())
		return
	}

	var msg, suggestion string

	switch appErr.Type {
	case domainErrors.TypeConfiguration:
		if appErr.Err != nil && strings.Contains(appErr.Err.Error(), "API key") {
			msg = t.GetMessage("ui_error.ai_api_key_missing_generic", 0, nil)
		} else {
			msg = t.GetMessage("ui_error.config_missing", 0, nil)
		}
		suggestion = t.GetMessage("ui_error.run_config_init", 0, nil)
	case domainErrors.TypeGit:
		if errors.Is(appErr, domainErrors.ErrNoChanges) {
			msg = t.GetMessage("ui_error.no_changes_detected", 0, nil)
			suggestion = t.GetMessage("ui_error.ensure_modified_files", 0, nil)
		} else if errors.Is(appErr, domainErrors.ErrNoDiff) {
			msg = t.GetMessage("ui_error.no_changes_detected", 0, nil)
			suggestion = t.GetMessage("ui_error.ensure_modified_files", 0, nil)
		} else if errors.Is(appErr, domainErrors.ErrGitUserNotConfigured) {
			msg = t.GetMessage("ui_error.git_user_not_configured", 0, nil)
			suggestion = t.GetMessage("ui_error.git_config_user_suggestion", 0, nil)
		} else if errors.Is(appErr, domainErrors.ErrGitEmailNotConfigured) {
			msg = t.GetMessage("ui_error.git_email_not_configured", 0, nil)
			suggestion = t.GetMessage("ui_error.git_config_email_suggestion", 0, nil)
		} else if errors.Is(appErr, domainErrors.ErrNotInGitRepo) {
			msg = t.GetMessage("ui_error.not_in_git_repo", 0, nil)
			suggestion = t.GetMessage("ui_error.git_init_suggestion", 0, nil)
		} else if errors.Is(appErr, domainErrors.ErrInvalidBranch) {
			msg = appErr.Message
			suggestion = "Switch to main or master branch to create releases"
		} else if errors.Is(appErr, domainErrors.ErrTagNotFound) {
			msg = appErr.Message
			if tagCtx, ok := appErr.Context["tag"].(string); ok {
				suggestion = fmt.Sprintf("Tag '%s' not found in repository. Check available tags with: git tag -l", tagCtx)
			}
		} else if errors.Is(appErr, domainErrors.ErrInvalidTagFormat) {
			msg = appErr.Message
			suggestion = "Tags must follow semantic versioning format (vX.Y.Z), e.g., v1.0.0"
		} else {
			msg = appErr.Message
			if appErr.Err != nil {
				suggestion = appErr.Err.Error()
			}
			if appErr.Context != nil {
				if file, ok := appErr.Context["file"].(string); ok {
					suggestion = fmt.Sprintf("File: %s - %s", file, suggestion)
				}
			}
		}
	case domainErrors.TypeAI:
		if errors.Is(appErr, domainErrors.ErrGeminiAPIKeyInvalid) {
			msg = t.GetMessage("ui_error.gemini_api_key_invalid", 0, nil)
			suggestion = t.GetMessage("ui_error.gemini_api_key_suggestion", 0, nil)
		} else if errors.Is(appErr, domainErrors.ErrGeminiQuotaExceeded) {
			msg = t.GetMessage("ui_error.gemini_quota_exceeded", 0, nil)
			suggestion = t.GetMessage("ui_error.gemini_quota_suggestion", 0, nil)
		} else {
			msg = t.GetMessage("ui.error_generating_suggestions", 0, nil)
			if appErr.Err != nil {
				suggestion = appErr.Err.Error()
			}
		}
	case domainErrors.TypeVCS:
		if errors.Is(appErr, domainErrors.ErrGitHubTokenInvalid) {
			msg = t.GetMessage("ui_error.github_token_invalid", 0, nil)
			suggestion = t.GetMessage("ui_error.github_token_suggestion", 0, nil)
		} else if errors.Is(appErr, domainErrors.ErrGitHubInsufficientPerms) {
			msg = t.GetMessage("ui_error.github_insufficient_perms", 0, nil)
			suggestion = t.GetMessage("ui_error.github_perms_suggestion", 0, nil)
		} else if errors.Is(appErr, domainErrors.ErrGitHubRateLimit) {
			msg = t.GetMessage("ui_error.github_rate_limit", 0, nil)
			suggestion = t.GetMessage("ui_error.github_rate_limit_suggestion", 0, nil)
		} else {
			msg = t.GetMessage("ui_error.vcs_error", 0, nil)
			suggestion = t.GetMessage("ui_error.run_config_init", 0, nil)
		}
	case domainErrors.TypeUpdate:
		msg = t.GetMessage("ui_error.update_failed", 0, nil)
	case domainErrors.TypeInternal:
		msg = t.GetMessage("ui_error.internal_error", 0, nil)
		if appErr.Err != nil {
			suggestion = appErr.Err.Error()
		}
	default:
		msg = appErr.Error()
	}

	if msg == "" {
		msg = appErr.Error()
	}

	PrintErrorWithSuggestion(msg, suggestion)
}

func PrintKeyValue(key, value string) {
	keyColored := Dim.Sprint(key + ":")
	valueColored := color.New(color.FgWhite, color.Bold).Sprint(value)
	fmt.Printf("   %s %s\n", keyColored, valueColored)
}

func AskConfirmation(question string) bool {
	fmt.Printf("\n%s (y/n): ", Info.Sprint(question))
	var response string
	_, _ = fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes" || response == "s" || response == "si"
}

func ShowDiff(files []string) error {
	if len(files) == 0 {
		return nil
	}

	cmd := exec.Command("git", append([]string{"diff", "--color=always", "--staged", "--"}, files...)...)
	output, err := cmd.CombinedOutput()

	if err != nil || len(output) == 0 {
		cmd = exec.Command("git", append([]string{"diff", "--color=always", "--"}, files...)...)
		output, err = cmd.CombinedOutput()
	}

	if err != nil {
		return fmt.Errorf("error getting diff: %w", err)
	}

	if len(output) > 0 {
		fmt.Println(string(output))
	}

	return nil
}

func WithSpinner(message string, fn func() error) error {
	s := NewSmartSpinner(message)
	s.Start()

	err := fn()

	if err != nil {
		s.Error(fmt.Sprintf("Error: %v", err))
		return err
	}

	s.Success("Done")
	return nil
}

func WithSpinnerAndDuration(message string, fn func() error) error {
	s := NewSmartSpinner(message)
	s.Start()

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	if err != nil {
		s.Error(fmt.Sprintf("Error: %v", err))
		return err
	}

	s.Stop()
	PrintDuration(message+" completed", duration)
	return nil
}

// ShowDiffStats shows change statistics (like git diff --stat)
func ShowDiffStats(files []string, headerMessage string) error {
	cmd := exec.Command("git", append([]string{"diff", "--stat", "--staged", "--"}, files...)...)
	output, err := cmd.CombinedOutput()

	if err != nil || len(output) == 0 {
		cmd = exec.Command("git", append([]string{"diff", "--stat", "--"}, files...)...)
		output, err = cmd.CombinedOutput()
	}

	if err != nil {
		return err
	}

	if len(output) > 0 {
		fmt.Printf("\n%s %s\n", StatsEmoji, headerMessage)
		fmt.Println(string(output))
	}

	return nil
}

// EditCommitMessage opens an editor for the user to edit the message
func EditCommitMessage(initialMessage string, editorErrorMsg string) (string, error) {
	tmpFile, err := os.CreateTemp("", "commit-msg-*.txt")
	if err != nil {
		return "", fmt.Errorf("%s: %w", editorErrorMsg, err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			return
		}
	}()

	if _, err := tmpFile.WriteString(initialMessage); err != nil {
		return "", fmt.Errorf("%s: %w", editorErrorMsg, err)
	}
	_ = tmpFile.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
		if _, err := exec.LookPath("nano"); err != nil {
			editor = "vi"
		}
	}

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w", editorErrorMsg, err)
	}

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("%s: %w", editorErrorMsg, err)
	}

	editedMessage := strings.TrimSpace(string(content))
	if editedMessage == "" {
		return "", fmt.Errorf("%s", editorErrorMsg)
	}

	return editedMessage, nil
}

// FileChange represents a modified file with its statistics
type FileChange struct {
	Path      string
	Additions int
	Deletions int
}

// ShowFilesTree shows modified files in tree format
func ShowFilesTree(files []string, headerMessage string) error {
	if len(files) == 0 {
		return nil
	}

	fileChanges, err := getFileStats(files)
	if err != nil {
		fmt.Printf("\n%s\n", headerMessage)
		for _, file := range files {
			fmt.Printf("   • %s\n", file)
		}
		return nil
	}

	fmt.Printf("\n%s\n", headerMessage)
	tree := buildFileTree(fileChanges)
	printTree(tree, "", true)

	return nil
}

// getFileStats gets change statistics for each file
func getFileStats(files []string) ([]FileChange, error) {
	cmd := exec.Command("git", append([]string{"diff", "--numstat", "--staged", "--"}, files...)...)
	output, err := cmd.CombinedOutput()

	if err != nil || len(output) == 0 {
		cmd = exec.Command("git", append([]string{"diff", "--numstat", "--"}, files...)...)
		output, err = cmd.CombinedOutput()
	}

	if err != nil {
		return nil, err
	}

	var changes []FileChange
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		additions := 0
		deletions := 0

		if parts[0] != "-" {
			additions = parseInt(parts[0])
		}
		if parts[1] != "-" {
			deletions = parseInt(parts[1])
		}

		changes = append(changes, FileChange{
			Path:      parts[2],
			Additions: additions,
			Deletions: deletions,
		})
	}

	return changes, nil
}

// treeNode represents a node in the file tree
type treeNode struct {
	name     string
	isFile   bool
	change   *FileChange
	children map[string]*treeNode
}

// buildFileTree builds a directory tree
func buildFileTree(changes []FileChange) *treeNode {
	root := &treeNode{
		name:     "",
		children: make(map[string]*treeNode),
	}

	for _, change := range changes {
		parts := strings.Split(change.Path, "/")
		current := root

		for i, part := range parts {
			isFile := i == len(parts)-1

			if current.children[part] == nil {
				current.children[part] = &treeNode{
					name:     part,
					isFile:   isFile,
					children: make(map[string]*treeNode),
				}

				if isFile {
					current.children[part].change = &change
				}
			}
			current = current.children[part]
		}
	}
	return root
}

// printTree prints the tree recursively
func printTree(node *treeNode, prefix string, isLast bool) {
	if node.name != "" {
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		name := node.name
		if !node.isFile {
			name = Info.Sprint(name + "/")
		}

		stats := ""
		if node.isFile && node.change != nil {
			statsColor := color.New(color.FgGreen)
			if node.change.Deletions > node.change.Additions {
				statsColor = color.New(color.FgRed)
			}
			stats = statsColor.Sprintf(" (+%d, -%d)", node.change.Additions, node.change.Deletions)
		}

		fmt.Printf("%s%s%s%s\n", prefix, connector, name, stats)
	}

	childPrefix := prefix
	if node.name != "" {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	var keys []string
	for key := range node.children {
		keys = append(keys, key)
	}

	sortFileTree(keys, node.children)

	for i, key := range keys {
		child := node.children[key]
		isLastChild := i == len(keys)-1
		printTree(child, childPrefix, isLastChild)
	}
}

// sortFileTree sorts the keys: directories first, then files
func sortFileTree(keys []string, nodes map[string]*treeNode) {
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			node1 := nodes[keys[i]]
			node2 := nodes[keys[j]]

			if node1.isFile && !node2.isFile {
				keys[i], keys[j] = keys[j], keys[i]
			} else if node1.isFile == node2.isFile {
				if keys[i] > keys[j] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}
	}
}

// parseInt converts string to int, returns 0 if it fails
func parseInt(s string) int {
	var result int
	_, _ = fmt.Sscanf(s, "%d", &result)
	return result
}
