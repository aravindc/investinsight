-- +goose Up
-- Sectors move from a fixed list in code (internal/sector, frontend
-- lib/sector.ts) into a table so new ones can be created from the Admin
-- page. instruments.sector references it by name.
CREATE TABLE sectors (
    name VARCHAR(50) PRIMARY KEY CHECK (btrim(name) <> ''),
    -- Display order; built-ins keep their original order and "Other" stays last.
    sort_order INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX sectors_name_lower_key ON sectors (lower(name));

INSERT INTO sectors (name, sort_order) VALUES
    ('Technology', 10),
    ('Financials', 20),
    ('Healthcare', 30),
    ('Consumer Staples', 40),
    ('Consumer Discretionary', 50),
    ('Energy', 60),
    ('Materials', 70),
    ('Industrials', 80),
    ('Utilities', 90),
    ('Real Estate', 100),
    ('Communication Services', 110),
    ('Funds & ETFs', 120),
    ('Other', 1000000);

-- Keep any sector already on an instrument but missing above, so the
-- foreign key below can't fail.
INSERT INTO sectors (name, sort_order)
SELECT DISTINCT sector, 500000 FROM instruments
WHERE sector NOT IN (SELECT name FROM sectors);

ALTER TABLE instruments
    ADD CONSTRAINT instruments_sector_fkey
    FOREIGN KEY (sector) REFERENCES sectors(name) ON UPDATE CASCADE;

-- +goose Down
ALTER TABLE instruments DROP CONSTRAINT IF EXISTS instruments_sector_fkey;
DROP TABLE IF EXISTS sectors;
