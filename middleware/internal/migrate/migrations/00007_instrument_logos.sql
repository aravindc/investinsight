-- +goose Up
-- One logo image per instrument, fetched from logo.dev and cached locally
-- so the frontend never needs the API token and repeated page loads don't
-- re-hit the external service.
CREATE TABLE instrument_logos (
    instrument_id UUID PRIMARY KEY REFERENCES instruments(id),
    image_data BYTEA NOT NULL,
    content_type VARCHAR(50) NOT NULL,
    fetched_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS instrument_logos CASCADE;
