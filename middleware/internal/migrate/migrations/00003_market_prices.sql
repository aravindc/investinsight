-- +goose Up
-- Daily closing price history per instrument, in the instrument's native
-- quote currency (e.g. GBp for LSE-listed stocks, USD for US stocks) as
-- returned by the price source — not converted to GBP here.
CREATE TABLE market_prices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instrument_id UUID NOT NULL REFERENCES instruments(id),
    price_date DATE NOT NULL,
    close_price NUMERIC(18, 6) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    source VARCHAR(20) NOT NULL DEFAULT 'yahoo',
    fetched_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (instrument_id, price_date)
);

CREATE INDEX idx_market_prices_instrument_date ON market_prices(instrument_id, price_date);

-- +goose Down
DROP TABLE IF EXISTS market_prices CASCADE;
