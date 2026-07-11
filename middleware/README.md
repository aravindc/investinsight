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
- `GET /api/portfolio/holdings-summary` — invested (cost basis) value of
  instruments currently held (net shares > 0). Market value based on live
  prices is not yet available and is always returned as `0` until a price
  feed is wired in.
- `GET /api/portfolio/dividends-summary` — total dividend income received
  to date (sum of `net_amount_gbp` for `DIVIDEND` entries).
- `GET /api/portfolio/invested-value-series` — cumulative net capital
  invested (BUY minus SELL cash flow) as of each day with trading activity,
  across every instrument ever traded (including fully exited positions).
  The value at each point is the true running total to date; period
  filtering (all/1yr/3mth/1mth/1wk/1day) is done client-side by zooming into
  this same series, not recomputed per window.
- `POST /api/csv-upload` — upload a broker CSV export (`multipart/form-data`,
  field `file`; optional `hasHeader` form field, defaults to `true`). Parses
  and ingests the file the same way as `csv_loader`, then normalizes new
  rows into `portfolio_transactions`. Returns
  `{ inserted, duplicate, errors }`: `duplicate` counts rows that already
  exist (matched on `action` + `transaction_time` + `isin` + `internal_id`,
  see [Deduplication](#deduplication)) and were skipped rather than
  re-inserted; `errors` lists CSV rows that failed to parse.

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
