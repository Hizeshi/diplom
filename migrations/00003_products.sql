-- +goose Up
CREATE TABLE products (
    id                SERIAL PRIMARY KEY,
    article           VARCHAR(100) NOT NULL UNIQUE,
    name_raw          TEXT NOT NULL,
    brand_id          INTEGER REFERENCES brands(id) ON DELETE SET NULL,
    product_type      VARCHAR(100),
    series_id         INTEGER REFERENCES product_series(id) ON DELETE SET NULL,
    color_id          INTEGER REFERENCES colors(id) ON DELETE SET NULL,
    price             NUMERIC(15, 2),
    stock             INTEGER DEFAULT 0,
    description       TEXT,
    params_text       TEXT,
    is_active         BOOLEAN DEFAULT true,
    configurator_type TEXT,
    search_vector     TSVECTOR,
    created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at        TIMESTAMP
);

CREATE INDEX idx_products_brand         ON products(brand_id);
CREATE INDEX idx_products_type          ON products(product_type);
CREATE INDEX idx_products_price         ON products(price);
CREATE INDEX idx_products_article       ON products(article);
CREATE INDEX idx_products_search_vector ON products USING GIN(search_vector);
CREATE INDEX idx_products_name_trgm     ON products USING GIN(name_raw gin_trgm_ops);
CREATE INDEX idx_products_article_trgm  ON products USING GIN(article gin_trgm_ops);

-- Auto-update search_vector on insert/update
CREATE OR REPLACE FUNCTION products_search_vector_trigger() RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('russian', coalesce(NEW.name_raw, '')), 'A') ||
        setweight(to_tsvector('russian', coalesce(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tsvectorupdate
    BEFORE INSERT OR UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION products_search_vector_trigger();

CREATE TABLE product_images (
    id           SERIAL PRIMARY KEY,
    product_id   INTEGER REFERENCES products(id) ON DELETE CASCADE,
    image_url    VARCHAR NOT NULL,
    bucket       TEXT NOT NULL DEFAULT 'product-images',
    object_path  TEXT NOT NULL,
    display_order INTEGER DEFAULT 0,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE product_vectors (
    id                    SERIAL PRIMARY KEY,
    product_id            INTEGER REFERENCES products(id) ON DELETE CASCADE,
    combined_embedding    VECTOR(1024),
    name_embedding        VECTOR(1024),
    description_embedding VECTOR(1024),
    attributes_embedding  VECTOR(1024),
    created_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS product_vectors;
DROP TABLE IF EXISTS product_images;
DROP TRIGGER  IF EXISTS tsvectorupdate ON products;
DROP FUNCTION IF EXISTS products_search_vector_trigger;
DROP TABLE    IF EXISTS products;
