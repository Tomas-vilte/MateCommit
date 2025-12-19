package routing

import (
	"testing"
)

func TestNewModelSelector(t *testing.T) {
	// Act
	selector := NewModelSelector()

	// Assert
	if selector == nil {
		t.Fatal("NewModelSelector() returned nil")
	}
}

func TestModelSelector_SelectBestModel(t *testing.T) {
	tests := []struct {
		name            string
		operation       string
		estimatedTokens int
		want            string
	}{
		{
			name:            "Generate release operation should return high quality model",
			operation:       "generate-release",
			estimatedTokens: 100,
			want:            "gemini-3-pro-preview",
		},
		{
			name:            "Generate issue operation should return high quality model",
			operation:       "generate-issue",
			estimatedTokens: 100,
			want:            "gemini-3-pro-preview",
		},
		{
			name:            "High token count should return flash-preview model",
			operation:       "summarize",
			estimatedTokens: 20000,
			want:            "gemini-3-flash-preview",
		},
		{
			name:            "Boundary token count should return flash-preview model",
			operation:       "summarize",
			estimatedTokens: 15001,
			want:            "gemini-3-flash-preview",
		},
		{
			name:            "Exact boundary token count should return default model",
			operation:       "summarize",
			estimatedTokens: 15000,
			want:            "gemini-2.5-flash",
		},
		{
			name:            "Small token count should return default model",
			operation:       "summarize",
			estimatedTokens: 500,
			want:            "gemini-2.5-flash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			m := &ModelSelector{}

			// Act
			got := m.SelectBestModel(tt.operation, tt.estimatedTokens)

			// Assert
			if got != tt.want {
				t.Errorf("ModelSelector.SelectBestModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelSelector_GetRationale(t *testing.T) {
	tests := []struct {
		name          string
		selectedModel string
		want          string
	}{
		{
			name:          "High quality model rationale",
			selectedModel: "gemini-3-pro-preview",
			want:          "routing.reason_high_quality",
		},
		{
			name:          "Large context model rationale",
			selectedModel: "gemini-3-flash-preview",
			want:          "routing.reason_large",
		},
		{
			name:          "Balance model rationale (from comments, even if mismatched in code constant)",
			selectedModel: "gemini-1.5-flash",
			want:          "routing.reason_balance",
		},
		{
			name:          "Unknown model should return default rationale",
			selectedModel: "unknown-model",
			want:          "routing.reason_default",
		},
		{
			name:          "Default model used in SelectBestModel should return default rationale",
			selectedModel: "gemini-2.5-flash",
			want:          "routing.reason_default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			m := &ModelSelector{}

			// Act
			got := m.GetRationale(tt.selectedModel)

			// Assert
			if got != tt.want {
				t.Errorf("ModelSelector.GetRationale() = %v, want %v", got, tt.want)
			}
		})
	}
}
