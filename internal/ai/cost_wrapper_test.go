package ai

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/services/cost"
	"github.com/stretchr/testify/mock"
)

type mockProvider struct {
	mock.Mock
}

func (m *mockProvider) GenerateCommitMessage(ctx context.Context, diff string) (string, error) {
	args := m.Called(ctx, diff)
	return args.String(0), args.Error(1)
}

func (m *mockProvider) GenerateReleaseNotes(ctx context.Context, tag string, commits []models.Commit) (string, error) {
	args := m.Called(ctx, tag, commits)
	return args.String(0), args.Error(1)
}

func (m *mockProvider) GeneratePullRequestDescription(ctx context.Context, pr models.PullRequest) (string, error) {
	args := m.Called(ctx, pr)
	return args.String(0), args.Error(1)
}

func (m *mockProvider) GetProviderName() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockProvider) GetModelName() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockProvider) CountTokens(ctx context.Context, text string) (int, error) {
	args := m.Called(ctx, text)
	return args.Int(0), args.Error(1)
}

func setupTestWrapper(t *testing.T, budget float64) (*CostAwareWrapper, *mockProvider, string) {
	tempHome, err := os.MkdirTemp("", "matecommit-home-*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}

	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tempHome)
	t.Cleanup(func() {
		_ = os.Setenv("HOME", oldHome)
		if err := os.RemoveAll(tempHome); err != nil {
			t.Errorf("failed to remove temp home: %v", err)
		}
	})

	mockP := new(mockProvider)

	cfg := WrapperConfig{
		Provider:              mockP,
		BudgetDaily:           budget,
		EstimatedOutputTokens: 200,
		SkipConfirmation:      true,
		OnConfirmation:        nil,
	}

	wrapper, err := NewCostAwareWrapper(cfg)
	if err != nil {
		t.Fatalf("NewCostAwareWrapper() error = %v", err)
	}

	return wrapper, mockP, tempHome
}

func TestNewCostAwareWrapper(t *testing.T) {
	w, _, _ := setupTestWrapper(t, 1.0)
	if w == nil {
		t.Fatal("expected wrapper to be non-nil")
	}
}

func TestCostAwareWrapper_WrapGenerate_CacheHit(t *testing.T) {
	// Arrange
	w, mockP, _ := setupTestWrapper(t, 1.0)
	ctx := context.Background()
	prompt := "test prompt"
	command := "test-cmd"
	expectedResp := "cached response"

	mockP.On("GetProviderName").Return("gemini")
	mockP.On("GetModelName").Return("gemini-1.5-flash")

	contentHash := w.cache.GenerateHash("gemini" + "gemini-1.5-flash" + prompt)
	_ = w.cache.Set(contentHash, expectedResp)

	// Act
	resp, usage, err := w.WrapGenerate(ctx, command, prompt, func(ctx context.Context, model, p string) (interface{}, *models.TokenUsage, error) {
		t.Fatal("generateFn should not be called on cache hit")
		return nil, nil, nil
	})

	// Assert
	if err != nil {
		t.Fatalf("WrapGenerate() error = %v", err)
	}
	if !usage.CacheHit {
		t.Error("expected CacheHit to be true")
	}
	if resp.(string) != expectedResp {
		t.Errorf("expected resp %q, got %v", expectedResp, resp)
	}
	mockP.AssertExpectations(t)
}

func TestCostAwareWrapper_WrapGenerate_NormalFlow(t *testing.T) {
	// Arrange
	w, mockP, _ := setupTestWrapper(t, 1.0)
	ctx := context.Background()
	prompt := "test prompt"
	command := "test-cmd"
	expectedResp := "fresh response"
	expectedUsage := &models.TokenUsage{InputTokens: 50, OutputTokens: 100}

	mockP.On("GetProviderName").Return("gemini")
	mockP.On("GetModelName").Return("gemini-1.5-flash")
	mockP.On("CountTokens", mock.Anything, mock.Anything).Return(100, nil)

	// Act
	resp, usage, err := w.WrapGenerate(ctx, command, prompt, func(ctx context.Context, model, p string) (interface{}, *models.TokenUsage, error) {
		return expectedResp, expectedUsage, nil
	})

	// Assert
	if err != nil {
		t.Fatalf("WrapGenerate() error = %v", err)
	}
	if usage.CacheHit {
		t.Error("expected CacheHit to be false")
	}
	if resp.(string) != expectedResp {
		t.Errorf("expected resp %q, got %v", expectedResp, resp)
	}
	if usage.InputTokens != 50 || usage.OutputTokens != 100 {
		t.Errorf("unexpected usage: %+v", usage)
	}

	contentHash := w.cache.GenerateHash("gemini" + "gemini-1.5-flash" + prompt)
	_, hit, _ := w.cache.Get(contentHash)
	if !hit {
		t.Error("expected response to be cached")
	}
	mockP.AssertExpectations(t)
}

func TestCostAwareWrapper_WrapGenerate_BudgetExceeded(t *testing.T) {
	// Arrange
	w, mockP, _ := setupTestWrapper(t, 0.0001)
	ctx := context.Background()

	mockP.On("GetProviderName").Return("gemini")
	mockP.On("GetModelName").Return("gemini-1.5-flash")
	mockP.On("CountTokens", mock.Anything, mock.Anything).Return(100, nil)

	_ = w.manager.SaveActivity(cost.ActivityRecord{
		Timestamp: time.Now(),
		CostUSD:   1.0,
	})

	// Act
	_, _, err := w.WrapGenerate(ctx, "cmd", "prompt", func(ctx context.Context, model, p string) (interface{}, *models.TokenUsage, error) {
		return "should not run", nil, nil
	})

	// Assert
	if err == nil {
		t.Error("expected error due to budget exceeded, got nil")
	}
	mockP.AssertExpectations(t)
}

// TestCostAwareWrapper_AskUserConfirmation fue removido porque askUserConfirmation
// ya no es un método del wrapper - ahora se pasa como callback (onConfirmation)
// desde la capa CLI. El comportamiento de confirmación ahora se testea en los tests de CLI.

func TestCostAwareWrapper_WrapGenerate_SuggestedModel(t *testing.T) {
	// Arrange
	budget := 1.0
	tempHome, err := os.MkdirTemp("", "matecommit-home-*")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tempHome)
	t.Cleanup(func() {
		_ = os.Setenv("HOME", oldHome)
		_ = os.RemoveAll(tempHome)
	})

	mockP := new(mockProvider)
	cfg := WrapperConfig{
		Provider:              mockP,
		BudgetDaily:           budget,
		EstimatedOutputTokens: 200,
		SkipConfirmation:      false,
		OnConfirmation: func(result ConfirmationResult) (string, bool) {
			return "suggested", true
		},
	}

	w, err := NewCostAwareWrapper(cfg)
	if err != nil {
		t.Fatalf("NewCostAwareWrapper() error = %v", err)
	}

	ctx := context.Background()
	prompt := "large prompt"

	mockP.On("GetProviderName").Return("gemini")
	mockP.On("GetModelName").Return("gemini-1.5-flash")
	mockP.On("CountTokens", mock.Anything, mock.Anything).Return(20000, nil)

	// Act
	var usedModel string
	_, usage, err := w.WrapGenerate(ctx, "summarize", prompt, func(ctx context.Context, model, p string) (interface{}, *models.TokenUsage, error) {
		usedModel = model
		return "ok", &models.TokenUsage{InputTokens: 20000, OutputTokens: 200}, nil
	})

	// Assert
	if err != nil {
		t.Fatalf("WrapGenerate() error = %v", err)
	}
	expectedModel := "gemini-3-flash-preview"
	if usedModel != expectedModel {
		t.Errorf("expected suggested model %q to be used, got %q", expectedModel, usedModel)
	}
	if usage.Model != expectedModel {
		t.Errorf("expected usage model %q, got %q", expectedModel, usage.Model)
	}
	mockP.AssertExpectations(t)
}
