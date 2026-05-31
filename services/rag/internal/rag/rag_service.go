package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/TSM-061/Raggy/rag/internal/chunk"
	"github.com/TSM-061/Raggy/rag/internal/upload"
	"github.com/TSM-061/Raggy/shared/logger"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/google/uuid"
)

type RAG struct {
	uploads  upload.Repo
	chunks   chunk.Repo
	embedder Embedder
}

type Embedder interface {
	Embed(ctx context.Context, tokens string) ([]float32, error)
}

func NewRAG(
	uploads upload.Repo,
	chunks chunk.Repo,
	embedder Embedder,
) *RAG {
	return &RAG{
		uploads:  uploads,
		chunks:   chunks,
		embedder: embedder,
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
		"chunk_index", info.ChunkIndex,
		"chunk_total", info.ChunkTotal,
	)

	return nil
}
