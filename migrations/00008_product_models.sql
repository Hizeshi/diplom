-- +goose Up
ALTER TABLE products ADD COLUMN IF NOT EXISTS model_url TEXT;
ALTER TABLE products ADD COLUMN IF NOT EXISTS model_path TEXT;

-- +goose Down
ALTER TABLE products DROP COLUMN IF EXISTS model_path;
ALTER TABLE products DROP COLUMN IF EXISTS model_url;
