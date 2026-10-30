package ai

import (
	"context"

	"github.com/TSM-061/Raggy/rag/internal/rag"
)

type MockAI struct{}

func (*MockAI) Embed(ctx context.Context, tokens string) ([]float32, error) {
	ctx, span := tracer.Start(ctx, "embeddings fake-embeding-67")
	defer span.End()
	return make([]float32, 1536), nil
}

func (*MockAI) GenerateContent(ctx context.Context, prompt string) (*rag.GeneratedAnswer, error) {
	ctx, span := tracer.Start(ctx, "fake-llm-67")
	defer span.End()
	return &rag.GeneratedAnswer{Status: rag.InsufficientEvidence, References: []string{}}, nil
}
