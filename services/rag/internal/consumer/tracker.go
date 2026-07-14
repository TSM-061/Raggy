package consumer

import (
	"context"
	"sync"

	"github.com/TSM-061/Raggy/rag/internal/rag"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type job struct {
	totalChunks  int
	currentCount int
	streamSpan   trace.Span
}

type tracker struct {
	// maps uploadID to a currently processing job
	activeJobs map[uuid.UUID]*job
	mutex      sync.Mutex
	rag        *rag.RAG
}

func NewJobTracker(rag *rag.RAG) *tracker {
	return &tracker{
		activeJobs: make(map[uuid.UUID]*job),
		rag:        rag,
	}
}

func (t *tracker) TrackAndProcessChunk(
	ctx context.Context, chunkInfo *rag.ChunkInformation) error {
	t.mutex.Lock()

	j, exists := t.activeJobs[chunkInfo.UploadID]

	if !exists {
		_, streamSpan := tracer.Start(
			ctx,
			"Chunk.Stream",
		)
		streamSpan.SetAttributes(
			attribute.Int("upload.total_chunks", chunkInfo.ChunkTotal),
		)

		j = &job{
			streamSpan:  streamSpan,
			totalChunks: chunkInfo.ChunkTotal,
		}

		t.activeJobs[chunkInfo.UploadID] = j
	}

	j.currentCount++
	isLastChunk := j.currentCount == j.totalChunks
	t.mutex.Unlock()

	chunkCtx, chunkSpan := tracer.Start(
		trace.ContextWithSpan(ctx, j.streamSpan),
		"Chunk.Embed",
	)
	chunkSpan.SetAttributes(attribute.Int("chunk.index", chunkInfo.ChunkIndex))

	err := t.rag.IngestChunk(chunkCtx, chunkInfo)

	chunkSpan.End()

	if isLastChunk {
		t.mutex.Lock()

		j.streamSpan.End()
		delete(t.activeJobs, chunkInfo.UploadID)

		t.mutex.Unlock()
	}

	return err
}
