-- +goose Up
-- Sector is now stored per instrument (previously a hardcoded Go map in
-- internal/sector), so it can be corrected via the Admin UI without a code
-- change/redeploy. Single-stock options-income wrapper products (e.g.
-- AAPI, an Apple-derived income ETF) store their resolved sector directly
-- (Technology, matching AAPL) rather than an indirection to an underlying
-- ticker. Unrecognized/future tickers default to 'Other', same as the old
-- ForTicker fallback.
ALTER TABLE instruments ADD COLUMN sector VARCHAR(50) NOT NULL DEFAULT 'Other';

-- +goose StatementBegin
UPDATE instruments SET sector = 'Technology' WHERE ticker = 'AAPL';
UPDATE instruments SET sector = 'Technology' WHERE ticker = 'AMD';
UPDATE instruments SET sector = 'Technology' WHERE ticker = 'AVGO';
UPDATE instruments SET sector = 'Technology' WHERE ticker = 'MSFT';
UPDATE instruments SET sector = 'Technology' WHERE ticker = 'NVDA';
UPDATE instruments SET sector = 'Technology' WHERE ticker = 'SGE';
UPDATE instruments SET sector = 'Technology' WHERE ticker = 'BKSY';
UPDATE instruments SET sector = 'Technology' WHERE ticker = 'PL';
UPDATE instruments SET sector = 'Technology' WHERE ticker = 'SMX';
UPDATE instruments SET sector = 'Technology' WHERE ticker = 'SPCX';
UPDATE instruments SET sector = 'Communication Services' WHERE ticker = 'GOOGL';
UPDATE instruments SET sector = 'Communication Services' WHERE ticker = 'META';
UPDATE instruments SET sector = 'Consumer Discretionary' WHERE ticker = 'AMZN';
UPDATE instruments SET sector = 'Consumer Discretionary' WHERE ticker = 'TSLA';
UPDATE instruments SET sector = 'Consumer Discretionary' WHERE ticker = 'MKS';
UPDATE instruments SET sector = 'Consumer Discretionary' WHERE ticker = 'TSCO';
UPDATE instruments SET sector = 'Consumer Discretionary' WHERE ticker = 'SBRY';
UPDATE instruments SET sector = 'Consumer Discretionary' WHERE ticker = 'PSN';
UPDATE instruments SET sector = 'Consumer Discretionary' WHERE ticker = 'TW';
UPDATE instruments SET sector = 'Consumer Staples' WHERE ticker = 'KO';
UPDATE instruments SET sector = 'Consumer Staples' WHERE ticker = 'PG';
UPDATE instruments SET sector = 'Consumer Staples' WHERE ticker = 'DGE';
UPDATE instruments SET sector = 'Consumer Staples' WHERE ticker = 'BATS';
UPDATE instruments SET sector = 'Consumer Staples' WHERE ticker = 'IMB';
UPDATE instruments SET sector = 'Consumer Staples' WHERE ticker = 'ABF';
UPDATE instruments SET sector = 'Consumer Staples' WHERE ticker = 'CCH';
UPDATE instruments SET sector = 'Consumer Staples' WHERE ticker = 'BAG';
UPDATE instruments SET sector = 'Consumer Staples' WHERE ticker = 'BVIC';
UPDATE instruments SET sector = 'Consumer Staples' WHERE ticker = 'TATE';
UPDATE instruments SET sector = 'Consumer Staples' WHERE ticker = 'ULVR';
UPDATE instruments SET sector = 'Consumer Staples' WHERE ticker = 'MICC';
UPDATE instruments SET sector = 'Healthcare' WHERE ticker = 'AZN';
UPDATE instruments SET sector = 'Healthcare' WHERE ticker = 'GSK';
UPDATE instruments SET sector = 'Financials' WHERE ticker = 'AV';
UPDATE instruments SET sector = 'Financials' WHERE ticker = 'ABDN';
UPDATE instruments SET sector = 'Financials' WHERE ticker = 'BARC';
UPDATE instruments SET sector = 'Financials' WHERE ticker = 'HSBA';
UPDATE instruments SET sector = 'Financials' WHERE ticker = 'LGEN';
UPDATE instruments SET sector = 'Financials' WHERE ticker = 'LLOY';
UPDATE instruments SET sector = 'Financials' WHERE ticker = 'MNG';
UPDATE instruments SET sector = 'Financials' WHERE ticker = 'MONY';
UPDATE instruments SET sector = 'Financials' WHERE ticker = 'NWG';
UPDATE instruments SET sector = 'Financials' WHERE ticker = 'SDLF';
UPDATE instruments SET sector = 'Financials' WHERE ticker = 'MAIN';
UPDATE instruments SET sector = 'Financials' WHERE ticker = 'OXLC';
UPDATE instruments SET sector = 'Financials' WHERE ticker = 'DBK';
UPDATE instruments SET sector = 'Financials' WHERE ticker = 'COIN';
UPDATE instruments SET sector = 'Energy' WHERE ticker = 'BP';
UPDATE instruments SET sector = 'Energy' WHERE ticker = 'SHEL';
UPDATE instruments SET sector = 'Materials' WHERE ticker = 'AAL';
UPDATE instruments SET sector = 'Materials' WHERE ticker = 'RIO';
UPDATE instruments SET sector = 'Materials' WHERE ticker = 'GLEN';
UPDATE instruments SET sector = 'Materials' WHERE ticker = 'CEY';
UPDATE instruments SET sector = 'Materials' WHERE ticker = 'AU';
UPDATE instruments SET sector = 'Materials' WHERE ticker = 'VALT';
UPDATE instruments SET sector = 'Materials' WHERE ticker = 'MNDI';
UPDATE instruments SET sector = 'Industrials' WHERE ticker = 'BA';
UPDATE instruments SET sector = 'Industrials' WHERE ticker = 'BNZL';
UPDATE instruments SET sector = 'Utilities' WHERE ticker = 'NG';
UPDATE instruments SET sector = 'Utilities' WHERE ticker = 'SSE';
UPDATE instruments SET sector = 'Utilities' WHERE ticker = 'UU';
UPDATE instruments SET sector = 'Utilities' WHERE ticker = 'NESF';
UPDATE instruments SET sector = 'Utilities' WHERE ticker = 'TRIG';
UPDATE instruments SET sector = 'Utilities' WHERE ticker = 'UKW';
UPDATE instruments SET sector = 'Real Estate' WHERE ticker = 'AGR';
UPDATE instruments SET sector = 'Real Estate' WHERE ticker = 'BLND';
UPDATE instruments SET sector = 'Real Estate' WHERE ticker = 'BYG';
UPDATE instruments SET sector = 'Real Estate' WHERE ticker = 'LAND';
UPDATE instruments SET sector = 'Real Estate' WHERE ticker = 'PHP';
UPDATE instruments SET sector = 'Real Estate' WHERE ticker = 'SGRO';
UPDATE instruments SET sector = 'Real Estate' WHERE ticker = 'O';
UPDATE instruments SET sector = 'Consumer Discretionary' WHERE ticker = 'BMW';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'EQQQ';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'IIND';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'INCU';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'JEGP';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'JEIP';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'JEQP';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'JGGI';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'SMIF';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'SSHY';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'STHS';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'VAGS';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'VHYL';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'VPNG';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'VUAG';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'VUSA';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'VWRP';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'WINC';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'CEPG';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'FEPD';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'FEPG';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'MAGD';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'QQQO';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'SPYO';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'XYLP';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'QYLP';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'TLTI';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'GLDE';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'SLVI';
UPDATE instruments SET sector = 'Funds & ETFs' WHERE ticker = 'YMAP';
UPDATE instruments SET sector = 'Technology' WHERE ticker = 'AAPI';
UPDATE instruments SET sector = 'Technology' WHERE ticker = 'AMDI';
UPDATE instruments SET sector = 'Consumer Discretionary' WHERE ticker = 'AMZD';
UPDATE instruments SET sector = 'Technology' WHERE ticker = 'AVGI';
UPDATE instruments SET sector = 'Financials' WHERE ticker = 'COII';
UPDATE instruments SET sector = 'Communication Services' WHERE ticker = 'GOOO';
UPDATE instruments SET sector = 'Communication Services' WHERE ticker = 'METI';
UPDATE instruments SET sector = 'Technology' WHERE ticker = 'MSFI';
UPDATE instruments SET sector = 'Other' WHERE ticker = 'MSTP';
UPDATE instruments SET sector = 'Technology' WHERE ticker = 'NVDD';
UPDATE instruments SET sector = 'Consumer Discretionary' WHERE ticker = 'TSLD';
-- +goose StatementEnd

-- +goose Down
ALTER TABLE instruments DROP COLUMN sector;
