package cost

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/fatih/color"
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

type Manager struct {
	historyPath string
	budgetDaily float64
	trans       *i18n.Translations
}

func NewManager(budgetDaily float64, trans *i18n.Translations) (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo home directory: %w", err)
	}

	matecommitDir := filepath.Join(homeDir, ".matecommit")
	if err := os.MkdirAll(matecommitDir, 0755); err != nil {
		return nil, fmt.Errorf("error creando directorio .matecommit: %w", err)
	}

	return &Manager{
		historyPath: filepath.Join(matecommitDir, "history.json"),
		budgetDaily: budgetDaily,
		trans:       trans,
	}, nil
}

// SaveActivity guarda un registro de actividad
func (m *Manager) SaveActivity(record ActivityRecord) error {
	records, err := m.loadHistory()
	if err != nil {
		records = []ActivityRecord{}
	}

	records = append(records, record)

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializando historial: %w", err)
	}

	if err := os.WriteFile(m.historyPath, data, 0644); err != nil {
		return fmt.Errorf("error guardando historial: %w", err)
	}

	return nil
}

// CheckBudget verifica si el costo estimado excede el presupuesto diario
// y muestra alertas visuales cuando se acerca al límite
func (m *Manager) CheckBudget(estimatedCost float64) error {
	if m.budgetDaily <= 0 {
		return nil
	}

	todayTotal, err := m.GetDailyTotal()
	if err != nil {
		return err
	}

	percentUsed := (todayTotal / m.budgetDaily) * 100
	newPercent := ((todayTotal + estimatedCost) / m.budgetDaily) * 100

	if percentUsed >= 50 && percentUsed < 75 {
		yellow := color.New(color.FgYellow)
		_, _ = yellow.Println(m.trans.GetMessage("cost.budget_alert_50", 0, map[string]interface{}{
			"Percent": fmt.Sprintf("%.0f", percentUsed),
			"Spent":   fmt.Sprintf("%.4f", todayTotal),
			"Limit":   fmt.Sprintf("%.2f", m.budgetDaily),
		}))
	} else if percentUsed >= 75 && percentUsed < 90 {
		yellow := color.New(color.FgYellow, color.Bold)
		_, _ = yellow.Println(m.trans.GetMessage("cost.budget_alert_75_title", 0, map[string]interface{}{
			"Percent": fmt.Sprintf("%.0f", percentUsed),
		}))
		_, _ = yellow.Println(m.trans.GetMessage("cost.budget_alert_75_spent", 0, map[string]interface{}{
			"Spent": fmt.Sprintf("%.4f", todayTotal),
			"Limit": fmt.Sprintf("%.2f", m.budgetDaily),
		}))
	} else if percentUsed >= 90 {
		red := color.New(color.FgRed, color.Bold)
		_, _ = red.Println(m.trans.GetMessage("cost.budget_alert_90_title", 0, map[string]interface{}{
			"Percent": fmt.Sprintf("%.0f", percentUsed),
		}))
		_, _ = red.Println(m.trans.GetMessage("cost.budget_alert_90_spent", 0, map[string]interface{}{
			"Spent": fmt.Sprintf("%.4f", todayTotal),
			"Limit": fmt.Sprintf("%.2f", m.budgetDaily),
		}))
		_, _ = red.Println(m.trans.GetMessage("cost.budget_alert_90_remaining", 0, map[string]interface{}{
			"Remaining": fmt.Sprintf("%.4f", m.budgetDaily-todayTotal),
		}))
	}

	if newPercent > 100 {
		red := color.New(color.FgRed, color.Bold)
		fmt.Println()
		_, _ = red.Println(m.trans.GetMessage("cost.budget_exceeded_title", 0, nil))
		fmt.Println(m.trans.GetMessage("cost.budget_exceeded_spent_today", 0, map[string]interface{}{
			"Spent": fmt.Sprintf("%.4f", todayTotal),
		}))
		fmt.Println(m.trans.GetMessage("cost.budget_exceeded_estimated", 0, map[string]interface{}{
			"Cost": fmt.Sprintf("%.4f", estimatedCost),
		}))
		fmt.Println(m.trans.GetMessage("cost.budget_exceeded_total", 0, map[string]interface{}{
			"Total": fmt.Sprintf("%.4f", todayTotal+estimatedCost),
		}))
		fmt.Println(m.trans.GetMessage("cost.budget_exceeded_limit", 0, map[string]interface{}{
			"Limit": fmt.Sprintf("%.2f", m.budgetDaily),
		}))
		fmt.Println(m.trans.GetMessage("cost.budget_exceeded_excess", 0, map[string]interface{}{
			"Excess": fmt.Sprintf("%.4f", (todayTotal+estimatedCost)-m.budgetDaily),
		}))
		fmt.Println()

		return fmt.Errorf("presupuesto diario excedido: actual $%.4f + estimado $%.4f > límite $%.2f",
			todayTotal, estimatedCost, m.budgetDaily)
	}

	return nil
}

// GetDailyTotal obtiene el total gastado hoy
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

// GetMonthlyTotal obtiene el total gastado este mes
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

// GetHistory obtiene todos los registros
func (m *Manager) GetHistory() ([]ActivityRecord, error) {
	return m.loadHistory()
}

func (m *Manager) loadHistory() ([]ActivityRecord, error) {
	data, err := os.ReadFile(m.historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ActivityRecord{}, nil
		}
		return nil, fmt.Errorf("error leyendo historial: %w", err)
	}

	var records []ActivityRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("error deserializando historial: %w", err)
	}

	return records, nil
}
