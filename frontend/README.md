# InvestInsight frontend

Portfolio dashboard for InvestInsight, built with React + Vite + TypeScript.

## Stack

- **Vite + React + TypeScript**
- **Tailwind CSS v4** + **shadcn/ui** (`radix-nova` style) for components
- **React Router** for navigation between Holdings / Dividends / Admin
- **TanStack Query** for data fetching against the middleware API
- **TanStack Table** for the transactions table
- **Recharts** for the invested vs. market value chart

## Theme

Light, dark, and system-preference modes, toggled from the header (`src/lib/theme.tsx`,
`src/components/ThemeToggle.tsx`). The choice persists in `localStorage`; in
"system" mode it follows the OS preference live via a `matchMedia` listener.
An inline script in `index.html` applies the right mode before React hydrates,
so there's no flash of the wrong theme on load.

## Navigation

A top nav bar (`src/components/NavBar.tsx`) links three pages, routed with
React Router (`src/App.tsx`, pages in `src/pages/`):

- **Holdings** (`/`) — the portfolio analysis view:
  - **Value card** — current market value (GBP) of instruments held today,
    priced live from Yahoo Finance by the middleware, with invested (cost
    basis) value shown underneath.
  - **Total profit card** — market value minus invested value, with a
    gain/loss percentage badge and a daily change line derived from the
    last two points of the market-value series.
  - **Holdings by sector pie chart** — current holdings' invested (cost
    basis) value grouped by sector, using a static ticker-to-sector mapping
    in the middleware. Capped at 8 slices (the validated categorical
    palette size); the smallest sectors beyond that fold into "Other".
  - **Cumulative invested vs. market value chart** — two lines: running
    total of net capital invested, and historical market value (shares
    held × closing price, both from the middleware), with period buttons
    (All / 1yr / 3mth / 1mth / 1wk / 1day) that zoom into the same
    underlying series client-side. The market value line starts as early
    as any held instrument has price data — usually close to the invested
    value line's start — with a note above the chart on the rare occasion
    it starts later.
  - **Transactions table** — recent ledger entries, filterable by type and
    instrument (client-side, via TanStack Table column filters).
- **Dividends** (`/dividends`):
  - **Dividends received chart** — cumulative dividend income received to
    date, month by month, with the running total shown as a badge.
  - **Dividend growth chart** — dividend income by calendar month, grouped
    bars comparing each year present in the ledger side by side.
- **Admin** (`/admin`) — operational actions, not analysis:
  - **Import transactions card** — upload a broker CSV export. Reports how
    many transactions were newly imported vs. skipped as duplicates, plus
    any rows that failed to parse. Successful uploads refresh the
    holdings, dividends, and transactions data automatically.
  - **Market prices card** — manually trigger the middleware's price
    backfill (`POST /api/market-prices/backfill`), which otherwise only
    runs when called directly. Reports instruments processed and any
    tickers that couldn't be priced.

## Development

```sh
npm install
npm run dev
```

The dev server proxies `/api/*` requests to the Gin middleware at
`http://localhost:3000` (see `vite.config.ts`). Start the middleware first,
or update the proxy target if it runs on a different port.

## API expectations

The dashboard expects nine endpoints from the middleware (`src/lib/api.ts`):

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
- `GET /api/portfolio/sector-allocations` — current holdings' invested
  value (GBP) grouped by sector, sorted descending.
- `GET /api/portfolio/monthly-dividends` — total dividend income (GBP) per
  calendar month across the full ledger history, ordered chronologically.
  Powers both the dividends-received (cumulative) and dividend-growth
  (year-over-year) charts client-side.
- `POST /api/csv-upload` — upload a broker CSV export (`multipart/form-data`);
  returns `{ inserted, duplicate, errors }`.
- `POST /api/market-prices/backfill` — fetch and store daily closing prices
  for every held instrument missing history; returns
  `{ instrumentsProcessed, failed }`.

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
