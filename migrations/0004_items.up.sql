CREATE TABLE IF NOT EXISTS keep.items (
  id uuid PRIMARY KEY,
  user_id uuid NOT NULL,
  title varchar(255) NOT NULL,
  type varchar(32) NOT NULL,
  data_encrypted bytea NOT NULL,
  data_nonce bytea NOT NULL,
  data_key_encrypted bytea NOT NULL,
  data_key_nonce bytea NOT NULL,
  kek_version integer NOT NULL,
  meta jsonb NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT fk_items_user FOREIGN KEY (user_id) REFERENCES keep.users(id) ON DELETE CASCADE,
  CONSTRAINT chk_items_type CHECK (type IN ('CREDENTIAL','CARD','TEXT','BINARY'))
);

CREATE INDEX IF NOT EXISTS idx_items_user_id ON keep.items(user_id);
CREATE INDEX IF NOT EXISTS idx_items_user_type ON keep.items(user_id, type);
CREATE INDEX IF NOT EXISTS idx_items_title_lower ON keep.items((lower(title)));
