-- +goose Up
ALTER TABLE portfolio_transactions
    ADD COLUMN account_id UUID REFERENCES accounts(id);

-- Backfill existing rows — everything before multi-account support
-- implicitly belonged to "the one portfolio".
UPDATE portfolio_transactions
SET account_id = '00000000-0000-0000-0000-000000000002'
WHERE account_id IS NULL;

CREATE INDEX idx_portfolio_transactions_account ON portfolio_transactions(account_id);

-- +goose Down
DROP INDEX IF EXISTS idx_portfolio_transactions_account;
ALTER TABLE portfolio_transactions DROP COLUMN IF EXISTS account_id;
