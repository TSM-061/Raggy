# RAG answers with source citations

Status: Implemented; live provider verification pending  
Date: 30 October 2026

## Objective

Extend the existing RAG query API to return grounded answers with inspectable source references and an explicit insufficient-evidence result. Frontend work is out of scope.

The current implementation retrieves chunks using the question embedding, but sends only the retrieved context to the generation model. The query API returns a single response string. This feature will include the question in the generation prompt and expose the evidence supporting the answer.

## User story

As an API consumer querying uploaded project reports, I want answers that identify their supporting document sections so that I can verify an answer against the original material and recognize when the reports do not contain enough information.

As a developer maintaining Raggy, I want predictable response states and validated source references so that I can diagnose retrieval and generation problems through automated tests and API responses.

## Scope

### Included

- Correct the generation prompt to include the original question.
- Assign request-local reference labels to retrieved chunks.
- Generate an answer with structured citation references.
- Return source metadata and supporting text from persisted chunks.
- Validate generated references against the supplied context.
- Return a distinct insufficient-evidence result for empty retrieval or a model assessment that the context cannot support an answer.
- Preserve existing query parameters, authentication, and input limits.
- Add focused automated tests and API documentation.

### Excluded

- Frontend changes, conversation history, and streaming responses.
- New document formats, chunking strategies, or embedding models.
- Hybrid search, reranking, or a calibrated similarity threshold.
- Upload lifecycle changes, ingestion retries, and document deletion propagation.
- New authorization or tenant-isolation behavior. Existing data-access rules remain applicable; this feature does not establish that they are sufficient for a multi-tenant deployment.
- Automated proof that an answer is factually correct. Reference validation establishes that cited chunks were supplied, not that they entail every generated claim.

## API behavior

Keep the existing query route and its `q` and `limit` parameters. Preserve `response` as the answer text, and add an explicit status and sources array. Consumers that ignore additional JSON fields can continue reading `response`; consumers that validate an exact schema will need to update.

Illustrative successful response:

```json
{
  "status": "answered",
  "response": "The report identifies corrosion as the primary concern [S1].",
  "sources": [
    {
      "reference": "S1",
      "chunk_id": "<persisted chunk UUID>",
      "upload_id": "<persisted upload UUID>",
      "chunk_index": 2,
      "title": "Example report",
      "section": "Condition assessment",
      "text": "<supporting section text from storage>"
    }
  ]
}
```

Illustrative unsupported-question response:

```json
{
  "status": "insufficient_evidence",
  "response": "The available reports do not contain enough evidence to answer this question.",
  "sources": []
}
```

Reference labels such as `S1` are stable within a response and map to persisted chunk IDs. They are not durable identifiers across queries. Return only cited sources, ordered by their assigned context labels, with duplicate references removed. Source metadata and text must come from storage, never from model-generated metadata. For the current Markdown profile, supporting text is the stored section body rather than a model-selected quotation.

Use HTTP 200 for both valid answer states. Invalid query input remains a client error. Database, provider, and generation-format failures remain server errors; they must not be reported as insufficient evidence. Retain a bounded generic error response without leaking provider output or document content.

## Acceptance criteria

1. Given a valid question, the generation request includes the original question and clearly separated retrieved context.
2. Given an answerable question against the sample reports, the API returns `status: answered`, nonempty answer text, and at least one source.
3. Every inline citation maps to exactly one returned source. Every returned source was retrieved for that request and cited by the answer.
4. Returned chunk IDs, upload IDs, section metadata, and supporting text match the persisted chunks. The model cannot supply or override these fields.
5. Given no retrieved chunks, the API returns `insufficient_evidence` with an empty sources array without invoking generation.
6. Given retrieved context that does not answer the question, the model can return an insufficient-evidence decision, which the API normalizes to the documented response with no citations.
7. Malformed structured output, unknown reference labels, missing citations on an answered result, or contradictory result fields produce a controlled server error rather than an unvalidated answer. There is no automatic regeneration retry in this version.
8. The prompt instructs the model to use supplied evidence, cite factual claims, abstain when evidence is inadequate, and treat instructions inside documents as untrusted content.
9. Existing authentication and query validation remain in force, including the supported retrieval limit of 1–5. Invalid inputs produce a client error before generation.
10. Provider and database failures remain distinguishable from valid insufficient-evidence responses in tests and internal diagnostics.
11. Automated tests cover the response contract, prompt construction, reference validation, and both answer states. A documented manual check exercises the real provider with sample reports.

## Implementation plan

### 1. Establish the response contract

- Introduce a typed search result in `services/rag/internal/rag` containing status, answer text, and resolved sources.
- Define source fields and JSON response types in the web layer.
- Update `Search` and `HandleQuery` to use the typed result while retaining the `response` field.
- Document how insufficient evidence and server errors differ.

### 2. Build question-aware context

- Use the existing retrieved chunk IDs, upload IDs, indices, and JSON content; avoid a database migration unless inspection identifies missing required data.
- Decode the supported project-report content into title, subheading, and section text. Define missing optional metadata as omitted fields; fail clearly on invalid persisted content.
- Assign labels deterministically from the retrieved order and build a label-to-chunk lookup.
- Construct the prompt with explicit instructions, the original question, and labeled context. Encode or escape content so document text cannot break the context boundaries.
- Return insufficient evidence immediately when retrieval is empty.

### 3. Request a structured generation result

- Extend the generator abstraction and Gemini adapter to request a schema-constrained result using capabilities supported by the installed SDK.
- Use a small internal schema: status, answer text, and a list of cited reference labels. Require inline answer references to use the same labels.
- Keep provider-specific schema configuration inside the adapter; update the mock generator to implement the same contract.
- Instruct the model to answer only from context and choose insufficient evidence when support is absent. Do not introduce an arbitrary vector-distance cutoff.

### 4. Validate and resolve citations

- Validate the result status, required fields, reference syntax, and consistency between inline references and the structured reference list.
- Require at least one valid reference for answered results. Reject unknown references and unsupported or contradictory fields.
- Resolve references through the request-local lookup, deduplicate them, and populate source fields directly from retrieved data.
- Normalize insufficient-evidence text in application code and return no sources.
- Treat invalid model output as a generation failure with a bounded error category in existing logs. Avoid recording raw prompts, source content, or provider responses solely for this feature.

### 5. Integrate the HTTP response

- Serialize the typed result using the documented response shape.
- Map service input-validation failures to HTTP 400, including retrieval limits above five; retain HTTP 500 for internal failures.
- Preserve the route, query parameters, and authentication middleware.

### 6. Verify and document

- Add unit tests with fake chunk repositories and generators for question inclusion, empty retrieval, answered results, abstention, invalid references, duplicate references, malformed output, and dependency failures.
- Add handler tests for JSON shape, status codes, and the retained `response` field.
- Test the adapter's structured-output handling without requiring live API credentials in automated tests.
- Run the RAG module tests and any affected shared-package checks using the repository's established tooling.
- Manually ingest the example Markdown reports and exercise one answerable and one unsupported question with the configured provider. Verify source fields against stored content.
- Add request/response examples and explain the limits of citation validation in the repository documentation.

## Delivery order

1. Typed result and question-aware prompt.
2. Structured generation and citation validation.
3. API integration, focused tests, and documentation.

The feature is complete when the acceptance criteria pass and the manual provider check is recorded. Automated tests establish contract and validation behavior; the manual check assesses model behavior without making live generation a deterministic test dependency.

## Risks and implementation decisions

- A valid source label does not guarantee that the answer accurately represents its source. Keep this limitation explicit and inspect representative sample answers.
- Nearest-neighbor retrieval can return irrelevant chunks. This version relies on explicit model abstention after retrieval; tune retrieval quality separately using an evaluation set.
- Documents can contain prompt-injection attempts. Clear boundaries and instructions reduce exposure but do not establish a security guarantee.
- Returning full section text can increase response sizes. The first version uses the existing maximum of five retrieved chunks; excerpt selection and response-size limits can be scoped separately if actual section sizes require them.
- Confirm structured-output support in the installed Gemini SDK before implementation. If the SDK cannot enforce the schema, retain strict application parsing and validation and record the adapter limitation.

## Code touchpoints

- `services/rag/internal/rag/rag_service.go`: result types, prompt construction, evidence decisions, and reference resolution.
- `services/rag/internal/ai/gemini.go`: structured generation configuration and parsing.
- `services/rag/internal/ai/mock.go`: generator contract changes.
- `services/rag/internal/web/handlers.go`: API serialization and error mapping.
- `services/rag/internal/chunk/postgres_repo.go`: existing source identity and retrieval behavior; change only if necessary.
- Corresponding RAG service tests and repository API documentation.
