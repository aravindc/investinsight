# InvestInsight frontend

Portfolio dashboard for InvestInsight, built with React + Vite + TypeScript.

## Stack

- **Vite + React + TypeScript**
- **Tailwind CSS v4** + **shadcn/ui** (`radix-nova` style) for components
- **TanStack Query** for data fetching against the middleware API
- **TanStack Table** for the transactions table
- **Recharts** for the invested vs. market value chart

## Dashboard

- **Import transactions card** — upload a broker CSV export. Reports how
  many transactions were newly imported vs. skipped as duplicates, plus any
  rows that failed to parse. Successful uploads refresh the holdings,
  dividends, and transactions data automatically.
- **Current holdings card** — invested (cost basis) value and current
  market value (GBP) of instruments currently held, priced live from Yahoo
  Finance by the middleware.
- **Total dividends card** — dividend income received to date. Shown
  alongside the holdings card in an even two-column row.
- **Cumulative invested vs. market value chart** — two lines: running total
  of net capital invested, and historical market value (shares held ×
  closing price, both from the middleware), with period buttons
  (All / 1yr / 3mth / 1mth / 1wk / 1day) that zoom into the same underlying
  series client-side. The market value line starts as early as any held
  instrument has price data — usually close to the invested value line's
  start — with a note above the chart on the rare occasion it starts later.
- **Transactions table** — recent ledger entries, filterable by type and
  instrument (client-side, via TanStack Table column filters).

## Development

```sh
npm install
npm run dev
```

The dev server proxies `/api/*` requests to the Gin middleware at
`http://localhost:3000` (see `vite.config.ts`). Start the middleware first,
or update the proxy target if it runs on a different port.

## API expectations

The dashboard expects six endpoints from the middleware (`src/lib/api.ts`):

- `GET /api/transactions` — list of `portfolio_transactions`, joined with
  instrument name.
- `GET /api/portfolio/holdings-summary` — invested value and market value
  (GBP) of instruments currently held.
- `GET /api/portfolio/dividends-summary` — total dividend income received
  to date (GBP).
- `GET /api/portfolio/invested-value-series` — cumulative invested value
  (GBP) time series, one point per day with trading activity.
- `GET /api/portfolio/market-value-series` — historical market value (GBP)
  time series, one point per day, starting as early as any currently-held
  instrument has price data. Each day is a best-effort total — a holding
  without a price for that specific day is simply omitted from that day's
  sum, not from the whole series.
- `POST /api/csv-upload` — upload a broker CSV export (`multipart/form-data`);
  returns `{ inserted, duplicate, errors }`.

Adjust `src/lib/api.ts` and the response types as the middleware's actual API
shape solidifies.

## Adding shadcn/ui components

```sh
npx shadcn@latest add <component>
```

Components are written to `src/components/ui`.

## Build

```sh
npm run build
```
