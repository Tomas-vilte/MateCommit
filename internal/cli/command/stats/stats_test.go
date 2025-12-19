package stats

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/services/cost"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStatsCommand(t *testing.T) {
	// Arrange & Act
	cmd := NewStatsCommand()

	// Assert
	assert.NotNil(t, cmd, "NewStatsCommand debería retornar una instancia no nula")
	assert.IsType(t, &StatsCommand{}, cmd, "debería retornar un puntero a StatsCommand")
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
	assert.NoError(t, err, "showDailyStats no debería retornar error con datos vacíos")
	assert.Contains(t, output, "No activity recorded", "debería indicar que no hay actividad")
	assert.Contains(t, output, "━", "debería contener el separador de la tabla")
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
			Timestamp:    now.Add(-1 * time.Hour),
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
	assert.NoError(t, err, "showDailyStats no debería retornar error")
	assert.Contains(t, outputStr, "suggest", "debería mostrar el comando suggest")
	assert.Contains(t, outputStr, "summarize-pr", "debería mostrar el comando summarize-pr")
	assert.Contains(t, outputStr, "$0.0015", "debería mostrar el costo del primer comando")
	assert.Contains(t, outputStr, "$0.0085", "debería mostrar el costo del segundo comando")
	assert.Contains(t, outputStr, "━", "debería contener separadores visuales")

	total, err := manager.GetDailyTotal()
	assert.NoError(t, err)
	assert.Equal(t, 0.0100, total, "el total calculado debería ser 0.0100")
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
			Timestamp:    now.Add(-30 * time.Minute),
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
	assert.Contains(t, outputStr, "[CACHE]", "debería indicar cuándo hay un cache hit")
	assert.Contains(t, outputStr, "$0.0000", "debería mostrar costo cero para cache hit")
	assert.Contains(t, outputStr, "$0.0015", "debería mostrar el total correcto")
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
	assert.Contains(t, output, "No activity recorded", "debería indicar que no hay actividad")
	assert.Contains(t, output, "━", "debería contener separadores visuales")
}

func TestShowMonthlyStats_WithActivity(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	now := time.Now()

	records := []cost.ActivityRecord{
		// Día 1
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
		// Día 2
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

	assert.Contains(t, outputStr, day1, "debería mostrar el día 1")
	assert.Contains(t, outputStr, day2, "debería mostrar el día 2")
	assert.Contains(t, outputStr, "$0.0040", "debería mostrar el total del día 1 (0.0015 + 0.0025)")
	assert.Contains(t, outputStr, "$0.0010", "debería mostrar el total del día 2")
	assert.Contains(t, outputStr, "━", "debería contener separadores visuales")

	total, err := manager.GetMonthlyTotal()
	assert.NoError(t, err)
	assert.Equal(t, 0.0050, total, "el total mensual debería ser 0.0050")
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
	assert.Contains(t, outputStr, expectedDay, "debería mostrar el día agrupado")
	assert.Contains(t, outputStr, "$0.0065", "debería sumar todos los costos del día (0.0010+0.0020+0.0030+0.0005)")
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
	assert.Contains(t, buf.String(), "14:30", "debería formatear la hora como HH:MM")
	assert.Contains(t, buf.String(), "suggest", "debería mostrar el comando")
}

func setupTestManager(t *testing.T, tempDir string, records []cost.ActivityRecord) *cost.Manager {
	t.Helper()

	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tempDir)
	t.Cleanup(func() {
		_ = os.Setenv("HOME", originalHome)
	})

	trans := setupTestTranslations(t)
	manager, err := cost.NewManager(0, trans)
	require.NoError(t, err, "no debería fallar al crear el manager de prueba")

	for _, record := range records {
		err := manager.SaveActivity(record)
		require.NoError(t, err, "no debería fallar al guardar actividad de prueba")
	}

	return manager
}

func setupTestTranslations(t *testing.T) *i18n.Translations {
	t.Helper()

	localesPath := filepath.Join("..", "..", "..", "i18n", "locales")
	trans, err := i18n.NewTranslations("en", localesPath)
	require.NoError(t, err, "no debería fallar al crear traducciones de prueba")

	return trans
}

func TestStatsCommand_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Saltando test de integración en modo short")
	}

	// Arrange
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tempDir)
	t.Cleanup(func() {
		_ = os.Setenv("HOME", originalHome)
	})

	trans := setupTestTranslations(t)
	manager, err := cost.NewManager(10.0, trans)
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
	assert.NotEmpty(t, dailyBuf.String(), "debería generar output para estadísticas diarias")

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
	assert.NotEmpty(t, monthlyBuf.String(), "debería generar output para estadísticas mensuales")
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
	assert.Error(t, err, "debería retornar error cuando el historial está corrupto")
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
	assert.Error(t, err, "debería retornar error cuando el historial está corrupto")
}
