CREATE TABLE IF NOT EXISTS keep.items_files (
  item_id uuid PRIMARY KEY,
  s3_bucket varchar(64) NOT NULL,
  s3_key varchar(512) NOT NULL,
  size bigint NOT NULL,
  sha256 varchar(64) NOT NULL,
  CONSTRAINT fk_items_files_item FOREIGN KEY (item_id) REFERENCES keep.items(id) ON DELETE CASCADE
);

ALTER TABLE keep.items ALTER COLUMN data_encrypted DROP NOT NULL;
ALTER TABLE keep.items ALTER COLUMN data_nonce DROP NOT NULL;
