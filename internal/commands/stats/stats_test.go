package stats

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/services/cost"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStatsCommand(t *testing.T) {
	// Arrange & Act
	cmd := NewStatsCommand()

	// Assert
	assert.NotNil(t, cmd, "NewStatsCommand should return a non-nil instance")
	assert.IsType(t, &StatsCommand{}, cmd, "should return a pointer to StatsCommand")
}

func TestShowDailyStats_NoActivity(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	manager := setupTestManager(t, tempDir, []cost.ActivityRecord{})
	trans := setupTestTranslations(t)
	cmd := NewStatsCommand()

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Act
	err := cmd.showDailyStats(manager, trans)

	_ = w.Close()
	os.Stdout = oldStdout
	_, _ = io.Copy(&buf, r)

	output := buf.String()

	// Assert
	assert.NoError(t, err, "showDailyStats should not return an error with empty data")
	assert.Contains(t, output, "No activity recorded", "should indicate that there is no activity")
	assert.Contains(t, output, "━", "should contain the table separator")
}

func TestShowDailyStats_WithActivity(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	now := time.Now()

	records := []cost.ActivityRecord{
		{
			Timestamp:    now,
			Command:      "suggest",
			Provider:     "gemini",
			Model:        "gemini-2.5-flash",
			TokensInput:  100,
			TokensOutput: 50,
			CostUSD:      0.0015,
			DurationMs:   1500,
			CacheHit:     false,
		},
		{
			Timestamp:    now,
			Command:      "summarize-pr",
			Provider:     "gemini",
			Model:        "gemini-3.0-flash",
			TokensInput:  500,
			TokensOutput: 200,
			CostUSD:      0.0085,
			DurationMs:   2300,
			CacheHit:     false,
		},
		{
			Timestamp:    now.AddDate(0, 0, -1),
			Command:      "suggest",
			Provider:     "gemini",
			Model:        "gemini-2.5-flash",
			TokensInput:  100,
			TokensOutput: 50,
			CostUSD:      0.0010,
			DurationMs:   1000,
			CacheHit:     false,
		},
	}

	manager := setupTestManager(t, tempDir, records)
	trans := setupTestTranslations(t)
	cmd := NewStatsCommand()

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Act
	err := cmd.showDailyStats(manager, trans)

	_ = w.Close()
	os.Stdout = oldStdout
	_, _ = io.Copy(&buf, r)

	outputStr := buf.String()

	// Assert
	assert.NoError(t, err, "showDailyStats should not return an error")
	assert.Contains(t, outputStr, "suggest", "should show the suggest command")
	assert.Contains(t, outputStr, "summarize-pr", "should show the summarize-pr command")
	assert.Contains(t, outputStr, "$0.0015", "should show the cost of the first command")
	assert.Contains(t, outputStr, "$0.0085", "should show the cost of the second command")
	assert.Contains(t, outputStr, "━", "should contain visual separators")

	total, err := manager.GetDailyTotal()
	assert.NoError(t, err)
	assert.Equal(t, 0.0100, total, "the calculated total should be 0.0100")
}

func TestShowDailyStats_WithCacheHits(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	now := time.Now()

	records := []cost.ActivityRecord{
		{
			Timestamp:    now,
			Command:      "suggest",
			Provider:     "gemini",
			Model:        "gemini-2.5-flash",
			TokensInput:  100,
			TokensOutput: 50,
			CostUSD:      0.0000,
			DurationMs:   50,
			CacheHit:     true,
		},
		{
			Timestamp:    now,
			Command:      "suggest",
			Provider:     "gemini",
			Model:        "gemini-2.5-flash",
			TokensInput:  100,
			TokensOutput: 50,
			CostUSD:      0.0015,
			DurationMs:   1500,
			CacheHit:     false,
		},
	}

	manager := setupTestManager(t, tempDir, records)
	trans := setupTestTranslations(t)
	cmd := NewStatsCommand()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Act
	err := cmd.showDailyStats(manager, trans)

	_ = w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	outputStr := string(output)

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, outputStr, "[CACHE]", "should indicate when there is a cache hit")
	assert.Contains(t, outputStr, "$0.0000", "should show zero cost for cache hit")
	assert.Contains(t, outputStr, "$0.0015", "should show the correct total")
}

func TestShowMonthlyStats_NoActivity(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	manager := setupTestManager(t, tempDir, []cost.ActivityRecord{})
	trans := setupTestTranslations(t)
	cmd := NewStatsCommand()

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Act
	err := cmd.showMonthlyStats(manager, trans)

	_ = w.Close()
	os.Stdout = oldStdout
	_, _ = io.Copy(&buf, r)

	output := buf.String()

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, output, "No activity recorded", "should indicate that there is no activity")
	assert.Contains(t, output, "━", "should contain visual separators")
}

func TestShowMonthlyStats_WithActivity(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	now := time.Now()

	records := []cost.ActivityRecord{
		// Day 1
		{
			Timestamp: time.Date(now.Year(), now.Month(), 1, 10, 0, 0, 0, time.Local),
			Command:   "suggest",
			CostUSD:   0.0015,
		},
		{
			Timestamp: time.Date(now.Year(), now.Month(), 1, 15, 0, 0, 0, time.Local),
			Command:   "summarize-pr",
			CostUSD:   0.0025,
		},
		// Day 2
		{
			Timestamp: time.Date(now.Year(), now.Month(), 2, 10, 0, 0, 0, time.Local),
			Command:   "suggest",
			CostUSD:   0.0010,
		},
		{
			Timestamp: now.AddDate(0, -1, 0),
			Command:   "suggest",
			CostUSD:   0.0050,
		},
	}

	manager := setupTestManager(t, tempDir, records)
	trans := setupTestTranslations(t)
	cmd := NewStatsCommand()

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Act
	err := cmd.showMonthlyStats(manager, trans)

	_ = w.Close()
	os.Stdout = oldStdout
	_, _ = io.Copy(&buf, r)

	outputStr := buf.String()

	// Assert
	assert.NoError(t, err)

	day1 := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).Format("2006-01-02")
	day2 := time.Date(now.Year(), now.Month(), 2, 0, 0, 0, 0, time.Local).Format("2006-01-02")

	assert.Contains(t, outputStr, day1, "should show day 1")
	assert.Contains(t, outputStr, day2, "should show day 2")
	assert.Contains(t, outputStr, "$0.0040", "should show the total for day 1 (0.0015 + 0.0025)")
	assert.Contains(t, outputStr, "$0.0010", "should show the total for day 2")
	assert.Contains(t, outputStr, "━", "should contain visual separators")

	total, err := manager.GetMonthlyTotal()
	assert.NoError(t, err)
	assert.Equal(t, 0.0050, total, "the monthly total should be 0.0050")
}

func TestShowMonthlyStats_GroupsByDay(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	now := time.Now()
	sameDay := time.Date(now.Year(), now.Month(), 15, 0, 0, 0, 0, time.Local)

	records := []cost.ActivityRecord{
		{Timestamp: sameDay.Add(1 * time.Hour), Command: "suggest", CostUSD: 0.0010},
		{Timestamp: sameDay.Add(5 * time.Hour), Command: "suggest", CostUSD: 0.0020},
		{Timestamp: sameDay.Add(10 * time.Hour), Command: "summarize-pr", CostUSD: 0.0030},
		{Timestamp: sameDay.Add(20 * time.Hour), Command: "suggest", CostUSD: 0.0005},
	}

	manager := setupTestManager(t, tempDir, records)
	trans := setupTestTranslations(t)
	cmd := NewStatsCommand()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Act
	err := cmd.showMonthlyStats(manager, trans)

	_ = w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	outputStr := string(output)

	// Assert
	assert.NoError(t, err)

	expectedDay := sameDay.Format("2006-01-02")
	assert.Contains(t, outputStr, expectedDay, "should show the grouped day")
	assert.Contains(t, outputStr, "$0.0065", "should sum all costs for the day (0.0010+0.0020+0.0030+0.0005)")
}

func TestShowDailyStats_FormatsTime(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	now := time.Now()
	specificTime := time.Date(now.Year(), now.Month(), now.Day(), 14, 30, 0, 0, time.Local)

	records := []cost.ActivityRecord{
		{
			Timestamp: specificTime,
			Command:   "suggest",
			Provider:  "gemini",
			Model:     "gemini-2.5-flash",
			CostUSD:   0.0015,
			CacheHit:  false,
		},
	}

	manager := setupTestManager(t, tempDir, records)
	trans := setupTestTranslations(t)
	cmd := NewStatsCommand()

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Act
	err := cmd.showDailyStats(manager, trans)

	_ = w.Close()
	os.Stdout = oldStdout
	_, _ = io.Copy(&buf, r)

	// Assert
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "14:30", "should format the time as HH:MM")
	assert.Contains(t, buf.String(), "suggest", "should show the command")
}

func setupTestManager(t *testing.T, tempDir string, records []cost.ActivityRecord) *cost.Manager {
	t.Helper()

	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tempDir)
	t.Cleanup(func() {
		_ = os.Setenv("HOME", originalHome)
	})

	manager, err := cost.NewManager(0)
	require.NoError(t, err, "should not fail to create the test manager")

	for _, record := range records {
		err := manager.SaveActivity(record)
		require.NoError(t, err, "should not fail to save test activity")
	}

	return manager
}

func setupTestTranslations(t *testing.T) *i18n.Translations {
	t.Helper()

	localesPath := filepath.Join("..", "..", "i18n", "locales")
	trans, err := i18n.NewTranslations("en", localesPath)
	require.NoError(t, err, "should not fail to create test translations")

	return trans
}

func TestStatsCommand_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Arrange
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tempDir)
	t.Cleanup(func() {
		_ = os.Setenv("HOME", originalHome)
	})

	trans := setupTestTranslations(t)
	manager, err := cost.NewManager(0)
	require.NoError(t, err)

	now := time.Now()
	testRecords := []cost.ActivityRecord{
		{
			Timestamp:    now,
			Command:      "suggest",
			Provider:     "gemini",
			Model:        "gemini-2.5-flash",
			TokensInput:  100,
			TokensOutput: 50,
			CostUSD:      0.0015,
			DurationMs:   1500,
			CacheHit:     false,
			Hash:         "test-hash-1",
		},
	}

	for _, record := range testRecords {
		err := manager.SaveActivity(record)
		require.NoError(t, err)
	}

	cmd := NewStatsCommand()

	// Act - Daily Stats
	var dailyBuf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	errDaily := cmd.showDailyStats(manager, trans)

	_ = w.Close()
	os.Stdout = oldStdout
	_, _ = io.Copy(&dailyBuf, r)

	// Assert
	assert.NoError(t, errDaily)
	assert.NotEmpty(t, dailyBuf.String(), "should generate output for daily statistics")

	// Act - Monthly Stats
	var monthlyBuf bytes.Buffer
	r2, w2, _ := os.Pipe()
	os.Stdout = w2

	errMonthly := cmd.showMonthlyStats(manager, trans)

	_ = w2.Close()
	os.Stdout = oldStdout
	_, _ = io.Copy(&monthlyBuf, r2)

	// Assert
	assert.NoError(t, errMonthly)
	assert.NotEmpty(t, monthlyBuf.String(), "should generate output for monthly statistics")
}

func TestShowDailyStats_HandlesManagerErrors(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	trans := setupTestTranslations(t)
	cmd := NewStatsCommand()

	manager := setupTestManager(t, tempDir, []cost.ActivityRecord{})

	historyPath := filepath.Join(tempDir, ".matecommit", "history.json")
	err := os.WriteFile(historyPath, []byte("invalid json{{{"), 0644)
	require.NoError(t, err)

	// Act
	err = cmd.showDailyStats(manager, trans)

	// Assert
	assert.Error(t, err, "should return an error when the history is corrupt")
}

func TestShowMonthlyStats_HandlesManagerErrors(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	trans := setupTestTranslations(t)
	cmd := NewStatsCommand()

	manager := setupTestManager(t, tempDir, []cost.ActivityRecord{})

	historyPath := filepath.Join(tempDir, ".matecommit", "history.json")
	err := os.WriteFile(historyPath, []byte("corrupted"), 0644)
	require.NoError(t, err)

	// Act
	err = cmd.showMonthlyStats(manager, trans)

	// Assert
	assert.Error(t, err, "should return an error when the history is corrupt")
}
