package models

type TokenUsage struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalTokens  int     `json:"total_tokens"`
	CostUSD      float64 `json:"cost_usd,omitempty"`
	Model        string  `json:"model,omitempty"`
	CacheHit     bool    `json:"cache_hit,omitempty"`
	DurationMs   int64   `json:"duration_ms,omitempty"`
}
