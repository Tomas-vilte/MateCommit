package cost

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
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
