-- +goose Up
ALTER TABLE source_transactions
    ADD COLUMN account_id UUID REFERENCES accounts(id);

UPDATE source_transactions
SET account_id = '00000000-0000-0000-0000-000000000002'
WHERE account_id IS NULL;

CREATE INDEX idx_source_transactions_account ON source_transactions(account_id);

-- Per-account dedup: same broker transaction can now independently exist
-- once per account (two different accounts holding an identical trade no
-- longer collide), while re-uploading the same account's CSV still dedupes.
-- NULL account_id rows (from not-yet-migrated ingest code, during the
-- transition window) won't dedupe against each other -- Postgres treats
-- NULL as distinct in unique indexes -- acceptable since this window closes
-- once the ingest-wiring step lands.
DROP INDEX IF EXISTS idx_source_transactions_dedup;

CREATE UNIQUE INDEX idx_source_transactions_dedup
    ON source_transactions (account_id, action, transaction_time, isin, internal_id);

-- +goose Down
DROP INDEX IF EXISTS idx_source_transactions_dedup;
CREATE UNIQUE INDEX idx_source_transactions_dedup
    ON source_transactions (action, transaction_time, isin, internal_id);

DROP INDEX IF EXISTS idx_source_transactions_account;
ALTER TABLE source_transactions DROP COLUMN IF EXISTS account_id;
