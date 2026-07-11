export type PortfolioTransaction = {
  id: string
  entryType: string
  timestamp: string
  instrumentName: string | null
  shares: string
  netAmountGbp: string
  totalImpactGbp: string
}

export type HoldingsSummary = {
  investedValueGbp: string
  marketValueGbp: string
}

export type DividendsSummary = {
  totalDividendsGbp: string
}

async function fetchJson<T>(path: string): Promise<T> {
  const res = await fetch(`/api${path}`)
  if (!res.ok) {
    throw new Error(`Request to ${path} failed: ${res.status}`)
  }
  return res.json() as Promise<T>
}

export function fetchTransactions() {
  return fetchJson<PortfolioTransaction[]>('/transactions')
}

export function fetchHoldingsSummary() {
  return fetchJson<HoldingsSummary>('/portfolio/holdings-summary')
}

export function fetchDividendsSummary() {
  return fetchJson<DividendsSummary>('/portfolio/dividends-summary')
}
