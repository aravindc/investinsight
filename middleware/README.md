# InvestInsight middleware

Gin-based API server that serves the portfolio ledger to the frontend.

## Stack

- **Gin** — HTTP router
- **zap** — structured logging
- **godotenv** — `.env` config loading
- **lib/pq** — Postgres driver
- **shopspring/decimal** — exact decimal arithmetic for money values (all
  GBP/share amounts are serialized as decimal strings, never `float64`, to
  avoid binary floating-point rounding). Note: `ValuePoint.ValueGBP` uses
  `decimal.Decimal` directly, so it serializes as a quoted JSON string
  rather than a bare number.
- **pressly/goose** — SQL schema migrations, embedded into the binary and
  applied automatically on startup
- Market prices are fetched directly from Yahoo Finance's public chart API
  (no third-party Go client library or Python dependency)

## Layout

- `main.go` — app wiring: config, logger, DB connection, migrations,
  middleware, routes.
- `internal/config` — `.env`-backed configuration.
- `internal/logger` — zap logger construction.
- `internal/db` — Postgres connection helper.
- `internal/migrate` — embeds `migrations/*.sql` and applies them with goose.
- `internal/portfolio` — repository, handlers, and JSON models for the
  portfolio ledger endpoints.
- `internal/ingest` — CSV parsing, broker-action normalization, and the
  ledger ingestion engine (ported from `csv_loader`) used by the CSV upload
  endpoint.
- `internal/marketprice` — Yahoo Finance chart API client, ticker→symbol
  mapping, the `market_prices` table repository, and the pricing service
  (backfill + current-price lookup) used by the holdings market value.
- `internal/sector` — static ticker→sector mapping used by the sector
  allocation endpoint.

## Migrations

Schema migrations live in `internal/migrate/migrations/` as goose-formatted
SQL files (`-- +goose Up` / `-- +goose Down`), embedded into the binary via
`go:embed`. They run automatically against the configured database on every
`go run .` / server startup — no separate migration step is needed.

To add a new migration, install the goose CLI and generate a new file in
that directory:

```sh
go install github.com/pressly/goose/v3/cmd/goose@latest
goose -dir internal/migrate/migrations create <name> sql
```

Then edit the generated file's `Up`/`Down` sections.

## Configuration

Copy `.env.example` to `.env` and fill in the database credentials (see the
root `docker-compose.yml`):

```sh
cp .env.example .env
```

## Running

```sh
go run .
```

The server listens on `APP_PORT` (default `3000`) and expects Postgres to be
reachable per `.env`. CORS is currently locked to `http://localhost:5173`
(the Vite dev server).

## Endpoints

- `GET /healthz` — liveness check.
- `GET /api/transactions` — most recent 200 `portfolio_transactions`, joined
  with instrument name.
- `GET /api/portfolio/holdings-summary` — invested (cost basis) value and
  current market value (GBP) of instruments currently held (net shares > 0).
  Market value is computed live from Yahoo Finance closing prices; see
  [Market prices](#market-prices). If a price can't be resolved for a
  holding (e.g. delisted/unmapped ticker), that holding is excluded from
  the market value total and logged, rather than failing the request.
- `GET /api/portfolio/dividends-summary` — total dividend income received
  to date (sum of `net_amount_gbp` for `DIVIDEND` entries).
- `GET /api/portfolio/invested-value-series` — cumulative net capital
  invested (BUY minus SELL cash flow) as of each day with trading activity,
  across every instrument ever traded (including fully exited positions).
  The value at each point is the true running total to date; period
  filtering (all/1yr/3mth/1mth/1wk/1day) is done client-side by zooming into
  this same series, not recomputed per window.
- `GET /api/portfolio/market-value-series` — true historical portfolio
  market value: for each day, (shares held that day × that instrument's
  closing price that day), summed across every currently-held instrument
  with price data. See [Market value series](#market-value-series) for how
  the start date and unpriceable instruments are handled.
- `POST /api/csv-upload` — upload a broker CSV export (`multipart/form-data`,
  field `file`; optional `hasHeader` form field, defaults to `true`). Parses
  and ingests the file the same way as `csv_loader`, then normalizes new
  rows into `portfolio_transactions`. Returns
  `{ inserted, duplicate, errors }`: `duplicate` counts rows that already
  exist (matched on `action` + `transaction_time` + `isin` + `internal_id`,
  see [Deduplication](#deduplication)) and were skipped rather than
  re-inserted; `errors` lists CSV rows that failed to parse.
- `GET /api/portfolio/sector-allocations` — current holdings' invested
  (cost basis) value grouped by sector, sorted descending. Sectors come
  from a static ticker→sector mapping (`internal/sector`), not a live data
  source; see [Sector allocation](#sector-allocation).
- `POST /api/market-prices/backfill` — fetch and store daily closing
  prices, from each instrument's first BUY date to today, for every
  instrument missing history (or top up instruments only missing recent
  days). Returns `{ instrumentsProcessed, failed }`, where `failed` lists
  tickers that couldn't be fetched (see
  [Market prices](#market-prices) for why this happens).

## Market prices

Daily closing prices are stored in `market_prices` (instrument, date,
close, currency, source), one row per instrument per trading day.

- **Symbol mapping**: an instrument's Yahoo Finance symbol is derived
  heuristically from its broker ticker and base currency — GBP/GBX
  (LSE-listed) → `.L` suffix (e.g. `BARC` → `BARC.L`); EUR (Deutsche
  Börse/XETRA-listed, the only other exchange currently in this portfolio)
  → `.DE` suffix; everything else is used as-is (assumed US-listed). This
  is a heuristic, not an explicit mapping, so some tickers won't resolve —
  investment trusts and similarly-suffixed funds in particular (e.g. this
  portfolio has two JPMorgan trust share classes, `JEGP`/`JEQP`, that don't
  resolve). Unresolvable tickers are skipped (logged, not fatal) both
  during backfill and when pricing holdings.
- **Backfill**: `POST /api/market-prices/backfill` fetches history from
  scratch for instruments with no stored prices, or only the missing days
  for instruments that already have some, starting from each instrument's
  earliest share-affecting event (BUY, SELL, STOCK_SPLIT,
  STOCK_DISTRIBUTION, or ACQUISITION — not just BUY, since an instrument
  can be acquired purely via a corporate action like a spin-off). It is
  also safe to call repeatedly — already up-to-date instruments are
  skipped.
- **Read path (t-2 / t-1 / t-0)**: `GET /api/portfolio/holdings-summary`
  (and any other current-price read) sources dates two or more calendar
  days old from the stored `market_prices` table, and always live-fetches
  the last two calendar days directly from Yahoo Finance — so the most
  recent price is never more than a few seconds stale, while historical
  reads stay fast and don't hit Yahoo repeatedly. If a stored price doesn't
  exist for the exact requested date (e.g. a weekend), the most recent
  prior close is used.
- **Currency conversion**: Yahoo reports LSE prices in pence (`GBp`);
  these are stored as-is in `market_prices.currency = 'GBP'` and divided
  by 100 when converted to GBP, matching the pence-per-share convention
  already used for broker CSV prices. Non-GBP instruments (currently EUR
  and USD) are converted using a live FX rate fetched from Yahoo Finance's
  FX pair symbols (e.g. `GBP=X` for USD/GBP, `EURGBP=X` for EUR/GBP) — see
  `internal/marketprice/fx.go`.

## Market value series

`GET /api/portfolio/market-value-series` reconstructs historical portfolio
value by combining two things per currently-held instrument: a running
daily share count (from `portfolio_transactions`) and a daily closing price
(from `market_prices`).

- **Start date and per-day best effort**: the series starts as early as
  *any* currently-held instrument has both shares and a price — in
  practice this closely matches the invested-value series' start. Each
  day's total is a best-effort sum: it includes every held instrument that
  has a price for that specific day, and simply omits any that don't
  (whether because that instrument's price history hadn't started yet by
  that day, or because it's permanently unpriceable — e.g. this portfolio
  has two JPMorgan trust share classes, `JEGP`/`JEQP`, whose symbols never
  resolve). This means an early day's total can under-count relative to
  what it would show once all holdings are priced, but it never blocks the
  chart the way requiring full coverage across the whole history would.
- **FX for historical dates**: rather than one live FX call per
  instrument-day (slow), the full FX rate history for each non-GBP
  currency in the series is fetched once per request and looked up by
  date in memory.

## Sector allocation

`GET /api/portfolio/sector-allocations` groups current holdings' invested
(cost basis) value by sector, for the dashboard's sector pie chart.

- **Static mapping, not a live lookup**: `internal/sector` hand-maps each
  ticker to a sector. There's no sector field anywhere upstream (broker
  CSVs, the `instruments` table, or the Yahoo chart API already in use), so
  this is maintained by hand rather than fetched — it needs a manual update
  when a new ticker outside the map is bought for the first time (it falls
  back to `Other` until then).
- **Options-income wrapper products** (e.g. `AAPI`/IncomeShares Apple
  Options, `MSTP`/YieldMax MSTR) are categorized by their *underlying*
  stock's sector, not as their own asset class — `internal/sector`'s
  `underlyingTicker` map resolves the wrapper to its underlying before the
  sector lookup. Multi-stock basket wrappers (e.g. IncomeShares Magnificent
  7, YieldMax Big Tech) have no single underlying and fall into
  `Funds & ETFs` instead.
- **Funds & ETFs** is its own bucket for diversified instruments (index
  funds, investment trusts, bond/commodity ETFs) rather than attributing
  them to one sector.
- Based on invested (cost basis) value, the same basis as the holdings
  summary card — not live market value, so it has no Yahoo Finance
  dependency and is never affected by unpriceable instruments.

## Deduplication

`source_transactions` has a unique index on
`(action, transaction_time, isin, internal_id)`. Re-uploading a CSV that
overlaps with previously loaded data — including through `csv_loader`, since
both paths write to the same table — inserts only the new rows; rows that
already exist are skipped and counted, not re-inserted or errored.

## Build

```sh
go build -o bin/middleware .
```
