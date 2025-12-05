CREATE TABLE IF NOT EXISTS keep.refresh_versions (
  user_id uuid PRIMARY KEY,
  version integer NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT fk_refresh_versions_user FOREIGN KEY (user_id) REFERENCES keep.users(id) ON DELETE CASCADE
);
