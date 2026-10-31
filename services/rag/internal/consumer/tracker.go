package consumer

import (
	"context"
	"github.com/TSM-061/Raggy/rag/internal/rag"
	"github.com/TSM-061/Raggy/shared/message/upload"
	"github.com/google/uuid"
)

type chunkProcessor interface {
	IngestChunk(context.Context, *rag.ChunkInformation) error
	ProcessedChunks(context.Context, uuid.UUID) (int, error)
}

type tracker struct{ rag chunkProcessor }

func NewJobTracker(service chunkProcessor) *tracker { return &tracker{rag: service} }

// TrackAndProcessChunk ingests a chunk and builds progress from persisted embeddings.
func (t *tracker) TrackAndProcessChunk(ctx context.Context, info *rag.ChunkInformation) (*upload.ProgressMessage, error) {
	ctx, span := tracer.Start(ctx, "Chunk.Embed")
	defer span.End()
	if err := t.rag.IngestChunk(ctx, info); err != nil {
		return nil, err
	}
	count, err := t.rag.ProcessedChunks(ctx, info.UploadID)
	if err != nil {
		return nil, err
	}
	return &upload.ProgressMessage{Type: upload.Progress, UploadID: info.UploadID.String(), ChunkTotal: info.ChunkTotal, ChunksProcessed: count}, nil
}
