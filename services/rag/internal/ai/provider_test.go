package ai

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TSM-061/Raggy/rag/internal/rag"
)

type tripwireTransport struct{ calls int }

func TestFakeChunkDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client, err := NewClient(context.Background(), ProviderFake, nil)
		if err != nil {
			t.Fatal(err)
		}
		fake := client.(*MockAI)
		start := time.Now()
		if _, err := fake.Embed(context.Background(), "query"); err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); elapsed != 0 {
			t.Fatalf("query was delayed: %v", elapsed)
		}
		if _, err := fake.EmbedChunk(context.Background(), "chunk"); err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); elapsed != time.Second {
			t.Fatalf("default chunk delay: %v", elapsed)
		}
		instant, err := NewClient(context.Background(), ProviderFake, nil, WithFakeChunkDelay(0))
		if err != nil {
			t.Fatal(err)
		}
		start = time.Now()
		if _, err := instant.(*MockAI).EmbedChunk(context.Background(), "chunk"); err != nil {
			t.Fatal(err)
		}
		if time.Since(start) != 0 {
			t.Fatal("zero delay was ignored")
		}
	})
}

func TestFakeChunkDelayCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		client := &MockAI{ChunkDelay: time.Hour}
		result := make(chan error, 1)
		start := time.Now()
		go func() { _, err := client.EmbedChunk(ctx, "chunk"); result <- err }()
		synctest.Wait()
		cancel()
		if err := <-result; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		if time.Since(start) != 0 {
			t.Fatal("cancellation waited for the chunk timer")
		}
	})
	if _, err := NewClient(context.Background(), ProviderFake, nil, WithFakeChunkDelay(-time.Second)); err == nil {
		t.Fatal("negative delay accepted")
	}
}

func (t *tripwireTransport) RoundTrip(*http.Request) (*http.Response, error) {
	t.calls++
	return nil, errors.New("unexpected AI HTTP request")
}

func TestFakeProviderNeverCallsHTTP(t *testing.T) {
	transport := &tripwireTransport{}
	original := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = original })
	client, err := NewClient(context.Background(), ProviderFake, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := client.(*MockAI); !ok {
		t.Fatalf("wrong provider: %T", client)
	}
	first, err := client.Embed(context.Background(), "a document")
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.Embed(context.Background(), "a question")
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1536 || first[0] != 1 || !reflect.DeepEqual(first, second) {
		t.Fatal("fake vectors must be compatible and deterministic")
	}
	for _, value := range first[1:] {
		if value != 0 {
			t.Fatal("unexpected vector component")
		}
	}
	first[0] = 9
	if second[0] != 1 {
		t.Fatal("embedding calls share mutable storage")
	}
	answer, err := client.GenerateContent(context.Background(), "instructions\n"+`{"question":"test","context":[{"reference":"S2","text":"sample"}]}`)
	if err != nil || answer.Status != rag.Answered || len(answer.References) != 1 || answer.References[0] != "S2" || !strings.Contains(answer.Answer, "[S2]") {
		t.Fatalf("invalid sample answer: %+v %v", answer, err)
	}
	if !strings.Contains(answer.Answer, "fake AI provider") {
		t.Fatal("sample output must identify fake mode")
	}
	if transport.calls != 0 {
		t.Fatalf("fake provider made %d network calls", transport.calls)
	}
}

func TestProviderSelection(t *testing.T) {
	if _, err := NewClient(context.Background(), ProviderGemini, &GeminiConfig{}); err == nil {
		t.Fatal("Gemini accepted a missing key")
	}
	if _, err := NewClient(context.Background(), "typo", &GeminiConfig{APIKey: "test-key"}); err == nil {
		t.Fatal("unknown provider accepted")
	}
	client, err := NewClient(context.Background(), ProviderGemini, &GeminiConfig{APIKey: "test-key"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := client.(*GeminiClient); !ok {
		t.Fatalf("wrong live provider: %T", client)
	}
}

func TestFakeGenerationAndCancellation(t *testing.T) {
	client := &MockAI{}
	answer, err := client.GenerateContent(context.Background(), `{"context":[]}`)
	if err != nil || answer.Status != rag.InsufficientEvidence || len(answer.References) != 0 {
		t.Fatalf("unexpected empty-context result: %+v %v", answer, err)
	}
	if _, err := client.GenerateContent(context.Background(), "invalid prompt"); !errors.Is(err, rag.ErrInvalidGeneration) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Embed(ctx, "test"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := client.GenerateContent(ctx, `{"context":[]}`); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
