package cost

import (
	"fmt"
	"strings"
)

type PricingTable struct {
	InputPricePerMillion  float64
	OutputPricePerMillion float64
}

type ProviderPricing map[string]map[string]PricingTable

// https://ai.google.dev/gemini-api/docs/pricing
var pricing = ProviderPricing{
	"gemini": {
		"gemini-1.5-flash":       {InputPricePerMillion: 0.075, OutputPricePerMillion: 0.30},
		"gemini-1.5-pro":         {InputPricePerMillion: 1.25, OutputPricePerMillion: 5.00},
		"gemini-2.5-flash":       {InputPricePerMillion: 0.10, OutputPricePerMillion: 0.40},
		"gemini-3-flash-preview": {InputPricePerMillion: 0.50, OutputPricePerMillion: 3.00},
		"gemini-3-pro-preview":   {InputPricePerMillion: 2.00, OutputPricePerMillion: 12.00},
	},
	"openai": {
		"gpt-4o":      {InputPricePerMillion: 2.50, OutputPricePerMillion: 10.00},
		"gpt-4o-mini": {InputPricePerMillion: 0.15, OutputPricePerMillion: 0.60},
		"gpt-4-turbo": {InputPricePerMillion: 10.00, OutputPricePerMillion: 30.00},
	},
	"anthropic": {
		"claude-3-5-sonnet": {InputPricePerMillion: 3.00, OutputPricePerMillion: 15.00},
		"claude-3-haiku":    {InputPricePerMillion: 0.25, OutputPricePerMillion: 1.25},
	},
}

type Calculator struct{}

func NewCalculator() *Calculator {
	return &Calculator{}
}

// EstimateCost calculates the estimated cost based on provider, model, and tokens
func (c *Calculator) EstimateCost(provider, model string, inputTokens, outputTokens int) float64 {
	provider = strings.ToLower(provider)
	model = strings.ToLower(model)

	providerPricing, exists := pricing[provider]
	if !exists {
		return 0
	}

	modelPricing, exists := providerPricing[model]
	if !exists {
		for modelName, prices := range providerPricing {
			if strings.Contains(model, modelName) {
				modelPricing = prices
				break
			}
		}
		if modelPricing.InputPricePerMillion == 0 {
			return 0
		}
	}

	inputCost := (float64(inputTokens) / 1_000_000) * modelPricing.InputPricePerMillion
	outputCost := (float64(outputTokens) / 1_000_000) * modelPricing.OutputPricePerMillion

	return inputCost + outputCost
}

// GetPricing returns the pricing table for a provider and model
func (c *Calculator) GetPricing(provider, model string) (PricingTable, error) {
	provider = strings.ToLower(provider)
	model = strings.ToLower(model)

	providerPricing, exists := pricing[provider]
	if !exists {
		return PricingTable{}, fmt.Errorf("provider %s not found", provider)
	}

	modelPricing, exists := providerPricing[model]
	if !exists {
		return PricingTable{}, fmt.Errorf("model %s not found for provider %s", model, provider)
	}

	return modelPricing, nil
}

// AddPricing allows adding pricing dynamically (useful for testing or new models)
func (c *Calculator) AddPricing(provider, model string, table PricingTable) {
	provider = strings.ToLower(provider)
	model = strings.ToLower(model)

	if _, exists := pricing[provider]; !exists {
		pricing[provider] = make(map[string]PricingTable)
	}
	pricing[provider][model] = table
}
