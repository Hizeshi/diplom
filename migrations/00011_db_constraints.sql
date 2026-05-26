-- +goose Up

-- ── cart_items ───────────────────────────────────────────────────────────────
ALTER TABLE cart_items ADD CONSTRAINT cart_items_quantity_positive
    CHECK (quantity > 0);

CREATE INDEX IF NOT EXISTS cart_items_user_id_idx ON cart_items (user_id);

-- ── favorites ────────────────────────────────────────────────────────────────
CREATE INDEX IF NOT EXISTS favorites_user_id_idx ON favorites (user_id);

-- ── product_views ────────────────────────────────────────────────────────────
CREATE INDEX IF NOT EXISTS product_views_user_id_idx    ON product_views (user_id);
CREATE INDEX IF NOT EXISTS product_views_product_id_idx ON product_views (product_id);

-- ── orders ───────────────────────────────────────────────────────────────────
-- Foreign keys (were missing in original schema)
ALTER TABLE order_items
    ADD CONSTRAINT order_items_product_id_fk
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE SET NULL
    NOT VALID;

ALTER TABLE order_items ADD CONSTRAINT order_items_quantity_positive
    CHECK (quantity > 0);

ALTER TABLE order_items ADD CONSTRAINT order_items_price_positive
    CHECK (price_at_purchase >= 0);

CREATE INDEX IF NOT EXISTS orders_user_id_idx    ON orders (user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS orders_status_idx     ON orders (status);
CREATE INDEX IF NOT EXISTS orders_created_at_idx ON orders (created_at DESC);

-- ── profiles ─────────────────────────────────────────────────────────────────
ALTER TABLE profiles ADD CONSTRAINT profiles_role_check
    CHECK (role IN ('customer', 'admin', 'manager'));

-- +goose Down
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS profiles_role_check;
DROP INDEX IF EXISTS orders_created_at_idx;
DROP INDEX IF EXISTS orders_status_idx;
DROP INDEX IF EXISTS orders_user_id_idx;
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_price_positive;
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_quantity_positive;
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_product_id_fk;
DROP INDEX IF EXISTS product_views_product_id_idx;
DROP INDEX IF EXISTS product_views_user_id_idx;
DROP INDEX IF EXISTS favorites_user_id_idx;
DROP INDEX IF EXISTS cart_items_user_id_idx;
ALTER TABLE cart_items DROP CONSTRAINT IF EXISTS cart_items_quantity_positive;
