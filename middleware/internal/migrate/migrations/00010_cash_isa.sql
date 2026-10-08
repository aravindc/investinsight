-- +goose Up
-- A Cash ISA attached to an account: a known balance on a date, plus a
-- history of interest rates. Interest isn't stored; it is replayed day by
-- day from balance_date (see internal/cashisa), so re-entering the balance
-- (e.g. after a deposit) simply re-anchors the calculation.
CREATE TABLE cash_isas (
    account_id UUID PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    balance NUMERIC(14, 2) NOT NULL CHECK (balance >= 0),
    balance_date DATE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- AER-style annual rate, applying from effective_from until the next row.
CREATE TABLE cash_isa_rates (
    account_id UUID NOT NULL REFERENCES cash_isas(account_id) ON DELETE CASCADE,
    effective_from DATE NOT NULL,
    annual_rate_percent NUMERIC(6, 3) NOT NULL CHECK (annual_rate_percent >= 0 AND annual_rate_percent <= 100),
    PRIMARY KEY (account_id, effective_from)
);

-- +goose Down
DROP TABLE IF EXISTS cash_isa_rates;
DROP TABLE IF EXISTS cash_isas;
