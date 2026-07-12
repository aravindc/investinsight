-- +goose Up
CREATE TABLE households (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    household_id UUID NOT NULL REFERENCES households(id),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (household_id, name)
);

CREATE INDEX idx_accounts_household ON accounts(household_id);

-- Seed a default household + account so existing pre-account data has
-- somewhere to backfill into (see later migrations), rather than being
-- orphaned. Fixed IDs (not gen_random_uuid()) so later migrations can
-- reference them directly without a lookup.
INSERT INTO households (id, name)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default Household');

INSERT INTO accounts (id, household_id, name)
VALUES ('00000000-0000-0000-0000-000000000002',
        '00000000-0000-0000-0000-000000000001',
        'Default Account');

-- +goose Down
DROP TABLE IF EXISTS accounts CASCADE;
DROP TABLE IF EXISTS households CASCADE;
