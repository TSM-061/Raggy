-- +goose Up
CREATE TABLE
  uploads (
    id UUID PRIMARY KEY DEFAULT uuidv7 (),
    uploaded_by UUID NOT NULL,
    original_name TEXT NOT NULL,
    content_type TEXT NOT NULL,
    profile_hint TEXT NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'uploaded', 'processed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW ()
  );

-- +goose Down
DROP TABLE IF EXISTS uploads;