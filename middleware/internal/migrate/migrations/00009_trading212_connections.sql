-- +goose Up
-- One Trading 212 API connection per account. The API key and secret are
-- stored encrypted (AES-256-GCM, key from CREDENTIALS_ENCRYPTION_KEY, with
-- the account id bound as associated data) and are never returned by the
-- API; key_hint (last 4 characters of the API key) is what the UI shows.
CREATE TABLE trading212_connections (
    account_id UUID PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    api_key_encrypted BYTEA NOT NULL,
    api_secret_encrypted BYTEA NOT NULL,
    key_hint VARCHAR(8) NOT NULL,
    t212_account_id BIGINT,
    currency VARCHAR(3),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Sync state. synced_through is the end of the last successfully
    -- ingested export window (informational: each sync starts from the
    -- account's newest transaction in source_transactions).
    sync_status VARCHAR(16) NOT NULL DEFAULT 'idle',
    sync_message TEXT,
    sync_started_at TIMESTAMP WITH TIME ZONE,
    sync_finished_at TIMESTAMP WITH TIME ZONE,
    synced_through TIMESTAMP WITH TIME ZONE,
    last_inserted INTEGER NOT NULL DEFAULT 0,
    last_duplicate INTEGER NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS trading212_connections;
