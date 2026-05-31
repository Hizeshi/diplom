-- +goose Up

-- Products: name, description, params
ALTER TABLE products
    ADD COLUMN IF NOT EXISTS name_i18n        JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS description_i18n JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS params_i18n      JSONB NOT NULL DEFAULT '{}';

-- Series
ALTER TABLE product_series
    ADD COLUMN IF NOT EXISTS name_i18n JSONB NOT NULL DEFAULT '{}';

-- Brands
ALTER TABLE brands
    ADD COLUMN IF NOT EXISTS name_i18n JSONB NOT NULL DEFAULT '{}';

-- Colors
ALTER TABLE colors
    ADD COLUMN IF NOT EXISTS name_i18n JSONB NOT NULL DEFAULT '{}';

-- GIN indexes for fast JSONB key lookup
CREATE INDEX IF NOT EXISTS idx_products_name_i18n        ON products USING GIN (name_i18n);
CREATE INDEX IF NOT EXISTS idx_product_series_name_i18n  ON product_series USING GIN (name_i18n);
CREATE INDEX IF NOT EXISTS idx_brands_name_i18n          ON brands USING GIN (name_i18n);
CREATE INDEX IF NOT EXISTS idx_colors_name_i18n          ON colors USING GIN (name_i18n);

-- +goose Down
DROP INDEX IF EXISTS idx_colors_name_i18n;
DROP INDEX IF EXISTS idx_brands_name_i18n;
DROP INDEX IF EXISTS idx_product_series_name_i18n;
DROP INDEX IF EXISTS idx_products_name_i18n;

ALTER TABLE colors        DROP COLUMN IF EXISTS name_i18n;
ALTER TABLE brands        DROP COLUMN IF EXISTS name_i18n;
ALTER TABLE product_series DROP COLUMN IF EXISTS name_i18n;
ALTER TABLE products      DROP COLUMN IF EXISTS params_i18n;
ALTER TABLE products      DROP COLUMN IF EXISTS description_i18n;
ALTER TABLE products      DROP COLUMN IF EXISTS name_i18n;
