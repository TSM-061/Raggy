package consumer

import (
	"context"
	"errors"
	"github.com/TSM-061/Raggy/rag/internal/rag"
	"github.com/google/uuid"
	"testing"
)

type fakeProcessor struct {
	err      error
	countErr error
	count    int
	counted  bool
}

func (f *fakeProcessor) IngestChunk(context.Context, *rag.ChunkInformation) error { return f.err }
func (f *fakeProcessor) ProcessedChunks(context.Context, uuid.UUID) (int, error) {
	f.counted = true
	return f.count, f.countErr
}
func TestTrackerCountsOnlyPersistedChunks(t *testing.T) {
	for _, tc := range []struct {
		name      string
		processor fakeProcessor
		success   bool
	}{
		{"persisted", fakeProcessor{count: 2}, true},
		{"embedding failure", fakeProcessor{err: errors.New("embedding failed")}, false},
		{"count failure", fakeProcessor{countErr: errors.New("database failed")}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := tc.processor
			id := uuid.New()
			event, err := NewJobTracker(&p).TrackAndProcessChunk(context.Background(), &rag.ChunkInformation{UploadID: id, ChunkTotal: 3, ChunkIndex: 1})
			if (err == nil) != tc.success {
				t.Fatalf("event=%+v err=%v", event, err)
			}
			if tc.success && (event.ChunksProcessed != 2 || event.ChunkTotal != 3 || event.UploadID != id.String()) {
				t.Fatal(event)
			}
			if p.err != nil && p.counted {
				t.Fatal("failed ingestion was counted")
			}
		})
	}
}
