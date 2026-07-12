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

export type CsvUploadResult = {
  inserted: number
  duplicate: number
  errors: string[]
}

export type Account = {
  id: string
  householdId: string
  name: string
}

export type Household = {
  id: string
  name: string
}

export type InvestedValuePoint = {
  date: string
  valueGbp: string
}

export type SectorAllocation = {
  sector: string
  instrumentCount: number
  investedGbp: string
  marketValueGbp: string
  gainGbp: string
  gainPercent: string
}

export type MonthlyDividend = {
  year: number
  month: number
  totalGbp: string
}

export type BackfillResult = {
  instrumentsProcessed: number
  failed: string[]
}

export type PortfolioIrr = {
  irr: number | null
  currentHoldingsIrr: number | null
}

export type PassiveIncome = {
  yieldPercent: string
  trailingYearGbp: string
  changePercent: string | null
}

export type AssetOption = {
  instrumentId: string
  ticker: string
  name: string
}

export type AssetCategory = {
  sector: string
  instruments: AssetOption[]
}

export type HoldingRow = {
  instrumentId: string
  ticker: string
  name: string
  shares: string
  costBasisGbp: string
  costBasisPerShareGbp: string
  currentValueGbp: string
  currentValuePerShareGbp: string
  priced: boolean
  dividendsReceivedGbp: string
  dividendsReceivedPerShareGbp: string
  dividendYieldPercent: string | null
  dividendGrowthPercent: string | null
  totalProfitGbp: string | null
  totalProfitPercent: string | null
  irrPercent: string | null
  portfolioSharePercent: string | null
}

async function fetchJson<T>(path: string, params?: Record<string, string | string[]>): Promise<T> {
  const qs = new URLSearchParams()
  for (const [key, value] of Object.entries(params ?? {})) {
    if (Array.isArray(value)) {
      value.forEach((v) => qs.append(key, v))
    } else {
      qs.append(key, value)
    }
  }
  const queryString = qs.toString()
  const res = await fetch(`/api${path}${queryString ? `?${queryString}` : ''}`)
  if (!res.ok) {
    throw new Error(`Request to ${path} failed: ${res.status}`)
  }
  return res.json() as Promise<T>
}

async function postJson<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`/api${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const data = await res.json().catch(() => null)
    throw new Error(data?.error ?? `Request to ${path} failed: ${res.status}`)
  }
  return res.json() as Promise<T>
}

export function fetchTransactions(scopeParam: string) {
  return fetchJson<PortfolioTransaction[]>('/transactions', { scope: scopeParam })
}

export function fetchHoldingsSummary(scopeParam: string, instrumentIds?: string[]) {
  return fetchJson<HoldingsSummary>('/portfolio/holdings-summary', {
    scope: scopeParam,
    ...(instrumentIds ? { instrumentId: instrumentIds } : {}),
  })
}

export function fetchDividendsSummary(scopeParam: string) {
  return fetchJson<DividendsSummary>('/portfolio/dividends-summary', { scope: scopeParam })
}

export function fetchInvestedValueSeries(scopeParam: string) {
  return fetchJson<InvestedValuePoint[]>('/portfolio/invested-value-series', { scope: scopeParam })
}

export function fetchMarketValueSeries(scopeParam: string) {
  return fetchJson<InvestedValuePoint[]>('/portfolio/market-value-series', { scope: scopeParam })
}

export function fetchSectorAllocations(scopeParam: string) {
  return fetchJson<SectorAllocation[]>('/portfolio/sector-allocations', { scope: scopeParam })
}

export function fetchMonthlyDividends(scopeParam: string) {
  return fetchJson<MonthlyDividend[]>('/portfolio/monthly-dividends', { scope: scopeParam })
}

export function fetchPortfolioIrr(scopeParam: string, instrumentIds?: string[]) {
  return fetchJson<PortfolioIrr>('/portfolio/irr', {
    scope: scopeParam,
    ...(instrumentIds ? { instrumentId: instrumentIds } : {}),
  })
}

export function fetchPassiveIncome(scopeParam: string, instrumentIds?: string[]) {
  return fetchJson<PassiveIncome>('/portfolio/passive-income', {
    scope: scopeParam,
    ...(instrumentIds ? { instrumentId: instrumentIds } : {}),
  })
}

export function fetchAssetCategories(scopeParam: string) {
  return fetchJson<AssetCategory[]>('/portfolio/asset-categories', { scope: scopeParam })
}

export function fetchHoldingsTable(scopeParam: string) {
  return fetchJson<HoldingRow[]>('/portfolio/holdings-table', { scope: scopeParam })
}

export async function backfillMarketPrices() {
  const res = await fetch('/api/market-prices/backfill', { method: 'POST' })
  if (!res.ok) {
    throw new Error(`Market price backfill failed: ${res.status}`)
  }
  return res.json() as Promise<BackfillResult>
}

export async function uploadCsv(file: File, hasHeader: boolean, accountId: string) {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('hasHeader', String(hasHeader))
  formData.append('accountId', accountId)

  const res = await fetch('/api/csv-upload', {
    method: 'POST',
    body: formData,
  })
  if (!res.ok) {
    throw new Error(`CSV upload failed: ${res.status}`)
  }
  return res.json() as Promise<CsvUploadResult>
}

export function fetchAccounts() {
  return fetchJson<Account[]>('/accounts')
}

export function createAccount(householdId: string, name: string) {
  return postJson<Account>('/accounts', { householdId, name })
}

export function fetchHouseholds() {
  return fetchJson<Household[]>('/households')
}

export function createHousehold(name: string) {
  return postJson<Household>('/households', { name })
}
