-- +goose Up
ALTER TABLE uploads
  ADD COLUMN chunk_total INTEGER CHECK (chunk_total >= 0),
  ADD COLUMN chunks_processed INTEGER NOT NULL DEFAULT 0 CHECK (chunks_processed >= 0),
  ADD CONSTRAINT upload_progress_bounds CHECK (chunk_total IS NULL OR chunks_processed <= chunk_total);
ALTER TABLE uploads DROP CONSTRAINT uploads_status_check;
ALTER TABLE uploads ADD CONSTRAINT uploads_status_check
  CHECK (status IN ('pending', 'uploaded', 'processing', 'processed'));

-- +goose Down
UPDATE uploads SET status = 'uploaded' WHERE status = 'processing';
ALTER TABLE uploads DROP CONSTRAINT uploads_status_check;
ALTER TABLE uploads ADD CONSTRAINT uploads_status_check CHECK (status IN ('pending', 'uploaded', 'processed'));
ALTER TABLE uploads DROP CONSTRAINT upload_progress_bounds;
ALTER TABLE uploads DROP COLUMN chunk_total, DROP COLUMN chunks_processed;
