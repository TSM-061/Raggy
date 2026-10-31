# Chunk completion and live dashboard progress

## Flow

1. Ingestion publishes a cumulative `upload.progress` snapshot with the expected chunk total and zero completed chunks to Kafka topic `upload-progress`, then publishes the document chunks.
2. RAG persists each successful embedding. Only after persistence succeeds does it count the document's stored embeddings and publish a new progress snapshot.
3. The dashboard consumes progress events and atomically updates `uploads.chunk_total`, `uploads.chunks_processed`, and status. PostgreSQL sends a notification after commit.
4. Each dashboard instance listens for those notifications and sends an invalidation to authenticated browsers through `GET /api/uploads/events`. The browser refreshes its current REST snapshot.

Example event (keyed by upload ID):

```json
{
  "type": "upload.progress",
  "uploadId": "<upload UUID>",
  "chunkTotal": 6,
  "chunksProcessed": 4
}
```

`GET /api/uploads` now includes `chunkTotal` (nullable when unknown) and `chunksProcessed` for every upload. Status transitions from `uploaded` to `processing` and finally `processed`; the interface labels the final state **Ready**. A known total of zero is represented as 0/0, distinct from an unknown total.

## Delivery and consistency

Events contain cumulative database counts, not in-memory increments. The dashboard uses the greater of its persisted count and the incoming count. Duplicate and out-of-order snapshots therefore cannot lower progress or double-count a chunk. A late S3 upload notification cannot move a processing or completed upload back to `uploaded`.

The chunk table already has a unique `(upload_id, chunk_index)` constraint. Persistence now upserts that key so replay does not insert a second chunk. Consumers manually commit offsets only after processing, including downstream event publication, succeeds. Transient failures retry until shutdown; a crash after persistence but before publication replays the uncommitted record. Commit retries do not repeat processing. Invalid input is explicitly rejected and logged rather than blocking a partition indefinitely. A permanently failing provider/database request still requires operator attention and can hold up its consumer; this change does not add a dead-letter queue or failure-state UI.

An upload ID identifies immutable input and a fixed chunk total. Re-ingestion with different content should use a new upload ID. Conflicting totals and invalid counts are rejected. Deleted uploads are ignored by the progress consumer. Previously ingested documents retain unknown counts until their processing events are replayed; there is no automatic historical backfill.

## Browser synchronization

The endpoint uses Server-Sent Events because updates only flow from server to browser. It uses existing session-cookie authentication, sends `uploads.changed` events with `{}` as data, and never includes document data in the stream. REST remains the source of truth.

The stream sends an initial reconciliation event, heartbeats every 15 seconds, and closes after five minutes to require a fresh authentication check. The client reconnects with bounded backoff and refreshes the REST snapshot, which also handles token refresh. Bursts of events are coalesced over 200 ms. The interface polls every 30 seconds while connected, or every five seconds without the stream, to recover from missed notifications. Native event streaming is disabled in the sample-data preview.

Each dashboard instance uses one dedicated PostgreSQL LISTEN connection, in addition to its ordinary pool. This enables updates across replicas sharing the same database. Listener reconnection invalidates clients to reconcile missed changes. The nginx endpoint disables buffering and has a 60-second read timeout; heartbeats keep active connections alive.

## Rollout

1. Apply the new dashboard migration, `20261031000100_chunk_progress.sql`, using the existing Goose migration process.
2. Create the `upload-progress` topic. The existing Kafka initialization script now includes it; rerun that initialization for an existing stack.
3. Deploy the dashboard subscriber and streaming endpoint, then the ingestion and RAG producers, then the frontend. Old frontend clients tolerate the additional REST fields.

No RAG database migration or additional AI configuration is required. Roll back producers before removing dashboard columns. The existing migration runner in Compose mounts the migration directory and applies the new file on its next run.

## Validation

- Shared tests cover event bounds, processing retries, commit retries, shutdown, and invalid input.
- RAG tests verify that failed persistence cannot produce completion snapshots. Broker integration verifies real Kafka publication using a fake embedding processor; it does not call an AI provider.
- Dashboard integration tests use PostgreSQL 18 and actual migrations to verify duplicate/out-of-order delivery, concurrent updates, fixed totals, API repository reads, zero chunks, and notifications. A broker test verifies Kafka events reaching the database.
- HTTP tests verify the SSE headers, initial reconciliation, and notification forwarding.
- Frontend tests verify progress rendering, event-driven refresh, reconnection, and coalescing. TypeScript and production builds cover integration with the existing workspace.

Integration tests are opt-in. Set `DASHBOARD_TEST_DATABASE_URL` for isolated PostgreSQL tests and `TEST_KAFKA_BROKER` for broker tests. Database tests create a unique temporary schema and remove it afterward. Run broker tests against a disposable broker: they use the application topic names and dashboard consumer group.

```sh
cd services/dashboard
go test -timeout 90s ./...
go vet ./...

cd ../rag
go test -timeout 60s ./...
go vet ./...

cd ../ingestion-worker
go test ./...
go vet ./...

cd ../../frontend
npm run typecheck
npm test
npm run build
```
