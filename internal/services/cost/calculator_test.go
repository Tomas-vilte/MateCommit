package cost

import (
	"strings"
	"testing"
)

func TestNewCalculator(t *testing.T) {
	// Act
	calc := NewCalculator()

	// Assert
	if calc == nil {
		t.Fatal("NewCalculator() returned nil")
	}
}

func TestCalculator_EstimateCost(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		model        string
		inputTokens  int
		outputTokens int
		want         float64
	}{
		{
			name:         "Gemini 1.5 Flash - exact match",
			provider:     "gemini",
			model:        "gemini-1.5-flash",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			want:         0.075 + 0.30,
		},
		{
			name:         "Gemini 1.5 Flash - case insensitive",
			provider:     "GEMINI",
			model:        "GEMINI-1.5-FLASH",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			want:         0.075 + 0.30,
		},
		{
			name:         "Gemini - partial model match (pro-preview)",
			provider:     "gemini",
			model:        "gemini-3-pro-preview-001",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			want:         2.00 + 12.00,
		},
		{
			name:         "OpenAI - GPT-4o mini",
			provider:     "openai",
			model:        "gpt-4o-mini",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			want:         0.15 + 0.60,
		},
		{
			name:         "Anthropic - Claude 3.5 Sonnet",
			provider:     "anthropic",
			model:        "claude-3-5-sonnet",
			inputTokens:  1_000_000,
			outputTokens: 1_000_000,
			want:         3.00 + 15.00,
		},
		{
			name:         "Unknown provider",
			provider:     "unknown",
			model:        "model",
			inputTokens:  1000,
			outputTokens: 1000,
			want:         0,
		},
		{
			name:         "Unknown model for known provider",
			provider:     "gemini",
			model:        "non-existent-model",
			inputTokens:  1000,
			outputTokens: 1000,
			want:         0,
		},
		{
			name:         "Zero tokens should result in zero cost",
			provider:     "gemini",
			model:        "gemini-1.5-flash",
			inputTokens:  0,
			outputTokens: 0,
			want:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			c := NewCalculator()

			// Act
			got := c.EstimateCost(tt.provider, tt.model, tt.inputTokens, tt.outputTokens)

			// Assert
			if got != tt.want {
				t.Errorf("Calculator.EstimateCost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculator_GetPricing(t *testing.T) {
	// Arrange
	c := NewCalculator()

	t.Run("Valid provider and model", func(t *testing.T) {
		// Act
		pricing, err := c.GetPricing("gemini", "gemini-1.5-flash")

		// Assert
		if err != nil {
			t.Errorf("GetPricing() error = %v, wantErr %v", err, false)
		}
		if pricing.InputPricePerMillion != 0.075 {
			t.Errorf("GetPricing() InputPricePerMillion = %v, want %v", pricing.InputPricePerMillion, 0.075)
		}
	})

	t.Run("Invalid provider", func(t *testing.T) {
		// Act
		_, err := c.GetPricing("invalid", "model")

		// Assert
		if err == nil {
			t.Error("GetPricing() error nil, want error for invalid provider")
		}
		if !strings.Contains(err.Error(), "provider") {
			t.Errorf("Error message %q doesn't mention provider", err.Error())
		}
	})

	t.Run("Invalid model", func(t *testing.T) {
		// Act
		_, err := c.GetPricing("gemini", "invalid-model")

		// Assert
		if err == nil {
			t.Error("GetPricing() error nil, want error for invalid model")
		}
		if !strings.Contains(err.Error(), "model") {
			t.Errorf("Error message %q doesn't mention model", err.Error())
		}
	})
}

func TestCalculator_AddPricing(t *testing.T) {
	// Arrange
	c := NewCalculator()
	provider := "custom-provider"
	model := "custom-model"
	table := PricingTable{InputPricePerMillion: 1.0, OutputPricePerMillion: 2.0}

	// Act
	c.AddPricing(provider, model, table)

	// Assert
	gotTable, err := c.GetPricing(provider, model)
	if err != nil {
		t.Fatalf("Failed to get added pricing: %v", err)
	}
	if gotTable != table {
		t.Errorf("GetPricing() = %v, want %v", gotTable, table)
	}

	cost := c.EstimateCost(provider, model, 1_000_000, 1_000_000)
	wantCost := 1.0 + 2.0
	if cost != wantCost {
		t.Errorf("EstimateCost() after AddPricing = %v, want %v", cost, wantCost)
	}
}
