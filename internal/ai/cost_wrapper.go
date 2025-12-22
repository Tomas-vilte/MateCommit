package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/thomas-vilte/matecommit/internal/cache"
	"github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/ports"
	"github.com/thomas-vilte/matecommit/internal/services/cost"
	"github.com/thomas-vilte/matecommit/internal/services/routing"
)

type ConfirmationCallback func(result ConfirmationResult) (choice string, proceed bool)

type GenerateFunc func(ctx context.Context, model string, prompt string) (interface{}, *models.TokenUsage, error)

type ConfirmationResult struct {
	EstimatedCost  float64
	InputTokens    int
	OutputTokens   int
	SuggestedModel string
	CurrentModel   string
	RationaleKey   string
}

type CostAwareWrapper struct {
	provider              ports.CostAwareAIProvider
	calculator            *cost.Calculator
	manager               *cost.Manager
	cache                 *cache.Cache
	modelSelector         *routing.ModelSelector
	estimatedOutputTokens int
	skipConfirmation      bool
	onConfirmation        ConfirmationCallback
}

type WrapperConfig struct {
	Provider              ports.CostAwareAIProvider
	BudgetDaily           float64
	EstimatedOutputTokens int
	SkipConfirmation      bool
	OnConfirmation        ConfirmationCallback
}

// NewCostAwareWrapper creates a provider-agnostic wrapper
func NewCostAwareWrapper(cfg WrapperConfig) (*CostAwareWrapper, error) {
	manager, err := cost.NewManager(cfg.BudgetDaily)
	if err != nil {
		return nil, fmt.Errorf("error creating cost manager: %w", err)
	}

	cacheService, err := cache.NewCache(24 * time.Hour)
	if err != nil {
		return nil, fmt.Errorf("error creating cache: %w", err)
	}

	return &CostAwareWrapper{
		provider:              cfg.Provider,
		calculator:            cost.NewCalculator(),
		manager:               manager,
		cache:                 cacheService,
		modelSelector:         routing.NewModelSelector(),
		estimatedOutputTokens: cfg.EstimatedOutputTokens,
		skipConfirmation:      cfg.SkipConfirmation,
		onConfirmation:        cfg.OnConfirmation,
	}, nil
}

// SetSkipConfirmation allows enabling or disabling manual user confirmation.
func (w *CostAwareWrapper) SetSkipConfirmation(skip bool) {
	w.skipConfirmation = skip
}

// WrapGenerate wraps any generation function with tracking
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

	slog.Debug("checking cache for operation",
		"command", command,
		"cache_key_hash", contentHash)

	if cachedData, hit, err := w.cache.Get(contentHash); err == nil && hit {
		var cachedResp interface{}
		if err := json.Unmarshal(cachedData, &cachedResp); err == nil {
			slog.Info("cache hit",
				"command", command,
				"cache_key_hash", contentHash)

			usage := &models.TokenUsage{
				CacheHit:   true,
				CostUSD:    0,
				DurationMs: time.Since(startTime).Milliseconds(),
				Model:      originalModel,
			}
			return cachedResp, usage, nil
		}
	}

	slog.Debug("cache miss, generating new content",
		"command", command)

	var inputTokens int
	tokens, err := w.provider.CountTokens(ctx, prompt)
	if err == nil {
		inputTokens = tokens
	}

	suggestedModel := w.modelSelector.SelectBestModel(command, inputTokens)
	hasSuggestion := suggestedModel != originalModel

	estimatedCost := w.calculator.EstimateCost(providerName, originalModel, inputTokens, w.estimatedOutputTokens)

	budgetStatus, err := w.manager.CheckBudget(estimatedCost)
	if err != nil {
		return nil, nil, err
	}

	if budgetStatus.IsExceeded {
		return nil, nil, errors.ErrQuotaExceeded
	}

	if (estimatedCost > 0.0001 || hasSuggestion) && !w.skipConfirmation && w.onConfirmation != nil {
		rationaleKey := ""
		if hasSuggestion {
			rationaleKey = w.modelSelector.GetRationale(suggestedModel)
		}

		choice, proceed := w.onConfirmation(ConfirmationResult{
			EstimatedCost:  estimatedCost,
			InputTokens:    inputTokens,
			OutputTokens:   w.estimatedOutputTokens,
			SuggestedModel: suggestedModel,
			CurrentModel:   originalModel,
			RationaleKey:   rationaleKey,
		})

		if !proceed {
			return nil, nil, errors.NewAppError(errors.TypeInternal, "operation cancelled by user", nil)
		}
		if choice == "suggested" {
			modelToUse = suggestedModel
		}
	}

	resp, usage, err := generateFn(ctx, modelToUse, prompt)
	if err != nil {
		return nil, nil, err
	}

	if err := w.cache.Set(contentHash, resp); err != nil {
		slog.Warn("failed to cache response",
			"command", command,
			"error", err)
	} else {
		slog.Debug("response cached successfully",
			"command", command,
			"cache_key_hash", contentHash)
	}

	if usage != nil {
		usage.Model = modelToUse
		usage.CostUSD = w.calculator.EstimateCost(providerName, modelToUse, usage.InputTokens, usage.OutputTokens)
		usage.DurationMs = time.Since(startTime).Milliseconds()
		usage.CacheHit = false

		_ = w.manager.SaveActivity(cost.ActivityRecord{
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
