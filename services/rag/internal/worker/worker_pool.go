package embedding

import (
	"context"
	"log/slog"
	"sync"

	"github.com/TSM-061/Raggy/rag/internal/rag"
	"github.com/google/uuid"
)

type EmbeddingService interface {
	Embed(ctx context.Context, tokens string) ([]float32, error)
}

type job struct {
	ChunkID uuid.UUID
	Raw     string
}

type WorkerPool struct {
	poolSize    int
	jobCapacity int
	jobs        chan job

	ragService       *services.RAGService
	embeddingService EmbeddingService
}

func NewWorkerPool(poolSize int, jobCapacity int) *WorkerPool {
	return &WorkerPool{
		poolSize: poolSize,
		jobs:     make(chan job, jobCapacity),
	}
}

func (wp *WorkerPool) Start(ctx context.Context, wg *sync.WaitGroup) {
	for id := 0; id < wp.poolSize; id++ {
		wg.Add(1)

		go func(workerID int) {
			defer wg.Done()

			logger := slog.With("worker_id", workerID)

			logger.Debug("worker started successfully")

			for {
				select {
				case <-ctx.Done():
					logger.Info("worker stopping: context canceled", "reason", ctx.Err())
					return

				case job, ok := <-wp.jobs:
					if !ok {
						logger.Info("worker stopping: jobs channel closed")
						return
					}

					jobLogger := logger.With("chunk_id", job.ChunkID)

					embedding, err := wp.embeddingService.Embed(ctx, job.Raw)
					if err != nil {
						jobLogger.Error("failed to generate embedding from external API",
							"error", err,
						)
						continue
					}

					err = wp.ragService.UpdateEmbedding(ctx, job.ChunkID, embedding)
					if err != nil {
						jobLogger.Error("failed to update embedding in RAG storage",
							"error", err,
						)
						continue
					}

					jobLogger.Debug("successfully processed and stored chunk embedding")
				}
			}
		}(id)
	}
}

func (wp *WorkerPool) AddJob(ctx context.Context, id uuid.UUID, text string) {
	select {
	case wp.jobs <- job{ChunkID: id, Raw: text}:
	case <-ctx.Done():
	}
}

func (wp *WorkerPool) Close() {
	close(wp.jobs)
}
