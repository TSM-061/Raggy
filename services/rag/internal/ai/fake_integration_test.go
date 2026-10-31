package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/TSM-061/Raggy/rag/internal/chunk"
	"github.com/TSM-061/Raggy/rag/internal/migrations"
	"github.com/TSM-061/Raggy/rag/internal/rag"
	"github.com/TSM-061/Raggy/rag/internal/upload"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestFakeProviderPgvectorPipeline(t *testing.T) {
	dsn := os.Getenv("TEST_RAG_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_RAG_DATABASE_URL for the pgvector pipeline check")
	}
	ctx := context.Background()
	root, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := root.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector"); err != nil {
		root.Close()
		t.Fatal(err)
	}
	schema := "fake_ai_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := root.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		root.Close()
		t.Fatal(err)
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		_, _ = root.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		root.Close()
	})
	files, _ := migrations.FS.ReadDir(".")
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}
		data, _ := migrations.FS.ReadFile(file.Name())
		if _, err := pool.Exec(ctx, strings.Split(string(data), "-- +goose Down")[0]); err != nil {
			t.Fatalf("migration %s: %v", file.Name(), err)
		}
	}
	transport := &tripwireTransport{}
	original := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = original })
	client, err := NewClient(ctx, ProviderFake, nil)
	if err != nil {
		t.Fatal(err)
	}
	service := rag.New(upload.NewPostgresRepo(pool), chunk.NewPostgresRepo(pool), client, client)
	id := uuid.New()
	content, _ := json.Marshal(map[string]string{"title": "Test document", "subheading": "Example section", "text": "This document contains sample text."})
	info := &rag.ChunkInformation{UploadID: id, ChunkIndex: 0, ChunkTotal: 1, Content: content, ContentString: "sample text"}
	for range 2 {
		if err := service.IngestChunk(ctx, info); err != nil {
			t.Fatal(err)
		}
	}
	count, err := service.ProcessedChunks(ctx, id)
	if err != nil || count != 1 {
		t.Fatalf("replayed chunk count=%d err=%v", count, err)
	}
	var dimensions int
	var distance float64
	if err := pool.QueryRow(ctx, "SELECT vector_dims(embedding), embedding <=> embedding FROM chunks WHERE upload_id=$1", id).Scan(&dimensions, &distance); err != nil {
		t.Fatal(err)
	}
	if dimensions != 1536 || distance != 0 {
		t.Fatalf("dimensions=%d cosine distance=%v", dimensions, distance)
	}
	result, err := service.Search(ctx, &rag.SearchQuery{Value: "Any question", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != rag.Answered || !strings.Contains(result.Response, "fake AI provider") || !strings.Contains(result.Response, "[S1]") || len(result.Sources) != 1 || result.Sources[0].UploadID != id || result.Sources[0].Text != "This document contains sample text." {
		t.Fatalf("bad pipeline response: %+v", result)
	}
	if transport.calls != 0 {
		t.Fatal("fake pipeline made HTTP calls")
	}
}
