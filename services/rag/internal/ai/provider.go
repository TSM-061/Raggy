package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TSM-061/Raggy/rag/internal/rag"
)

const (
	ProviderGemini = "gemini"
	ProviderFake   = "fake"
)

type Client interface {
	rag.Embedder
	rag.Generator
}

// ValidateProvider checks the selected backend and its required credentials.
func ValidateProvider(provider string, cfg *GeminiConfig) error {
	switch provider {
	case ProviderFake:
		return nil
	case ProviderGemini:
		if cfg == nil || strings.TrimSpace(cfg.APIKey) == "" {
			return fmt.Errorf("GEMINI_API_KEY is required when AI_PROVIDER=gemini")
		}
		return nil
	default:
		return fmt.Errorf("AI_PROVIDER must be gemini or fake")
	}
}

type Option func(*MockAI)

func WithFakeChunkDelay(delay time.Duration) Option {
	return func(client *MockAI) { client.ChunkDelay = delay }
}

// NewClient selects the live or fake implementation for embedding and generation.
func NewClient(ctx context.Context, provider string, cfg *GeminiConfig, options ...Option) (Client, error) {
	if err := ValidateProvider(provider, cfg); err != nil {
		return nil, err
	}
	if provider == ProviderFake {
		client := &MockAI{ChunkDelay: time.Second}
		for _, option := range options {
			option(client)
		}
		if client.ChunkDelay < 0 {
			return nil, fmt.Errorf("FAKE_CHUNK_DELAY must not be negative")
		}
		return client, nil
	}
	return NewGeminiClient(ctx, cfg)
}
