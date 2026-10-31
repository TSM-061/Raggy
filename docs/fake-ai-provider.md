# Fake AI provider

Set this in the RAG service environment (or `services/rag/.env` for the existing Compose stack):

```dotenv
AI_PROVIDER=fake
FAKE_CHUNK_DELAY=1s
```

Restart the RAG service after changing the setting. `GEMINI_API_KEY` can be empty or absent in fake mode. The default remains `AI_PROVIDER=gemini`, which requires a nonempty `GEMINI_API_KEY`. Unknown provider names fail startup rather than silently selecting another implementation.

## Behavior

- All embedding and generation calls use a local fake client. The Gemini client is never constructed in fake mode, and neither operation makes external AI HTTP requests.
- Every embedding is a fresh 1,536-element vector: the first value is 1 and the remaining values are 0. This matches the current database schema and keeps the normal cosine-distance search and persisted-chunk counting paths available.
- Generation returns a clearly labelled generic sample answer. It cites the first retrieved source, so inline annotations and source excerpts pass through the existing validation and API response path. It does not claim to answer the question semantically.
- With no retrieved context, the existing RAG service returns insufficient evidence. Upload and ingest at least one document to see the sample answer through the query API.
- Cancellation is respected by both fake operations.
- Fake chunk embeddings wait one second by default so dashboard progress is visible. Set `FAKE_CHUNK_DELAY` to another duration (for example `500ms` or `2s`), or `0s` to disable the wait. Cancellation interrupts the delay immediately. Query embeddings and sample answer generation remain immediate; Gemini operations ignore this setting. The existing RAG worker processes chunks sequentially, so a document with four chunks takes about four seconds plus storage and messaging time in fake mode.

Database writes, storage downloads, Kafka processing, chunk-completion reporting, authentication, and dashboard updates remain real. This switch replaces only the two AI operations. You still need the normal infrastructure running.

## Live interface versus browser preview

To exercise the backend from the interface, use `VITE_DEMO_MODE=false` for the frontend and `AI_PROVIDER=fake` for the RAG service. The separate `VITE_DEMO_MODE=true` frontend preview bypasses backend APIs entirely and therefore does not test ingestion or chunk completion.

For Compose, copy `services/rag/.env.example` to `services/rag/.env` if needed, set `AI_PROVIDER=fake`, and restart the service:

```sh
docker compose -f docker/dev.compose.yaml up -d --build rag-service
```

Use a separate development dataset or re-ingest documents after changing providers. Fake vectors are identical and provide no semantic ranking; switching back to Gemini does not regenerate existing stored vectors automatically. Fake mode uses the database's fixed 1,536 dimensions and ignores Gemini model and dimensionality choices.

## Verification

From `services/rag`:

```sh
go test ./...
go vet ./...
```

Tests cover provider selection, the default provider, conditional API-key validation, deterministic independently allocated embeddings, sample citations, empty context, cancellation, and a transport tripwire that detects unexpected AI HTTP requests.

For the optional real-database pipeline check, set `TEST_RAG_DATABASE_URL` to an isolated PostgreSQL 18 database with pgvector available, then run `go test -run TestFakeProviderPgvectorPipeline ./internal/ai`. It applies the real migrations in a temporary schema, persists and replays a chunk, verifies dimensions and cosine distance, checks the completed-chunk count, and searches for a validated annotated response. No real AI provider is contacted.
