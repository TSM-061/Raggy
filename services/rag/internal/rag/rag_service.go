package rag

import (
	"context"
	"encoding/json"
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

type Generator interface {
	GenerateContent(ctx context.Context, prompt string) (string, error)
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

	embedding, err := r.embedder.Embed(ctx, info.ContentString)
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
		"Chunk embedding successfully persisted",
		slog.String("chunk_id", chunk.ID.String()),
		slog.Int("chunk_index", info.ChunkIndex),
		slog.Int("chunk_total", info.ChunkTotal),
	)

	return nil
}

type SearchQuery struct {
	Value string
	Limit int
}

func (r *RAG) Search(ctx context.Context, query *SearchQuery) (string, error) {
	log := logger.FromContext(ctx)

	limitMax := 5

	if query.Value == "" {
		return "", fmt.Errorf("%w: query value can't be empty", serviceerr.InvalidInput)
	}

	if query.Limit < 1 || query.Limit > limitMax {
		return "", fmt.Errorf(
			"%w: query limit %d out of bounds, range [1,%d)",
			serviceerr.InvalidInput,
			query.Limit,
			limitMax,
		)
	}

	embedding, err := r.embedder.Embed(ctx, query.Value)
	if err != nil {
		return "", fmt.Errorf("calculate query embedding: %w", err)
	}

	chunks, err := r.chunks.Search(ctx, embedding, query.Limit)
	if err != nil {
		return "", fmt.Errorf("search chunk vectors: %w", err)
	}

	var promptBuilder strings.Builder

	promptBuilder.WriteString("<context>\n")
	for _, c := range chunks {
		promptBuilder.WriteString("<context_item>")
		promptBuilder.Write(c.Content)
		promptBuilder.WriteString("</context_item>\n")
	}
	promptBuilder.WriteString("</context>\n")

	response, err := r.generator.GenerateContent(ctx, promptBuilder.String())
	if err != nil {
		return "", fmt.Errorf("generate prompt response: %w", err)
	}

	log.InfoContext(ctx,
		"search query made",
		slog.Any("query", query),
	)

	return response, nil
}
