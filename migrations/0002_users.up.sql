CREATE TABLE IF NOT EXISTS keep.users (
  id uuid PRIMARY KEY,
  login text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
