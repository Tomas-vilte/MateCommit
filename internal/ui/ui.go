package ui

import (
	"fmt"
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
	SparkleEmoji = Accent.Sprint("✨")
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
