# InvestInsight middleware

Gin-based API server that serves the portfolio ledger to the frontend.

## Stack

- **Gin** — HTTP router
- **zap** — structured logging
- **godotenv** — `.env` config loading
- **lib/pq** — Postgres driver
- **shopspring/decimal** — exact decimal arithmetic for money values (all
  GBP/share amounts are serialized as decimal strings, never `float64`, to
  avoid binary floating-point rounding)
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

## Build

```sh
go build -o bin/middleware .
```
