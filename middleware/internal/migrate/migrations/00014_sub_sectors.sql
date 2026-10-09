-- +goose Up
-- Sectors split into sub-sectors (e.g. Financials → Banks, Insurance).
-- Unlike sub-categories, a sub-sector is optional on an instrument; when
-- set it must belong to the instrument's sector.
CREATE TABLE sub_sectors (
    sector VARCHAR(50) NOT NULL REFERENCES sectors(name) ON UPDATE CASCADE ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL CHECK (btrim(name) <> ''),
    sort_order INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (sector, name)
);
CREATE UNIQUE INDEX sub_sectors_name_lower_key ON sub_sectors (sector, lower(name));

ALTER TABLE instruments ADD COLUMN sub_sector VARCHAR(50);

-- MATCH SIMPLE (the default): no check while sub_sector is NULL.
ALTER TABLE instruments
    ADD CONSTRAINT instruments_sub_sector_fkey
    FOREIGN KEY (sector, sub_sector) REFERENCES sub_sectors(sector, name) ON UPDATE CASCADE;

-- +goose Down
ALTER TABLE instruments DROP CONSTRAINT IF EXISTS instruments_sub_sector_fkey;
ALTER TABLE instruments DROP COLUMN IF EXISTS sub_sector;
DROP TABLE IF EXISTS sub_sectors;
