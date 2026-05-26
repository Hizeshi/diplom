-- +goose Up
CREATE TABLE sales_knowledge (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    topic      TEXT,
    content    TEXT,
    embedding  VECTOR(1024),
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE contact_requests (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       TEXT NOT NULL,
    email      TEXT NOT NULL,
    message    TEXT NOT NULL,
    status     TEXT DEFAULT 'new',
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS contact_requests;
DROP TABLE IF EXISTS sales_knowledge;
