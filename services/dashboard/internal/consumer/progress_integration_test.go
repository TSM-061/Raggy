package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TSM-061/Raggy/dashboard/internal/migrations"
	"github.com/TSM-061/Raggy/dashboard/internal/progress"
	"github.com/TSM-061/Raggy/dashboard/internal/services"
	"github.com/TSM-061/Raggy/dashboard/internal/upload"
	uploadmsg "github.com/TSM-061/Raggy/shared/message/upload"
	"github.com/TSM-061/Raggy/shared/serviceerr"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
)

func progressDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DASHBOARD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set DASHBOARD_TEST_DATABASE_URL for PostgreSQL integration checks")
	}
	ctx := context.Background()
	root, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "progress_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = root.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		root.Close()
		t.Fatal(err)
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
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
		up := strings.Split(string(data), "-- +goose Down")[0]
		if _, err := pool.Exec(ctx, up); err != nil {
			t.Fatalf("migration %s: %v", file.Name(), err)
		}
	}
	return pool
}

func TestProgressPersistsMonotonicallyAndNotifies(t *testing.T) {
	pool := progressDatabase(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repo := upload.NewPostgresRepo(pool)
	item := &upload.Upload{UploadedBy: uuid.New(), OriginalName: "document.md", ContentType: "text/markdown", ProfileHint: "profile-hint.project-report-md", SizeBytes: 100, Status: upload.StatusUploaded}
	if err := repo.Create(ctx, item); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, item.ID)
	if err != nil || got.ChunkTotal != nil {
		t.Fatalf("legacy progress should be unknown: %+v %v", got, err)
	}
	hub := progress.NewHub()
	changes, unsubscribe := hub.Subscribe()
	defer unsubscribe()
	done := make(chan struct{})
	go func() { hub.Run(ctx, pool.Config().ConnConfig); close(done) }()
	defer func() { cancel(); <-done }()
	select {
	case <-changes:
	case <-time.After(5 * time.Second):
		t.Fatal("listener did not connect")
	}
	runner := &Runner{uploads: services.NewUploadService(repo, nil, time.Minute, validator.New())}
	for _, count := range []int{0, 2, 1, 2, 3, 0} {
		data, _ := json.Marshal(uploadmsg.ProgressMessage{Type: uploadmsg.Progress, UploadID: item.ID.String(), ChunkTotal: 3, ChunksProcessed: count})
		if err := runner.processRecord(ctx, &kgo.Record{Topic: uploadmsg.ProgressTopic, Value: data}); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-changes:
	case <-time.After(5 * time.Second):
		t.Fatal("no post-commit progress notification")
	}
	if err := repo.UpdateStatus(ctx, item.ID, upload.StatusUploaded); err != nil {
		t.Fatal(err)
	}
	got, err = repo.GetByID(ctx, item.ID)
	if err != nil || got.ChunksProcessed != 3 || got.ChunkTotal == nil || *got.ChunkTotal != 3 || got.Status != upload.StatusProcessed {
		t.Fatalf("bad final progress: %+v %v", got, err)
	}
	if err := repo.ApplyProgress(ctx, item.ID, 4, 1); !errors.Is(err, serviceerr.InvalidInput) {
		t.Fatalf("total changed: %v", err)
	}
	if err := repo.ApplyProgress(ctx, item.ID, 3, 4); !errors.Is(err, serviceerr.InvalidInput) {
		t.Fatalf("invalid counts accepted: %v", err)
	}
	list, _, err := repo.List(ctx, 20, 0)
	if err != nil || len(list) != 1 || list[0].ChunksProcessed != 3 {
		t.Fatalf("progress not in list: %+v %v", list, err)
	}
}

func TestConcurrentProgressDoesNotLoseCompletion(t *testing.T) {
	pool := progressDatabase(t)
	ctx := context.Background()
	repo := upload.NewPostgresRepo(pool)
	item := &upload.Upload{UploadedBy: uuid.New(), OriginalName: "document.md", ContentType: "text/markdown", ProfileHint: "md", SizeBytes: 1, Status: upload.StatusUploaded}
	if err := repo.Create(ctx, item); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 21; i++ {
		wg.Add(1)
		go func(count int) {
			defer wg.Done()
			if err := repo.ApplyProgress(ctx, item.ID, 6, count); err != nil {
				t.Error(err)
			}
		}(i % 7)
	}
	wg.Wait()
	got, err := repo.GetByID(ctx, item.ID)
	if err != nil || got.ChunksProcessed != 6 || got.Status != upload.StatusProcessed {
		t.Fatalf("lost progress: %+v %v", got, err)
	}
	empty := &upload.Upload{UploadedBy: uuid.New(), OriginalName: "empty.md", ContentType: "text/markdown", ProfileHint: "md", SizeBytes: 1, Status: upload.StatusUploaded}
	if err := repo.Create(ctx, empty); err != nil {
		t.Fatal(err)
	}
	if err := repo.ApplyProgress(ctx, empty.ID, 0, 0); err != nil {
		t.Fatal(err)
	}
	got, _ = repo.GetByID(ctx, empty.ID)
	if got.Status != upload.StatusProcessed || got.ChunkTotal == nil || *got.ChunkTotal != 0 {
		t.Fatal(got)
	}
}

func TestKafkaProgressReachesDashboard(t *testing.T) {
	broker := os.Getenv("TEST_KAFKA_BROKER")
	if broker == "" {
		t.Skip("set TEST_KAFKA_BROKER for broker integration")
	}
	pool := progressDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	repo := upload.NewPostgresRepo(pool)
	item := &upload.Upload{UploadedBy: uuid.New(), OriginalName: "broker.md", ContentType: "text/markdown", ProfileHint: "md", SizeBytes: 1, Status: upload.StatusUploaded}
	if err := repo.Create(ctx, item); err != nil {
		t.Fatal(err)
	}
	runner, err := New(&Config{SeedBrokers: []string{broker}, MaxPollRecords: 10}, services.NewUploadService(repo, nil, time.Minute, validator.New()))
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { runner.Start(ctx); close(done) }()
	defer func() { cancel(); <-done; runner.Close() }()
	producer, err := kgo.NewClient(kgo.SeedBrokers(broker), kgo.AllowAutoTopicCreation())
	if err != nil {
		t.Fatal(err)
	}
	defer producer.Close()
	for _, count := range []int{0, 2, 1, 2, 3, 0} {
		data, _ := json.Marshal(uploadmsg.ProgressMessage{Type: uploadmsg.Progress, UploadID: item.ID.String(), ChunkTotal: 3, ChunksProcessed: count})
		if err := producer.ProduceSync(ctx, &kgo.Record{Topic: uploadmsg.ProgressTopic, Key: []byte(item.ID.String()), Value: data}).FirstErr(); err != nil {
			t.Fatal(err)
		}
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Kafka progress did not reach database")
		case <-ticker.C:
			got, err := repo.GetByID(ctx, item.ID)
			if err == nil && got.ChunksProcessed == 3 && got.Status == upload.StatusProcessed {
				return
			}
		}
	}
}
