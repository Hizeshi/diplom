-- +goose Up
CREATE TABLE orders (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id        UUID,
    status         TEXT DEFAULT 'new',
    total_amount   NUMERIC NOT NULL,
    payment_method TEXT NOT NULL,
    payment_status TEXT DEFAULT 'pending',
    full_name      TEXT,
    phone          TEXT,
    address        TEXT,
    comment        TEXT,
    created_at     TIMESTAMPTZ DEFAULT now(),
    updated_at     TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE order_items (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id          BIGINT REFERENCES orders(id),
    product_id        BIGINT,
    product_name      TEXT,
    price_at_purchase NUMERIC,
    quantity          INTEGER,
    created_at        TIMESTAMPTZ DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
