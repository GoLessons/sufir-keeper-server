DROP TABLE IF EXISTS keep.items_files;
ALTER TABLE keep.items ALTER COLUMN data_encrypted SET NOT NULL;
ALTER TABLE keep.items ALTER COLUMN data_nonce SET NOT NULL;
