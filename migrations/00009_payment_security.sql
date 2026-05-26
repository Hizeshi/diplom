-- +goose Up

-- Track payment transaction ID for idempotency (prevent duplicate webhook processing).
ALTER TABLE orders ADD COLUMN IF NOT EXISTS payment_tx_id TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS orders_payment_tx_id_idx ON orders (payment_tx_id) WHERE payment_tx_id IS NOT NULL;

-- Normalise legacy data before adding constraints.
-- 'paid' was used before we standardised on 'confirmed'.
UPDATE orders SET status = 'confirmed' WHERE status = 'paid';
-- Empty payment_method → 'other' (was optional in old checkout flow).
UPDATE orders SET payment_method = 'other' WHERE payment_method IS NULL OR payment_method = '';

-- Constrain allowed values for status and payment_status.
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('new', 'confirmed', 'cancelled', 'processing'));

ALTER TABLE orders ADD CONSTRAINT orders_payment_status_check
    CHECK (payment_status IN ('pending', 'success', 'failed'));

ALTER TABLE orders ADD CONSTRAINT orders_payment_method_check
    CHECK (payment_method IN ('card', 'cash', 'other'));

-- +goose Down
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_payment_method_check;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_payment_status_check;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
DROP INDEX IF EXISTS orders_payment_tx_id_idx;
ALTER TABLE orders DROP COLUMN IF EXISTS payment_tx_id;
