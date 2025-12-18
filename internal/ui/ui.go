package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

var (
	// Colores para diferentes tipos de mensajes
	Success = color.New(color.FgGreen, color.Bold)
	Error   = color.New(color.FgRed, color.Bold)
	Warning = color.New(color.FgYellow, color.Bold)
	Info    = color.New(color.FgCyan, color.Bold)
	Accent  = color.New(color.FgMagenta, color.Bold)
	Dim     = color.New(color.FgHiBlack)

	// Emojis con colores
	SuccessEmoji = Success.Sprint("✓")
	ErrorEmoji   = Error.Sprint("✗")
	WarningEmoji = Warning.Sprint("⚠")
	InfoEmoji    = Info.Sprint("*")
	RocketEmoji  = Accent.Sprint("🚀")
	StatsEmoji   = Accent.Sprint("📊")
)

// SmartSpinner es un spinner con capacidades mejoradas
type SmartSpinner struct {
	spinner *spinner.Spinner
}

// NewSmartSpinner crea un nuevo spinner con mensaje inicial
func NewSmartSpinner(initialMessage string) *SmartSpinner {
	s := spinner.New(
		spinner.CharSets[14],
		100*time.Millisecond,
		spinner.WithColor("cyan"),
		spinner.WithSuffix(" "+initialMessage),
	)
	return &SmartSpinner{spinner: s}
}

func (s *SmartSpinner) Start() {
	s.spinner.Start()
}

func (s *SmartSpinner) Stop() {
	s.spinner.Stop()
}

func (s *SmartSpinner) UpdateMessage(msg string) {
	s.spinner.Suffix = " " + msg
}

func (s *SmartSpinner) Success(msg string) {
	s.spinner.Stop()
	PrintSuccess(msg)
}

func (s *SmartSpinner) Error(msg string) {
	s.spinner.Stop()
	PrintError(msg)
}

func (s *SmartSpinner) Warning(msg string) {
	s.spinner.Stop()
	PrintWarning(msg)
}

func (s *SmartSpinner) Log(msg string) {
	s.spinner.Stop()
	fmt.Println(msg)
	s.spinner.Start()
}

// SpinnerBuilder permite construir spinners con configuración flexible
type SpinnerBuilder struct {
	message string
	charset int
	color   string
	speed   time.Duration
}

// NewSpinner crea un nuevo builder para spinners
func NewSpinner() *SpinnerBuilder {
	return &SpinnerBuilder{
		charset: 14,
		color:   "cyan",
		speed:   100 * time.Millisecond,
	}
}

// WithMessage establece el mensaje del spinner
func (b *SpinnerBuilder) WithMessage(msg string) *SpinnerBuilder {
	b.message = msg
	return b
}

// WithColor establece el color del spinner
func (b *SpinnerBuilder) WithColor(color string) *SpinnerBuilder {
	b.color = color
	return b
}

// WithSpeed establece la velocidad del spinner
func (b *SpinnerBuilder) WithSpeed(speed time.Duration) *SpinnerBuilder {
	b.speed = speed
	return b
}

// WithCharset establece el charset del spinner
func (b *SpinnerBuilder) WithCharset(charset int) *SpinnerBuilder {
	b.charset = charset
	return b
}

// Build construye el SmartSpinner con la configuración especificada
func (b *SpinnerBuilder) Build() *SmartSpinner {
	s := spinner.New(
		spinner.CharSets[b.charset],
		b.speed,
		spinner.WithColor(b.color),
		spinner.WithSuffix(" "+b.message),
	)
	return &SmartSpinner{spinner: s}
}

func PrintSuccess(msg string) {
	fmt.Printf("%s %s\n", SuccessEmoji, Success.Sprint(msg))
}

func PrintError(msg string) {
	fmt.Printf("%s %s\n", ErrorEmoji, Error.Sprint(msg))
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
	PrintError(errMsg)
	fmt.Printf("\n%s %s\n", Info.Sprint("💡"), suggestion)
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
		return fmt.Errorf("error al obtener diff: %w", err)
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

// ShowDiffStats muestra estadísticas de cambios (como git diff --stat)
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

// EditCommitMessage abre un editor para que el usuario edite el mensaje
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

// FileChange representa un archivo modificado con sus estadísticas
type FileChange struct {
	Path      string
	Additions int
	Deletions int
}

// ShowFilesTree muestra los archivos modificados en formato árbol
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

// getFileStats obtiene las estadísticas de cambios para cada archivo
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

// treeNode representa un nodo en el árbol de archivos
type treeNode struct {
	name     string
	isFile   bool
	change   *FileChange
	children map[string]*treeNode
}

// buildFileTree construye un árbol de directorios
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

// printTree imprime el árbol recursivamente
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

// sortFileTree ordena las claves: directorios primero, luego archivos
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

// parseInt convierte string a int, retorna 0 si falla
func parseInt(s string) int {
	var result int
	_, _ = fmt.Sscanf(s, "%d", &result)
	return result
}
