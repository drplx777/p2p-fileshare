CREATE TABLE IF NOT EXISTS files (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  size_bytes BIGINT NOT NULL,
  sha256_hex TEXT NOT NULL,
  cid TEXT NOT NULL UNIQUE,
  local_path TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_files_created_at ON files (created_at DESC);
