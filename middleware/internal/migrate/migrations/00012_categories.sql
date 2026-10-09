-- +goose Up
-- Instruments are classified by Category → Sub-Category (e.g. Equity →
-- ETF), alongside the independent Sector (00011). A sub-category belongs to
-- exactly one category; instruments reference the (category, sub_category)
-- pair, so the two can't disagree.
CREATE TABLE categories (
    name VARCHAR(50) PRIMARY KEY CHECK (btrim(name) <> ''),
    sort_order INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX categories_name_lower_key ON categories (lower(name));

CREATE TABLE sub_categories (
    category VARCHAR(50) NOT NULL REFERENCES categories(name) ON UPDATE CASCADE ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL CHECK (btrim(name) <> ''),
    sort_order INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (category, name)
);
CREATE UNIQUE INDEX sub_categories_name_lower_key ON sub_categories (category, lower(name));

-- "Other" / "Other" is the protected catch-all, and the default for newly
-- imported instruments until they're classified.
INSERT INTO categories (name, sort_order) VALUES
    ('Equity', 10),
    ('Other', 1000000);
INSERT INTO sub_categories (category, name, sort_order) VALUES
    ('Equity', 'Stock', 10),
    ('Equity', 'ETF', 20),
    ('Other', 'Other', 1000000);

ALTER TABLE instruments
    ADD COLUMN category VARCHAR(50) NOT NULL DEFAULT 'Other',
    ADD COLUMN sub_category VARCHAR(50) NOT NULL DEFAULT 'Other';

-- Classify existing instruments from their sector.
UPDATE instruments SET category = 'Equity',
    sub_category = CASE WHEN sector = 'Funds & ETFs' THEN 'ETF' ELSE 'Stock' END;

ALTER TABLE instruments
    ADD CONSTRAINT instruments_sub_category_fkey
    FOREIGN KEY (category, sub_category) REFERENCES sub_categories(category, name) ON UPDATE CASCADE;

-- +goose Down
ALTER TABLE instruments DROP CONSTRAINT IF EXISTS instruments_sub_category_fkey;
ALTER TABLE instruments DROP COLUMN IF EXISTS sub_category, DROP COLUMN IF EXISTS category;
DROP TABLE IF EXISTS sub_categories;
DROP TABLE IF EXISTS categories;
