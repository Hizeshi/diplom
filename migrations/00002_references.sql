-- +goose Up
CREATE TABLE brands (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR NOT NULL,
    logo_url    VARCHAR,
    country     VARCHAR,
    description TEXT,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMP
);

CREATE TABLE colors (
    id         SERIAL PRIMARY KEY,
    name       VARCHAR NOT NULL,
    hex_code   VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE product_series (
    id          SERIAL PRIMARY KEY,
    brand_id    INTEGER REFERENCES brands(id) ON DELETE SET NULL,
    name        VARCHAR NOT NULL,
    description TEXT,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS product_series;
DROP TABLE IF EXISTS colors;
DROP TABLE IF EXISTS brands;
