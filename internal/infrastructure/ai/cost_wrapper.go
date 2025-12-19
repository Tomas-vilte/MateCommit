package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/cache"
	cost2 "github.com/Tomas-vilte/MateCommit/internal/services/cost"
	"github.com/Tomas-vilte/MateCommit/internal/services/routing"
	"github.com/fatih/color"
)

type GenerateFunc func(ctx context.Context, modelName string, prompt string) (interface{}, *models.TokenUsage, error)

type CostAwareWrapper struct {
	provider              ports.CostAwareAIProvider
	calculator            *cost2.Calculator
	manager               *cost2.Manager
	cache                 *cache.Cache
	modelSelector         *routing.ModelSelector
	trans                 *i18n.Translations
	estimatedOutputTokens int
	skipConfirmation      bool
}

type WrapperConfig struct {
	Provider              ports.CostAwareAIProvider
	BudgetDaily           float64
	Trans                 *i18n.Translations
	EstimatedOutputTokens int
	SkipConfirmation      bool
}

// NewCostAwareWrapper crea un wrapper agnóstico de proveedor
func NewCostAwareWrapper(cfg WrapperConfig) (*CostAwareWrapper, error) {
	manager, err := cost2.NewManager(cfg.BudgetDaily, cfg.Trans)
	if err != nil {
		return nil, fmt.Errorf("error creando cost manager: %w", err)
	}

	cacheService, err := cache.NewCache(24 * time.Hour)
	if err != nil {
		return nil, fmt.Errorf("error creando cache: %w", err)
	}

	return &CostAwareWrapper{
		provider:              cfg.Provider,
		calculator:            cost2.NewCalculator(),
		manager:               manager,
		cache:                 cacheService,
		modelSelector:         routing.NewModelSelector(),
		trans:                 cfg.Trans,
		estimatedOutputTokens: cfg.EstimatedOutputTokens,
		skipConfirmation:      cfg.SkipConfirmation,
	}, nil
}

// WrapGenerate envuelve cualquier función de generación con tracking
func (w *CostAwareWrapper) WrapGenerate(
	ctx context.Context,
	command string,
	prompt string,
	generateFn GenerateFunc,
) (interface{}, *models.TokenUsage, error) {
	startTime := time.Now()

	providerName := w.provider.GetProviderName()
	originalModel := w.provider.GetModelName()
	modelToUse := originalModel

	contentHash := w.cache.GenerateHash(providerName + originalModel + prompt)

	if cachedData, hit, err := w.cache.Get(contentHash); err == nil && hit {
		var cachedResp interface{}
		if err := json.Unmarshal(cachedData, &cachedResp); err == nil {
			usage := &models.TokenUsage{
				CacheHit:   true,
				CostUSD:    0,
				DurationMs: time.Since(startTime).Milliseconds(),
				Model:      originalModel,
			}
			return cachedResp, usage, nil
		}
	}

	var inputTokens int
	tokens, err := w.provider.CountTokens(ctx, prompt)
	if err == nil {
		inputTokens = tokens
	}

	suggestedModel := w.modelSelector.SelectBestModel(command, inputTokens)
	hasSuggestion := suggestedModel != originalModel

	estimatedCost := w.calculator.EstimateCost(providerName, originalModel, inputTokens, w.estimatedOutputTokens)

	if hasSuggestion && !w.skipConfirmation {
		rationaleKey := w.modelSelector.GetRationale(command, suggestedModel)
		rationale := w.trans.GetMessage(rationaleKey, 0, nil)
		yellow := color.New(color.FgYellow)
		_, _ = yellow.Println(w.trans.GetMessage("cost.routing_suggestion", 0, map[string]interface{}{
			"Rationale": rationale,
		}))
		_, _ = yellow.Println(w.trans.GetMessage("cost.routing_suggested_model", 0, map[string]interface{}{
			"Suggested": suggestedModel,
			"Current":   originalModel,
		}))

		estimatedCost = w.calculator.EstimateCost(providerName, suggestedModel, inputTokens, w.estimatedOutputTokens)
	}

	if err := w.manager.CheckBudget(estimatedCost); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", w.trans.GetMessage("cost.budget_exceeded", 0, nil), err)
	}

	if (estimatedCost > 0.005 || hasSuggestion) && !w.skipConfirmation {
		choice, proceed := w.askUserConfirmation(estimatedCost, inputTokens, w.estimatedOutputTokens, suggestedModel)
		if !proceed {
			return nil, nil, fmt.Errorf("operación cancelada por el usuario")
		}
		if choice == "suggested" {
			modelToUse = suggestedModel
		}
	}

	resp, usage, err := generateFn(ctx, modelToUse, prompt)
	if err != nil {
		return nil, nil, err
	}

	_ = w.cache.Set(contentHash, resp)

	if usage != nil {
		usage.Model = modelToUse
		usage.CostUSD = w.calculator.EstimateCost(providerName, modelToUse, usage.InputTokens, usage.OutputTokens)
		usage.DurationMs = time.Since(startTime).Milliseconds()
		usage.CacheHit = false

		_ = w.manager.SaveActivity(cost2.ActivityRecord{
			Timestamp:    time.Now(),
			Command:      command,
			Provider:     providerName,
			Model:        modelToUse,
			TokensInput:  usage.InputTokens,
			TokensOutput: usage.OutputTokens,
			CostUSD:      usage.CostUSD,
			DurationMs:   usage.DurationMs,
			CacheHit:     false,
			Hash:         contentHash,
		})
	}

	return resp, usage, nil
}

// askUserConfirmation pregunta al usuario si desea continuar y permite cambiar al modelo sugerido
func (w *CostAwareWrapper) askUserConfirmation(estimatedCost float64, inputTokens, outputTokens int, suggestedModel string) (string, bool) {
	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)
	originalModel := w.provider.GetModelName()
	hasSuggestion := suggestedModel != "" && suggestedModel != originalModel

	fmt.Println()
	_, _ = cyan.Println(w.trans.GetMessage("cost.confirmation_separator", 0, nil))
	_, _ = cyan.Println(w.trans.GetMessage("cost.confirmation_header", 0, nil))
	_, _ = cyan.Println(w.trans.GetMessage("cost.confirmation_separator", 0, nil))

	fmt.Println(w.trans.GetMessage("cost.confirmation_input_tokens", 0, map[string]interface{}{
		"Tokens": yellow.Sprintf("%d", inputTokens),
	}))
	fmt.Println(w.trans.GetMessage("cost.confirmation_output_tokens", 0, map[string]interface{}{
		"Tokens": yellow.Sprintf("%d", outputTokens),
	}))

	costLabel := w.trans.GetMessage("cost.confirmation_estimated_cost", 0, map[string]interface{}{
		"Cost": yellow.Sprintf("$%.4f", estimatedCost),
	})
	if hasSuggestion {
		fmt.Printf("%s (%s)\n", costLabel, suggestedModel)
	} else {
		fmt.Println(costLabel)
	}

	_, _ = cyan.Println(w.trans.GetMessage("cost.confirmation_separator", 0, nil))

	if hasSuggestion {
		fmt.Printf("%s ", w.trans.GetMessage("cost.confirmation_use_suggested", 0, nil))
		fmt.Printf("%s\n", color.HiBlackString(w.trans.GetMessage("cost.confirmation_use_suggested_help", 0, nil)))
	} else {
		fmt.Printf("%s ", w.trans.GetMessage("cost.confirmation_prompt", 0, nil))
	}

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", false
	}

	response = strings.TrimSpace(strings.ToLower(response))

	if !hasSuggestion {
		proceed := response == "" || response == "y" || response == "yes" || response == "si" || response == "s"
		if proceed {
			return "original", true
		}
		return "", false
	}

	switch response {
	case "", "y", "yes", "si", "s":
		return "suggested", true
	case "m", "stay":
		return "original", true
	case "c", "cancel", "n", "no":
		return "", false
	default:
		return "", false
	}
}
