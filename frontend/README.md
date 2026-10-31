# Raggy workspace

A responsive React interface built with TanStack Start, Tailwind CSS, and shadcn/ui (Radix primitives). It uses the existing cookie-authenticated backend APIs; production contains no demo data.

## Features

- Sign in, automatic access-token refresh, session-expiry handling, and sign out.
- A paginated documents list with live chunk-completion counts, progress bars, and server event updates with polling fallback.
- Markdown document uploads through the dashboard API and its signed storage URL, with file validation and transfer retry.
- Questions across the workspace, cancellable requests, explicit insufficient-evidence results, and source excerpts opened from inline citations.
- Responsive layouts, keyboard navigation, accessible dialog primitives, and loading, empty, and error states.

The dashboard records expected and completed chunks and marks a document ready when all expected embeddings have been persisted. Existing documents without completion events show an unknown chunk total. See [chunk progress](../docs/chunk-progress.md) for event delivery, migrations, and deployment details. The UI does not add document-level query filters or deletion controls.

## Run locally

For an interface-only preview, set `VITE_DEMO_MODE=true` in `.env` and restart Vite. A permanent banner identifies sample data. Questions return canned examples, uploads only change in-memory metadata, and no files, credentials, or prompts are sent to services. Reloading resets the preview. Leave this flag false for the live backend.

Use Node 22.12+ or Node 24 and the npm lockfile:

```sh
npm ci
cp .env.example .env
npm run dev
```

Open `http://localhost:3000`. Set `API_PROXY_TARGET` in `.env` to the running nginx entrypoint, normally `http://localhost`. This is a Vite development proxy for `/api`; no credential or token is stored in localStorage. Session cookies remain HttpOnly and are managed by the auth service.

For the existing Compose stack, use its nginx entrypoint. The frontend honors the `PORT` environment variable (80 in Compose), with Vite falling back to 3000 locally. Do not set the development proxy to the frontend's own address. When running the frontend alone inside Docker, use `API_PROXY_TARGET=http://host.docker.internal` to reach nginx on the host.

The signed upload URL must be reachable from the user's browser, and storage must allow browser PUT requests and the returned signed headers through CORS. The UI sends those headers exactly as returned and omits session cookies on storage requests. Configure CORS for the actual UI origin when deploying; no storage permissions are changed by this frontend.

Use an existing provisioned account. The UI does not include registration. Question answering requires the citation-enabled backend, valid Gemini configuration, and real document embeddings; see [the query API documentation](../docs/rag-query-api.md).

## Check and build

```sh
npm run typecheck
npm test
npm run build
```

Vitest uses a separate configuration so component tests do not start the SSR server. Tests cover session refresh and expiry, sign-in/out, signed upload headers, file validation, answer/source interactions, insufficient evidence, connection failures, and retained questions after errors. These tests use fixture API responses and do not call a real AI provider.

`npm run build` generates the Nitro Node server in `.output/server/index.mjs`. Serve it behind the same `/api` reverse proxy as the backend; the Vite development proxy is not part of the production output.
