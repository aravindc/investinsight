# InvestInsight

A personal investment portfolio ledger. Broker CSV exports are ingested into a
Postgres database and normalized into a standardized, GBP-denominated
transaction ledger for performance analysis (e.g. XIRR/TWR).

## Architecture

```
CSV export (broker) --> csv_loader (Go) --> Postgres
                                             ├─ source_transactions   (raw staging)
                                             ├─ instruments           (asset registry)
                                             └─ portfolio_transactions (normalized GBP ledger)
                                                        │
                                                        ▼
                                      middleware (Gin API) --> frontend (React dashboard)
```

- **`source_transactions`** stores each CSV row largely as-is.
- **`instruments`** is a deduplicated registry of assets (keyed by ISIN).
- **`portfolio_transactions`** is the standardized ledger: every entry is
  normalized to an `entry_type` (`BUY`, `SELL`, `DIVIDEND`, `DEPOSIT`,
  `WITHDRAWAL`, `INTEREST`, `STOCK_DISTRIBUTION`, `STOCK_SPLIT`,
  `ACQUISITION`) with amounts converted to GBP.

Ingestion is atomic: the CSV loader inserts raw rows and runs the
normalization engine inside a single database transaction, so a failure in
either phase rolls back the whole import.

## Project layout

- **`docker-compose.yml`** — Postgres 18 + pgAdmin services for local
  development.
- **`csv_loader/`** — Go CLI that parses a broker CSV file, loads it into
  `source_transactions`, and runs the normalization engine to populate
  `portfolio_transactions`.
- **`backend/`** — broker CSV exports by year (e.g. `2025.csv`,
  `2025_all.csv`).
- **`middleware/`** — Go API server (Gin, zap, dotenv) that serves the
  portfolio ledger to the frontend.
- **`frontend/`** — React (Vite + TypeScript) dashboard for viewing
  holdings, dividends, and transactions. Uses Tailwind CSS, shadcn/ui, and
  TanStack Query/Table. See [`frontend/README.md`](frontend/README.md).
- **`wip.sql`** — scratch queries used during development.

## Getting started

### 1. Start the database

Copy the env template and fill in credentials:

```sh
cp .env.example .env
```

Then start the stack:

```sh
docker compose up -d
```

This starts Postgres on `localhost:5888` and pgAdmin on `localhost:5050`,
using the credentials from `.env` (see `docker-compose.yml`).

### 2. Run the middleware API

```sh
cd middleware
cp .env.example .env   # fill in DB credentials
go run .
```

On startup the middleware applies database migrations (via
[goose](https://github.com/pressly/goose), embedded in `internal/migrate`)
before serving. Serves the ledger to the frontend at `http://localhost:3000`.
See [`middleware/README.md`](middleware/README.md) for endpoint and migration
details.

### 3. Configure the CSV loader

Create a `.env` file inside `csv_loader/` with:

```
DB_HOST=localhost
DB_PORT=5888
DB_USER=admin
DB_PASSWORD=<password>
DB_NAME=investinsight_db
DB_SSLMODE=disable
```

### 4. Load a CSV file

```sh
cd csv_loader
go build -o csv_loader .
./csv_loader -f path/to/export.csv
```

Use `-h=false` if the CSV file has no header row.

### 5. Run the frontend

```sh
cd frontend
npm install
npm run dev
```

Opens the dashboard at `http://localhost:5173`, proxying `/api/*` to the
middleware. See [`frontend/README.md`](frontend/README.md).

## Expected CSV format

The loader expects 21 columns matching a Trading 212–style export:

```
Action, Time, ISIN, Ticker, Name, Notes, ID, No. of shares, Price / share,
Currency (Price / share), Exchange rate, Result, Currency (Result), Total,
Currency (Total), Withholding tax, Currency (Withholding tax),
Stamp duty reserve tax, Currency (Stamp duty reserve tax),
Currency conversion fee, Currency (Currency conversion fee)
```

## Notes

- Prices quoted in `GBX` (pence) are automatically converted to `GBP`.
- `.env` files and other local configuration contain credentials and should
  never be committed.
