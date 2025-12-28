package cost

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestManager(t *testing.T, budgetDaily float64) (*Manager, string) {
	tempDir, err := os.MkdirTemp("", "matecommit-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	m := &Manager{
		historyPath: filepath.Join(tempDir, "history.json"),
		budgetDaily: budgetDaily,
	}

	return m, tempDir
}

func TestNewManager(t *testing.T) {
	// Act
	m, err := NewManager(1.0)

	// Assert
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
	if m.budgetDaily != 1.0 {
		t.Errorf("expected budget 1.0, got %v", m.budgetDaily)
	}
}

func TestManager_SaveAndLoadActivity(t *testing.T) {
	// Arrange
	m, tempDir := setupTestManager(t, 1.0)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("RemoveAll() error = %v", err)
		}
	}()
	record := ActivityRecord{
		Timestamp:    time.Now(),
		Command:      "generate-commit",
		Provider:     "gemini",
		Model:        "gemini-1.5-flash",
		TokensInput:  1000,
		TokensOutput: 500,
		CostUSD:      0.001,
	}

	// Act
	err := m.SaveActivity(record)
	if err != nil {
		t.Fatalf("SaveActivity() error = %v", err)
	}

	history, err := m.GetHistory()

	// Assert
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 record, got %v", len(history))
	}
	if history[0].Command != record.Command {
		t.Errorf("expected command %s, got %s", record.Command, history[0].Command)
	}
}

func TestManager_Totals(t *testing.T) {
	// Arrange
	m, tempDir := setupTestManager(t, 10.0)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("RemoveAll() error = %v", err)
		}
	}()
	now := time.Now()
	todayRecord := ActivityRecord{Timestamp: now, CostUSD: 1.5}
	yesterdayRecord := ActivityRecord{Timestamp: now.AddDate(0, 0, -1), CostUSD: 2.0}
	lastMonthRecord := ActivityRecord{Timestamp: now.AddDate(0, -1, 0), CostUSD: 5.0}

	records := []ActivityRecord{todayRecord, yesterdayRecord, lastMonthRecord}
	data, _ := json.Marshal(records)
	_ = os.WriteFile(m.historyPath, data, 0644)

	// Act
	dailyTotal, errDaily := m.GetDailyTotal()
	monthlyTotal, errMonthly := m.GetMonthlyTotal()

	// Assert
	if errDaily != nil {
		t.Errorf("GetDailyTotal() error = %v", errDaily)
	}
	if dailyTotal != 1.5 {
		t.Errorf("dailyTotal = %v, want 1.5", dailyTotal)
	}

	if errMonthly != nil {
		t.Errorf("GetMonthlyTotal() error = %v", errMonthly)
	}
	expectedMonthly := 1.5
	if yesterdayRecord.Timestamp.Format("2006-01") == now.Format("2006-01") {
		expectedMonthly += 2.0
	}

	if monthlyTotal != expectedMonthly {
		t.Errorf("monthlyTotal = %v, want %v", monthlyTotal, expectedMonthly)
	}
}

func TestManager_CheckBudget(t *testing.T) {
	tests := []struct {
		name          string
		budget        float64
		existingSpend float64
		estimated     float64
		wantErr       bool
	}{
		{
			name:          "Budget not exceeded",
			budget:        1.0,
			existingSpend: 0.1,
			estimated:     0.1,
			wantErr:       false,
		},
		{
			name:          "Budget exceeded exactly",
			budget:        1.0,
			existingSpend: 0.5,
			estimated:     0.6,
			wantErr:       true,
		},
		{
			name:          "Budget exceeded by far",
			budget:        1.0,
			existingSpend: 1.1,
			estimated:     0.1,
			wantErr:       true,
		},
		{
			name:          "Zero budget disables check",
			budget:        0,
			existingSpend: 10.0,
			estimated:     1.0,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			m, tempDir := setupTestManager(t, tt.budget)
			defer func() {
				if err := os.RemoveAll(tempDir); err != nil {
					t.Errorf("RemoveAll() error = %v", err)
				}
			}()
			if tt.existingSpend > 0 {
				record := ActivityRecord{Timestamp: time.Now(), CostUSD: tt.existingSpend}
				_ = m.SaveActivity(record)
			}

			// Act
			status, err := m.CheckBudget(tt.estimated)

			// Assert
			if err != nil && !tt.wantErr {
				t.Errorf("CheckBudget() unexpected error = %v", err)
			}
			if status != nil && status.IsExceeded != tt.wantErr {
				t.Errorf("CheckBudget() status.IsExceeded = %v, want %v", status.IsExceeded, tt.wantErr)
			}
		})
	}
}

func TestManager_CheckBudgetAlerts(t *testing.T) {
	tests := []struct {
		name          string
		budget        float64
		existingSpend float64
	}{
		{
			name:          "Budget usage 55% (50-75 range)",
			budget:        10.0,
			existingSpend: 5.5,
		},
		{
			name:          "Budget usage 80% (75-90 range)",
			budget:        10.0,
			existingSpend: 8.0,
		},
		{
			name:          "Budget usage 95% (90+ range)",
			budget:        10.0,
			existingSpend: 9.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			m, tempDir := setupTestManager(t, tt.budget)
			defer func() {
				if err := os.RemoveAll(tempDir); err != nil {
					t.Errorf("RemoveAll() error = %v", err)
				}
			}()

			record := ActivityRecord{Timestamp: time.Now(), CostUSD: tt.existingSpend}
			_ = m.SaveActivity(record)

			// Act
			status, err := m.CheckBudget(0.01)

			// Assert
			if err != nil {
				t.Errorf("CheckBudget() unexpected error = %v", err)
			}
			if status == nil {
				t.Fatal("CheckBudget() returned nil status")
			}
			if !status.IsWarning {
				t.Errorf("CheckBudget() expected status.IsWarning = true for spend %v", tt.existingSpend)
			}
		})
	}
}

func TestGetBreakdownByCommand(t *testing.T) {
	t.Run("should return breakdown grouped by command", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		manager, err := NewManager(0)
		require.NoError(t, err)
		manager.historyPath = tmpDir + "/history.json"

		now := time.Now()
		records := []ActivityRecord{
			{
				Timestamp:    now,
				Command:      "suggest",
				CostUSD:      0.0032,
				TokensInput:  245,
				TokensOutput: 156,
				CacheHit:     false,
			},
			{
				Timestamp:    now.Add(-1 * time.Hour),
				Command:      "suggest",
				CostUSD:      0.0028,
				TokensInput:  230,
				TokensOutput: 145,
				CacheHit:     true,
			},
			{
				Timestamp:    now.Add(-2 * time.Hour),
				Command:      "summarize-pr",
				CostUSD:      0.0156,
				TokensInput:  1024,
				TokensOutput: 512,
				CacheHit:     false,
			},
		}

		for _, r := range records {
			err = manager.SaveActivity(r)
			require.NoError(t, err)
		}

		// Act
		breakdown, err := manager.GetBreakdownByCommand()

		// Assert
		require.NoError(t, err)
		require.NotNil(t, breakdown)
		assert.Equal(t, 3, breakdown.TotalCalls)
		assert.Equal(t, 0.0032+0.0028+0.0156, breakdown.TotalCost)

		var suggestStats *CommandStats
		for i := range breakdown.ByCommand {
			if breakdown.ByCommand[i].Command == "suggest" {
				suggestStats = &breakdown.ByCommand[i]
				break
			}
		}
		require.NotNil(t, suggestStats)
		assert.Equal(t, 2, suggestStats.CallCount)
		assert.Equal(t, 0.0032+0.0028, suggestStats.TotalCost)
		assert.Equal(t, 245+156+230+145, suggestStats.TotalTokens)
		assert.InDelta(t, 50.0, suggestStats.CacheHitRate, 1.0)
	})

	t.Run("should handle empty history", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		manager, err := NewManager(0)
		require.NoError(t, err)
		manager.historyPath = tmpDir + "/history.json"

		// Act
		breakdown, err := manager.GetBreakdownByCommand()

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 0, breakdown.TotalCalls)
		assert.Equal(t, 0.0, breakdown.TotalCost)
		assert.Empty(t, breakdown.ByCommand)
	})

	t.Run("should only include current month", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		manager, err := NewManager(0)
		require.NoError(t, err)
		manager.historyPath = tmpDir + "/history.json"

		now := time.Now()
		lastMonth := now.AddDate(0, -1, 0)

		records := []ActivityRecord{
			{
				Timestamp: now,
				Command:   "suggest",
				CostUSD:   0.0032,
			},
			{
				Timestamp: lastMonth,
				Command:   "suggest",
				CostUSD:   0.0050,
			},
		}

		for _, r := range records {
			err = manager.SaveActivity(r)
			require.NoError(t, err)
		}

		// Act
		breakdown, err := manager.GetBreakdownByCommand()

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 1, breakdown.TotalCalls)
		assert.Equal(t, 0.0032, breakdown.TotalCost)
	})
}

func TestGetForecast(t *testing.T) {
	t.Run("should calculate forecast correctly", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		manager, err := NewManager(0)
		require.NoError(t, err)
		manager.historyPath = tmpDir + "/history.json"

		now := time.Now()
		daysInMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
		daysElapsed := now.Day()

		totalSpent := 0.15
		for i := 0; i < 5; i++ {
			err := manager.SaveActivity(ActivityRecord{
				Timestamp: now.AddDate(0, 0, -i),
				Command:   "suggest",
				CostUSD:   0.03,
			})
			require.NoError(t, err)
		}

		// Act
		forecast, err := manager.GetForecast()

		// Assert
		require.NoError(t, err)
		require.NotNil(t, forecast)
		assert.Equal(t, daysInMonth, forecast.DaysInMonth)
		assert.Equal(t, daysElapsed, forecast.DaysElapsed)
		assert.Equal(t, totalSpent, forecast.MonthToDate)
		assert.InDelta(t, totalSpent/float64(daysElapsed), forecast.DailyAverage, 0.001)
		assert.InDelta(t, forecast.DailyAverage*float64(daysInMonth), forecast.ProjectedMonthEnd, 0.01)
	})

	t.Run("should handle no activity", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		manager, err := NewManager(0)
		require.NoError(t, err)
		manager.historyPath = tmpDir + "/history.json"

		// Act
		forecast, err := manager.GetForecast()

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 0.0, forecast.MonthToDate)
		assert.Equal(t, 0.0, forecast.DailyAverage)
		assert.Equal(t, 0.0, forecast.ProjectedMonthEnd)
	})
}

func TestGetCacheStats(t *testing.T) {
	t.Run("should calculate cache hit rate correctly", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		manager, err := NewManager(0)
		require.NoError(t, err)
		manager.historyPath = tmpDir + "/history.json"

		now := time.Now()
		records := []ActivityRecord{
			{Timestamp: now, Command: "suggest", CostUSD: 0.003, CacheHit: false},
			{Timestamp: now.Add(-1 * time.Hour), Command: "suggest", CostUSD: 0.002, CacheHit: true},
			{Timestamp: now.Add(-2 * time.Hour), Command: "suggest", CostUSD: 0.002, CacheHit: true},
			{Timestamp: now.Add(-3 * time.Hour), Command: "suggest", CostUSD: 0.003, CacheHit: false},
		}

		for _, r := range records {
			err = manager.SaveActivity(r)
			require.NoError(t, err)
		}

		// Act
		hitRate, saved, err := manager.GetCacheStats()

		// Assert
		require.NoError(t, err)
		assert.InDelta(t, 50.0, hitRate, 1.0)
		assert.Equal(t, 0.004, saved)
	})

	t.Run("should handle no cache hits", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		manager, err := NewManager(0)
		require.NoError(t, err)
		manager.historyPath = tmpDir + "/history.json"

		now := time.Now()
		err = manager.SaveActivity(ActivityRecord{
			Timestamp: now,
			Command:   "suggest",
			CostUSD:   0.003,
			CacheHit:  false,
		})
		require.NoError(t, err)

		// Act
		hitRate, saved, err := manager.GetCacheStats()

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 0.0, hitRate)
		assert.Equal(t, 0.0, saved)
	})

	t.Run("should only count current month", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		manager, err := NewManager(0)
		require.NoError(t, err)
		manager.historyPath = tmpDir + "/history.json"

		now := time.Now()
		lastMonth := now.AddDate(0, -1, 0)

		records := []ActivityRecord{
			{Timestamp: now, Command: "suggest", CostUSD: 0.002, CacheHit: true},
			{Timestamp: lastMonth, Command: "suggest", CostUSD: 0.002, CacheHit: true}, // Should not count
		}

		for _, r := range records {
			err = manager.SaveActivity(r)
			require.NoError(t, err)
		}

		// Act
		hitRate, saved, err := manager.GetCacheStats()

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 100.0, hitRate)
		assert.Equal(t, 0.002, saved)
	})
}
