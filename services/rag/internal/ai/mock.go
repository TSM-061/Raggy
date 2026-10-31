package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/TSM-061/Raggy/rag/internal/rag"
)

type MockAI struct{ ChunkDelay time.Duration }

// EmbedChunk simulates ingestion latency without slowing query embeddings.
func (m *MockAI) EmbedChunk(ctx context.Context, text string) ([]float32, error) {
	if m.ChunkDelay > 0 {
		timer := time.NewTimer(m.ChunkDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return m.Embed(ctx, text)
}

// FakeEmbeddingDimensions matches the existing vector(1536) database schema.
const FakeEmbeddingDimensions = 1536

func (*MockAI) Embed(ctx context.Context, tokens string) ([]float32, error) {
	ctx, span := tracer.Start(ctx, "embeddings fake")
	defer span.End()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	vector := make([]float32, FakeEmbeddingDimensions)
	// A nonzero unit vector is accepted by cosine-distance search and indexing.
	// Identical vectors deliberately provide no semantic ranking in fake mode.
	vector[0] = 1
	return vector, nil
}

func (*MockAI) GenerateContent(ctx context.Context, prompt string) (*rag.GeneratedAnswer, error) {
	ctx, span := tracer.Start(ctx, "generation fake")
	defer span.End()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// The RAG prompt ends with a single JSON line (document newlines are escaped).
	// Read its context so the canned answer cites a supplied source, not a made-up ID.
	var input struct {
		Context []rag.Source `json:"context"`
	}
	if err := json.Unmarshal([]byte(prompt[strings.LastIndex(prompt, "\n")+1:]), &input); err != nil {
		return nil, rag.ErrInvalidGeneration
	}
	if len(input.Context) == 0 {
		return &rag.GeneratedAnswer{Status: rag.InsufficientEvidence, References: []string{}}, nil
	}
	reference := input.Context[0].Reference
	if reference == "" {
		return nil, rag.ErrInvalidGeneration
	}
	return &rag.GeneratedAnswer{
		Status:     rag.Answered,
		Answer:     fmt.Sprintf("This is a sample answer from the fake AI provider. It demonstrates a response with a citation to a retrieved document section [%s]. No external AI service was called.", reference),
		References: []string{reference},
	}, nil
}
