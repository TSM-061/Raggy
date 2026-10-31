package consumer

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/TSM-061/Raggy/rag/internal/rag"
	chunkmsg "github.com/TSM-061/Raggy/shared/message/chunk"
	uploadmsg "github.com/TSM-061/Raggy/shared/message/upload"
	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestPersistedChunkPublishesProgressToKafka(t *testing.T) {
	broker := os.Getenv("TEST_KAFKA_BROKER")
	if broker == "" {
		t.Skip("set TEST_KAFKA_BROKER for broker integration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	runner, err := New(&Config{SeedBrokers: []string{broker}}, NewJobTracker(&fakeProcessor{count: 2}))
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Close()
	receiver, err := kgo.NewClient(kgo.SeedBrokers(broker), kgo.ConsumeTopics(uploadmsg.ProgressTopic), kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()), kgo.AllowAutoTopicCreation())
	if err != nil {
		t.Fatal(err)
	}
	defer receiver.Close()
	id := uuid.New()
	payload, _ := json.Marshal(chunkmsg.StreamPayload{Index: 1, Total: 3, Content: json.RawMessage(`{"text":"sample"}`), ContentString: "sample"})
	if err := runner.HandleChunkStream(ctx, chunkmsg.Message{UploadID: id.String(), Type: chunkmsg.StreamEvent, Payload: payload}); err != nil {
		t.Fatal(err)
	}
	for ctx.Err() == nil {
		fetches := receiver.PollRecords(ctx, 100)
		for _, record := range fetches.Records() {
			var event uploadmsg.ProgressMessage
			if json.Unmarshal(record.Value, &event) == nil && event.UploadID == id.String() {
				if event.ChunksProcessed != 2 || event.ChunkTotal != 3 {
					t.Fatal(event)
				}
				return
			}
		}
	}
	t.Fatal("no chunk completion event received")
}

var _ chunkProcessor = (*rag.RAG)(nil)
