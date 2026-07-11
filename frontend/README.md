# InvestInsight frontend

Portfolio dashboard for InvestInsight, built with React + Vite + TypeScript.

## Stack

- **Vite + React + TypeScript**
- **Tailwind CSS v4** + **shadcn/ui** (`radix-nova` style) for components
- **TanStack Query** for data fetching against the middleware API
- **TanStack Table** for the transactions table

## Dashboard

- **Current holdings card** — invested (cost basis) value and market value
  of instruments currently held. Market value is `£0` until the middleware
  has a live price feed.
- **Total dividends card** — dividend income received to date. Shown
  alongside the holdings card in an even two-column row.
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

The dashboard expects three endpoints from the middleware (`src/lib/api.ts`):

- `GET /api/transactions` — list of `portfolio_transactions`, joined with
  instrument name.
- `GET /api/portfolio/holdings-summary` — invested value and market value
  (GBP) of instruments currently held.
- `GET /api/portfolio/dividends-summary` — total dividend income received
  to date (GBP).

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
