package cost

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type ActivityRecord struct {
	Timestamp    time.Time `json:"timestamp"`
	Command      string    `json:"command"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	TokensInput  int       `json:"tokens_input"`
	TokensOutput int       `json:"tokens_output"`
	CostUSD      float64   `json:"cost_usd"`
	DurationMs   int64     `json:"duration_ms"`
	CacheHit     bool      `json:"cache_hit"`
	Hash         string    `json:"hash"`
}

type BudgetStatus struct {
	IsExceeded   bool
	PercentUsed  float64
	TodayTotal   float64
	Estimated    float64
	Limit        float64
	IsWarning    bool
	WarningLevel int // 50, 75, 90
}

type Manager struct {
	historyPath string
	budgetDaily float64
}

type CommandStats struct {
	Command      string
	CallCount    int
	TotalCost    float64
	TotalTokens  int
	AvgCost      float64
	CacheHitRate float64
}

type StatsBreakdown struct {
	ByCommand  []CommandStats
	TotalCalls int
	TotalCost  float64
}

type ForecastStats struct {
	MonthToDate       float64
	DaysInMonth       int
	DaysElapsed       int
	DailyAverage      float64
	ProjectedMonthEnd float64
}

func NewManager(budgetDaily float64) (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error getting home directory: %w", err)
	}

	matecommitDir := filepath.Join(homeDir, ".matecommit")
	if err := os.MkdirAll(matecommitDir, 0755); err != nil {
		return nil, fmt.Errorf("error creating .matecommit directory: %w", err)
	}

	return &Manager{
		historyPath: filepath.Join(matecommitDir, "history.json"),
		budgetDaily: budgetDaily,
	}, nil
}

// SaveActivity saves an activity record
func (m *Manager) SaveActivity(record ActivityRecord) error {
	slog.Debug("saving activity record",
		"command", record.Command,
		"provider", record.Provider,
		"model", record.Model,
		"tokens_input", record.TokensInput,
		"tokens_output", record.TokensOutput,
		"cost_usd", record.CostUSD,
		"cache_hit", record.CacheHit)

	records, err := m.loadHistory()
	if err != nil {
		records = []ActivityRecord{}
	}

	records = append(records, record)

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		slog.Error("failed to serialize activity history",
			"error", err)
		return fmt.Errorf("error serializing history: %w", err)
	}

	if err := os.WriteFile(m.historyPath, data, 0644); err != nil {
		slog.Error("failed to write activity history",
			"path", m.historyPath,
			"error", err)
		return fmt.Errorf("error saving history: %w", err)
	}

	slog.Debug("activity record saved successfully",
		"total_records", len(records))

	return nil
}

// CheckBudget checks if the estimated cost exceeds the daily budget
func (m *Manager) CheckBudget(estimatedCost float64) (*BudgetStatus, error) {
	slog.Debug("checking budget",
		"estimated_cost", estimatedCost,
		"budget_daily", m.budgetDaily)

	if m.budgetDaily <= 0 {
		slog.Debug("no budget limit configured")
		return &BudgetStatus{}, nil
	}

	todayTotal, err := m.GetDailyTotal()
	if err != nil {
		slog.Error("failed to get daily total",
			"error", err)
		return nil, err
	}

	percentUsed := (todayTotal / m.budgetDaily) * 100
	newPercent := ((todayTotal + estimatedCost) / m.budgetDaily) * 100

	status := &BudgetStatus{
		IsExceeded:  newPercent > 100,
		PercentUsed: percentUsed,
		TodayTotal:  todayTotal,
		Estimated:   estimatedCost,
		Limit:       m.budgetDaily,
	}

	if percentUsed >= 90 {
		status.IsWarning = true
		status.WarningLevel = 90
	} else if percentUsed >= 75 {
		status.IsWarning = true
		status.WarningLevel = 75
	} else if percentUsed >= 50 {
		status.IsWarning = true
		status.WarningLevel = 50
	}

	slog.Info("budget check completed",
		"today_total", todayTotal,
		"estimated_cost", estimatedCost,
		"percent_used", percentUsed,
		"is_exceeded", status.IsExceeded,
		"is_warning", status.IsWarning)

	return status, nil
}

// GetDailyTotal gets the total spent today
func (m *Manager) GetDailyTotal() (float64, error) {
	records, err := m.loadHistory()
	if err != nil {
		return 0, nil
	}

	today := time.Now().Format("2006-01-02")
	var total float64

	for _, record := range records {
		if record.Timestamp.Format("2006-01-02") == today {
			total += record.CostUSD
		}
	}

	return total, nil
}

// GetMonthlyTotal gets the total spent this month
func (m *Manager) GetMonthlyTotal() (float64, error) {
	records, err := m.loadHistory()
	if err != nil {
		return 0, nil
	}

	currentMonth := time.Now().Format("2006-01")
	var total float64

	for _, record := range records {
		if record.Timestamp.Format("2006-01") == currentMonth {
			total += record.CostUSD
		}
	}

	return total, nil
}

// GetHistory gets all records
func (m *Manager) GetHistory() ([]ActivityRecord, error) {
	return m.loadHistory()
}

// GetBreakdownByCommand returns usage statistics grouped by command
func (m *Manager) GetBreakdownByCommand() (*StatsBreakdown, error) {
	records, err := m.GetHistory()
	if err != nil {
		return nil, err
	}

	currentMonth := time.Now().Format("2006-01")
	commandMap := make(map[string]*CommandStats)
	totalCalls := 0
	totalCost := 0.0

	for _, record := range records {
		if record.Timestamp.Format("2006-01") != currentMonth {
			continue
		}

		totalCalls++
		totalCost += record.CostUSD

		stats, exists := commandMap[record.Command]
		if !exists {
			stats = &CommandStats{
				Command: record.Command,
			}
			commandMap[record.Command] = stats
		}

		stats.CallCount++
		stats.TotalCost += record.CostUSD
		stats.TotalTokens += record.TokensInput + record.TokensOutput

		if record.CacheHit {
			stats.CacheHitRate++
		}
	}

	breakdown := &StatsBreakdown{
		ByCommand:  make([]CommandStats, 0, len(commandMap)),
		TotalCalls: totalCalls,
		TotalCost:  totalCost,
	}

	for _, stats := range commandMap {
		if stats.CallCount > 0 {
			stats.AvgCost = stats.TotalCost / float64(stats.CallCount)
			stats.CacheHitRate = (stats.CacheHitRate / float64(stats.CallCount)) * 100
		}
		breakdown.ByCommand = append(breakdown.ByCommand, *stats)
	}

	sort.Slice(breakdown.ByCommand, func(i, j int) bool {
		return breakdown.ByCommand[i].TotalCost > breakdown.ByCommand[j].TotalCost
	})
	return breakdown, nil
}

// GetForecast calculates projected monthly spending
func (m *Manager) GetForecast() (*ForecastStats, error) {
	monthTotal, err := m.GetMonthlyTotal()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	daysInMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	daysElapsed := now.Day()

	var dailyAvg float64
	if daysElapsed > 0 {
		dailyAvg = monthTotal / float64(daysElapsed)
	}

	projected := dailyAvg * float64(daysInMonth)

	return &ForecastStats{
		MonthToDate:       monthTotal,
		DaysInMonth:       daysInMonth,
		DaysElapsed:       daysElapsed,
		DailyAverage:      dailyAvg,
		ProjectedMonthEnd: projected,
	}, nil
}

// GetCacheStats returns cache hit statistics
func (m *Manager) GetCacheStats() (hitRate float64, totalSaved float64, err error) {
	records, err := m.GetHistory()
	if err != nil {
		return 0, 0, err
	}

	currentMonth := time.Now().Format("2006-01")
	totalCalls := 0
	cacheHits := 0
	var saved float64

	for _, record := range records {
		if record.Timestamp.Format("2006-01") != currentMonth {
			continue
		}

		totalCalls++
		if record.CacheHit {
			cacheHits++
			saved += record.CostUSD
		}
	}

	if totalCalls > 0 {
		hitRate = (float64(cacheHits) / float64(totalCalls)) * 100
	}

	return hitRate, saved, nil
}

func (m *Manager) loadHistory() ([]ActivityRecord, error) {
	data, err := os.ReadFile(m.historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ActivityRecord{}, nil
		}
		return nil, fmt.Errorf("error reading history: %w", err)
	}

	var records []ActivityRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("error deserializing history: %w", err)
	}

	return records, nil
}
