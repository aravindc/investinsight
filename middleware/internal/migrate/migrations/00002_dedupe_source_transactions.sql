-- +goose Up
-- Dedup key for re-uploaded broker CSV exports: a row is considered the
-- same transaction if it shares the same action, timestamp, instrument
-- (isin), and broker-assigned internal_id. Verified unique across all
-- existing rows, including cash-only entries (deposits/interest/etc, where
-- isin is NULL) and dividend rows (where the broker leaves internal_id
-- blank) — those remain distinguishable by action + transaction_time.
CREATE UNIQUE INDEX idx_source_transactions_dedup
    ON source_transactions (action, transaction_time, isin, internal_id);

-- +goose Down
DROP INDEX IF EXISTS idx_source_transactions_dedup;
