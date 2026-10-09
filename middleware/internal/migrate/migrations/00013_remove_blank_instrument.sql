-- +goose Up
-- Rows without an ISIN (e.g. "Dividend adjustment" withholding-tax
-- corrections, mapped to DIVIDEND) used to be linked to an instrument with
-- an empty ISIN, ticker and name. Ingest no longer does that; unlink the
-- existing rows (they still count towards dividend totals, which don't need
-- an instrument), remove the blank instrument, and forbid empty ISINs.
UPDATE portfolio_transactions SET instrument_id = NULL
WHERE instrument_id IN (SELECT id FROM instruments WHERE btrim(isin) = '');

DELETE FROM market_prices WHERE instrument_id IN (SELECT id FROM instruments WHERE btrim(isin) = '');
DELETE FROM instrument_logos WHERE instrument_id IN (SELECT id FROM instruments WHERE btrim(isin) = '');
DELETE FROM instruments WHERE btrim(isin) = '';

ALTER TABLE instruments ADD CONSTRAINT instruments_isin_not_blank CHECK (btrim(isin) <> '');

-- +goose Down
ALTER TABLE instruments DROP CONSTRAINT IF EXISTS instruments_isin_not_blank;
