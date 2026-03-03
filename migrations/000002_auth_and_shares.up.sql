-- Users for auth
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- Add owner to files (nullable for existing rows)
ALTER TABLE files ADD COLUMN user_id TEXT REFERENCES users(id);
CREATE INDEX IF NOT EXISTS idx_files_user_id ON files (user_id);

-- Share links: one token per file, owner can revoke by deleting row
CREATE TABLE IF NOT EXISTS file_shares (
  file_id TEXT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  token TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ,
  PRIMARY KEY (file_id)
);

CREATE INDEX IF NOT EXISTS idx_file_shares_token ON file_shares (token);
