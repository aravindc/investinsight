-- +goose Up
CREATE TABLE source_transactions (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,         -- Database Auto-generated UUID
    action VARCHAR(50) NOT NULL,                           -- 1. Action
    transaction_time TIMESTAMP WITH TIME ZONE NOT NULL,    -- 2. Time
    isin VARCHAR(12),                                      -- 3. ISIN
    ticker VARCHAR(20),                                    -- 4. Ticker
    name VARCHAR(255),                                     -- 5. Name
    notes TEXT,                                            -- 6. Notes
    internal_id VARCHAR(100),                              -- 7. ID from CSV (Not a Primary Key anymore)
    no_of_shares NUMERIC(18, 8) NOT NULL,                  -- 8. No. of shares
    price_per_share NUMERIC(18, 4) NOT NULL,               -- 9. Price / share
    currency_price_per_share VARCHAR(3) NOT NULL,          -- 10. Currency (Price / share)
    exchange_rate NUMERIC(18, 6) DEFAULT 1.0,              -- 11. Exchange rate
    result NUMERIC(18, 4),                                 -- 12. Result
    currency_result VARCHAR(3),                            -- 13. Currency (Result)
    total NUMERIC(18, 4) NOT NULL,                         -- 14. Total
    currency_total VARCHAR(3) NOT NULL,                    -- 15. Currency (Total)
    withholding_tax NUMERIC(18, 4) DEFAULT 0.0,            -- 16. Withholding tax
    currency_withholding_tax VARCHAR(3),                   -- 17. Currency (Withholding tax)
    stamp_duty_reserve_tax NUMERIC(18, 4) DEFAULT 0.0,    -- 18. Stamp duty reserve tax
    currency_stamp_duty_reserve_tax VARCHAR(3),            -- 19. Currency (Stamp duty reserve tax)
    currency_conversion_fee NUMERIC(18, 4) DEFAULT 0.0,    -- 20. Currency conversion fee
    currency_conversion_fee_currency VARCHAR(3)            -- 21. Currency (Currency conversion fee)
);

CREATE INDEX idx_source_ticker_time ON source_transactions(ticker, transaction_time);

-- 1. Asset Registry
CREATE TABLE instruments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    isin VARCHAR(12) UNIQUE NOT NULL,
    ticker VARCHAR(20) NOT NULL,
    name VARCHAR(255) NOT NULL,
    base_currency VARCHAR(3) NOT NULL, -- e.g., 'USD' for AAPL, 'GBX' or 'GBP' for VUKE
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. Standardized Ledger
CREATE TABLE portfolio_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_transaction_id UUID REFERENCES source_transactions(id),
    instrument_id UUID REFERENCES instruments(id), -- NULL for pure cash events (deposits/interest)

    -- Normalized categories: 'BUY', 'SELL', 'DIVIDEND', 'DEPOSIT', 'WITHDRAWAL', 'INTEREST'
    entry_type VARCHAR(20) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Share math
    shares NUMERIC(18, 8) DEFAULT 0.0,
    price_per_share_local NUMERIC(18, 4) DEFAULT 0.0,
    fx_rate_to_gbp NUMERIC(18, 6) DEFAULT 1.0,

    -- Strict GBP Ledger Columns (Your portfolio base currency)
    net_amount_gbp NUMERIC(18, 4) NOT NULL,       -- Pure asset value in GBP
    fees_gbp NUMERIC(18, 4) DEFAULT 0.0,           -- Combined FX fees, broker fees
    taxes_gbp NUMERIC(18, 4) DEFAULT 0.0,          -- Stamp duty (SDRT) + Withholding taxes
    total_impact_gbp NUMERIC(18, 4) NOT NULL       -- Actual cash out/in from your ISA balance
);

-- Indices for rapid performance calculations (XIRR/TWR)
CREATE INDEX idx_perf_tx_timestamp ON portfolio_transactions(timestamp);
CREATE INDEX idx_perf_tx_instrument ON portfolio_transactions(instrument_id);

-- +goose Down
DROP TABLE IF EXISTS portfolio_transactions CASCADE;
DROP TABLE IF EXISTS instruments CASCADE;
DROP TABLE IF EXISTS source_transactions CASCADE;
