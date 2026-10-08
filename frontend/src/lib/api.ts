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
  unpricedHoldings: number
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
  sector: string
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

async function patchNoContent(path: string, body: unknown): Promise<void> {
  const res = await fetch(`/api${path}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const data = await res.json().catch(() => null)
    throw new Error(data?.error ?? `Request to ${path} failed: ${res.status}`)
  }
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

export async function backfillLogos() {
  const res = await fetch('/api/logos/backfill', { method: 'POST' })
  if (!res.ok) {
    throw new Error(`Logo backfill failed: ${res.status}`)
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

export type Instrument = {
  id: string
  ticker: string
  name: string
  sector: string
}

export function fetchInstruments() {
  return fetchJson<Instrument[]>('/instruments')
}

export function updateInstrumentSector(instrumentId: string, sectorName: string) {
  return patchNoContent(`/instruments/${instrumentId}/sector`, { sector: sectorName })
}

export type Trading212SyncStatus = 'idle' | 'running' | 'succeeded' | 'failed'

/** An account's Trading 212 link. The API key/secret are never returned. */
export type Trading212Connection = {
  accountId: string
  keyHint: string
  t212AccountId: number | null
  currency: string | null
  updatedAt: string
  syncStatus: Trading212SyncStatus
  syncMessage: string | null
  syncStartedAt: string | null
  syncFinishedAt: string | null
  syncedThrough: string | null
  lastInserted: number
  lastDuplicate: number
}

async function sendJson<T>(method: 'PUT' | 'POST' | 'DELETE', path: string, body?: unknown): Promise<T | null> {
  const res = await fetch(`/api${path}`, {
    method,
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (!res.ok) {
    const data = await res.json().catch(() => null)
    throw new Error(data?.error ?? `Request to ${path} failed: ${res.status}`)
  }
  return res.status === 204 ? null : (res.json() as Promise<T>)
}

export function fetchTrading212Connections() {
  return fetchJson<Trading212Connection[]>('/trading212/connections')
}

export async function connectTrading212(accountId: string, apiKey: string, apiSecret: string) {
  return (await sendJson<Trading212Connection>('PUT', `/accounts/${accountId}/trading212`, {
    apiKey,
    apiSecret,
  }))!
}

export async function disconnectTrading212(accountId: string) {
  await sendJson<void>('DELETE', `/accounts/${accountId}/trading212`)
}

export async function syncTrading212(accountId: string) {
  await sendJson<{ status: string }>('POST', `/accounts/${accountId}/trading212/sync`)
}

export type ReconciliationLine = {
  key: string
  label: string
  trading212: string
  app: string
  /** app − Trading 212 */
  difference: string
  differencePercent: string | null
  matches: boolean
}

export type Trading212Reconciliation = {
  accountId: string
  t212AccountId: number
  currency: string
  fetchedAt: string
  tolerancePercent: string
  lines: ReconciliationLine[]
  unpricedHoldings: number
  trading212Info: {
    totalValue: string
    cashAvailable: string
    cashInPies: string
    cashReserved: string
    realizedProfitLoss: string
  }
}

export async function fetchTrading212Reconciliation(accountId: string) {
  const res = await fetch(`/api/accounts/${accountId}/trading212/reconciliation`)
  if (!res.ok) {
    const data = await res.json().catch(() => null)
    throw new Error(data?.error ?? `Reconciliation failed: ${res.status}`)
  }
  return res.json() as Promise<Trading212Reconciliation>
}

export type Trading212Cash = {
  cashGbp: string
  cashInPiesGbp: string
  accountsIncluded: number
  accountsExcluded: { accountId: string; name: string; reason: string }[]
  oldestFetchedAt: string | null
}

export function fetchTrading212Cash(scopeParam: string) {
  return fetchJson<Trading212Cash>('/trading212/cash', { scope: scopeParam })
}
