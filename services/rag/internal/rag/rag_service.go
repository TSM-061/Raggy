package rag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/TSM-061/Raggy/rag/internal/chunk"
	"github.com/TSM-061/Raggy/rag/internal/upload"
	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/google/uuid"
)

type Embedder interface {
	Embed(ctx context.Context, tokens string) ([]float32, error)
}

// ChunkEmbedder optionally customizes ingestion while retaining fast query embeddings.
type ChunkEmbedder interface {
	EmbedChunk(context.Context, string) ([]float32, error)
}

type Generator interface {
	GenerateContent(ctx context.Context, prompt string) (*GeneratedAnswer, error)
}

type RAG struct {
	uploads   upload.Repo
	chunks    chunk.Repo
	embedder  Embedder
	generator Generator
}

func New(
	uploads upload.Repo,
	chunks chunk.Repo,
	embedder Embedder,
	generator Generator,
) *RAG {
	return &RAG{
		uploads:   uploads,
		chunks:    chunks,
		embedder:  embedder,
		generator: generator,
	}
}

type ChunkInformation struct {
	UploadID   uuid.UUID
	ChunkIndex int
	ChunkTotal int

	Content       json.RawMessage
	ContentString string
}

func (r *RAG) IngestChunk(ctx context.Context, info *ChunkInformation) error {

	log := logger.FromContext(ctx)

	if info == nil || info.UploadID == uuid.Nil {
		return fmt.Errorf("%w: missing chunk information", serviceerr.InvalidInput)
	}
	if info.ChunkIndex < 0 || info.ChunkIndex >= info.ChunkTotal {
		return fmt.Errorf(
			"%w: chunk index %d out of bounds, range [0,%d)",
			serviceerr.InvalidInput,
			info.ChunkIndex,
			info.ChunkTotal,
		)
	}

	if err := r.uploads.Upsert(ctx, &upload.Upload{
		ID:         info.UploadID,
		ChunkTotal: info.ChunkTotal,
	}); err != nil {
		return fmt.Errorf("upsert upload metadata: %w", err)
	}

	embed := r.embedder.Embed
	if chunkEmbedder, ok := r.embedder.(ChunkEmbedder); ok {
		embed = chunkEmbedder.EmbedChunk
	}
	embedding, err := embed(ctx, info.ContentString)
	if err != nil {
		return fmt.Errorf("calculate chunk embedding: %w", err)
	}

	chunk := chunk.Chunk{
		UploadID:   info.UploadID,
		ChunkIndex: info.ChunkIndex,
		Content:    info.Content,
		Embedding:  embedding,
	}

	if err := r.chunks.Create(ctx, &chunk); err != nil {
		return fmt.Errorf("persist chunk embedding: %w", err)
	}

	log.InfoContext(
		ctx,
		"chunk embedding successfully persisted",
		slog.String("chunk_id", chunk.ID.String()),
		slog.Int("chunk_index", info.ChunkIndex),
	)

	return nil
}

func (r *RAG) ProcessedChunks(ctx context.Context, uploadID uuid.UUID) (int, error) {
	return r.chunks.CountByUploadID(ctx, uploadID)
}

type SearchQuery struct {
	Value string
	Limit int
}

func (r *RAG) Search(ctx context.Context, query *SearchQuery) (*SearchResult, error) {
	log := logger.FromContext(ctx)

	limitMax := 5

	if query == nil || strings.TrimSpace(query.Value) == "" {
		return nil, fmt.Errorf("%w: query value can't be empty", serviceerr.InvalidInput)
	}

	if query.Limit < 1 || query.Limit > limitMax {
		return nil, fmt.Errorf(
			"%w: query limit %d out of bounds, range [1,%d)",
			serviceerr.InvalidInput,
			query.Limit,
			limitMax,
		)
	}

	embedding, err := r.embedder.Embed(ctx, query.Value)
	if err != nil {
		log.ErrorContext(ctx, "rag search failed", slog.String("category", "embedding"))
		return nil, fmt.Errorf("calculate query embedding: %w", err)
	}

	chunks, err := r.chunks.Search(ctx, embedding, query.Limit)
	if err != nil {
		log.ErrorContext(ctx, "rag search failed", slog.String("category", "retrieval"))
		return nil, fmt.Errorf("search chunk vectors: %w", err)
	}

	if len(chunks) == 0 {
		return insufficientResult(), nil
	}
	prompt, sources, err := buildPrompt(query.Value, chunks)
	if err != nil {
		log.ErrorContext(ctx, "rag search failed", slog.String("category", "persisted_content"))
		return nil, err
	}
	answer, err := r.generator.GenerateContent(ctx, prompt)
	if err != nil {
		category := "provider"
		if errors.Is(err, ErrInvalidGeneration) {
			category = "generation_format"
		}
		log.ErrorContext(ctx, "rag generation failed", slog.String("category", category))
		return nil, fmt.Errorf("generate answer: %w", err)
	}
	result, err := resolveAnswer(answer, sources)
	if err != nil {
		log.ErrorContext(ctx, "rag generation failed", slog.String("category", "invalid_references"))
		return nil, err
	}
	return result, nil
}
