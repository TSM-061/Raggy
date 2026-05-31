-- +goose Up
CREATE TABLE
  uploads (
    id UUID PRIMARY KEY DEFAULT uuidv7 (),
    status TEXT NOT NULL DEFAULT 'receiving' CHECK (
      status IN ('processing', 'ingested', 'embedding', 'ready')
    ),
    chunk_total INT NOT NULL DEFAULT 0 CHECK (chunk_total >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW ()
  );

CREATE TABLE
  chunks (
    id UUID PRIMARY KEY DEFAULT uuidv7 (),
    upload_id UUID NOT NULL REFERENCES uploads (id) ON DELETE CASCADE,
    chunk_index INT NOT NULL CHECK (chunk_index >= 0),
    content JSONB NOT NULL,
    embedding vector(1536),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    UNIQUE (upload_id, chunk_index)
  );

-- +goose Down
DROP TABLE IF EXISTS chunks;
DROP TABLE IF EXISTS uploads;
