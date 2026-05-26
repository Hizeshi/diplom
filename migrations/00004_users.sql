-- +goose Up
-- profiles extends Supabase Auth users (auth.users)
CREATE TABLE profiles (
    id         UUID PRIMARY KEY,
    email      TEXT,
    full_name  TEXT,
    phone      TEXT,
    avatar_url TEXT,
    avatar_path TEXT,
    role       TEXT DEFAULT 'customer',
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE cart_items (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    UUID NOT NULL,
    product_id BIGINT NOT NULL,
    quantity   INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(user_id, product_id)
);

CREATE TABLE favorites (
    user_id    UUID NOT NULL,
    product_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY(user_id, product_id)
);

CREATE TABLE product_views (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    UUID NOT NULL,
    product_id BIGINT NOT NULL,
    viewed_at  TIMESTAMPTZ DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS product_views;
DROP TABLE IF EXISTS favorites;
DROP TABLE IF EXISTS cart_items;
DROP TABLE IF EXISTS profiles;
