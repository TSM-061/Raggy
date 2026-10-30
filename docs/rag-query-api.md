# RAG query API

`GET /api/search?q=<question>&limit=5` uses the existing authentication middleware. Supply a valid session as for the other authenticated APIs. `q` must contain non-whitespace text; `limit` is an integer from 1 to 5 and defaults to 5.

## Answered response (HTTP 200)

```json
{
  "status": "answered",
  "response": "The wharf deck supports cranes up to 200 tonnes [S1].",
  "sources": [
    {
      "reference": "S1",
      "chunk_id": "c973800b-c5f7-46d5-a1d3-d7fa2a6a07be",
      "upload_id": "fa08fa32-da25-4d8b-9714-960680bfe963",
      "chunk_index": 1,
      "title": "Heavy-Lift Material Wharf Expansion",
      "section": "Design Requirements",
      "text": "Support heavy track-mounted mobile cranes (up to 200-tonne operating weight) on the wharf deck."
    }
  ]
}
```

This is an illustrative response, not a recorded provider result. Supporting text is the full persisted section body. Optional title and section fields are omitted when absent. Source IDs and content come from storage; the model chooses only reference labels. Labels are request-local, and sources are deduplicated and returned in context order. Existing clients can keep reading `response`; clients with strict JSON schemas must allow the new fields.

## Insufficient evidence (HTTP 200)

```json
{
  "status": "insufficient_evidence",
  "response": "The available reports do not contain enough evidence to answer this question.",
  "sources": []
}
```

Empty retrieval bypasses generation. Nonempty retrieval can also yield this response when the model judges the evidence inadequate. Database failures, provider errors, malformed output, and invalid citations return HTTP 500 with the generic body `search failed`; they are not evidence decisions. Invalid inputs return HTTP 400. Authentication behavior is unchanged.

## Generation and validation

The generation prompt includes the original question and JSON-encoded context. Gemini is requested to return structured JSON with status, answer, and reference labels. Answered results require nonempty text and at least one reference, with the same reference set appearing inline as `[S1]`. Grouped citations such as `[S1, S2]` are rejected; use `[S1] [S2]`. Square-bracket expressions in an answer are reserved for citations. Unknown fields, malformed output, unknown references, and contradictory abstention fields are rejected without regeneration retries.

Citation validation proves that labels identify supplied chunks. It does not prove that every claim is supported or that abstention is always correct. This version does not add a similarity cutoff or change document access rules.

## Running and checking

The service now uses the configured Gemini client rather than the former hardcoded mock. Set `GEMINI_API_KEY` in `services/rag/.env` when using the existing development Compose setup. Model names and dimensions remain configurable through the existing `GEMINI_` variables. Re-ingest documents previously embedded with mock vectors before evaluating real retrieval; query and stored embeddings must use the same model and dimensions.

Automated checks from `services/rag`:

```sh
go test ./...
go vet ./...
```

The adapter tests use a local HTTP server and need no provider credentials. Service tests use fake repositories and generators, so no database or Kafka is required.

Manual provider check, once the development stack and credentials are available:

1. Sign in through the auth API and retain the session cookies. Create an upload through the dashboard API, then PUT `examples/portland-wharf.md` to the returned signed URL using its signed headers and the project-report Markdown profile.
2. Wait for ingestion and embedding to complete; confirm chunks exist in the RAG database.
3. Query `/api/search?q=What%20crane%20operating%20weight%20must%20the%20wharf%20support%3F&limit=5` with the session. Expect an answered result describing up to 200 tonnes and citations matching retrieved stored sections.
4. Query `/api/search?q=What%20is%20the%20project%20manager%27s%20personal%20phone%20number%3F&limit=5`. Expect insufficient evidence with no sources.
5. Compare each returned chunk ID, upload ID, title, section, and text against the database. Record model configuration and outcomes, excluding credentials.

Live provider verification has not yet been recorded. The automated checks validate contracts and error handling; the manual check assesses real retrieval and generation behavior.
