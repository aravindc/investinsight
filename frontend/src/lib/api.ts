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

export function fetchTransactions(scopeParam: string) {
  return fetchJson<PortfolioTransaction[]>('/transactions', { scope: scopeParam })
}

export function fetchHoldingsSummary(scopeParam: string) {
  return fetchJson<HoldingsSummary>('/portfolio/holdings-summary', { scope: scopeParam })
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

export type AllocationGroupBy = 'sector' | 'sub-sector' | 'category' | 'sub-category'

// Gain and yield are null unless every holding in the group was priced.
export type AllocationFigures = {
  instrumentCount: number
  unpricedCount: number
  investedGbp: string
  marketValueGbp: string
  gainGbp: string | null
  gainPercent: string | null
  trailingYearDividendsGbp: string
  yieldPercent: string | null
  // The priced holding with the most value, and its share of the group.
  largestHolding: { name: string; ticker: string; sharePercent: string } | null
  // How many equal-sized holdings would be as concentrated (1 / Σ weight²).
  effectiveHoldings: string | null
}

export type AllocationGroup = {
  name: string
  parent: string | null
  unclassified: boolean
}

export type Allocation = AllocationFigures & {
  name: string
  // The sector or category a sub-sector or sub-category belongs to; null at
  // the top level. Sub-level names are only unique within their parent.
  parent: string | null
  // A holding with no sub-sector, grouped as "Unclassified" under its sector.
  unclassified: boolean
  sharePercent: string
}

export type Allocations = {
  groupBy: AllocationGroupBy
  totals: AllocationFigures
  groups: Allocation[]
}

export function fetchAllocations(scopeParam: string, groupBy: AllocationGroupBy) {
  return fetchJson<Allocations>('/portfolio/allocations', { scope: scopeParam, groupBy })
}

export type AllocationMatrix = {
  rowsBy: AllocationGroupBy
  colsBy: AllocationGroupBy
  rows: AllocationGroup[]
  cols: AllocationGroup[]
  // Non-empty combinations only; row and col index into rows and cols.
  cells: (AllocationFigures & { row: number; col: number; sharePercent: string })[]
  totals: AllocationFigures
}

export function fetchAllocationMatrix(
  scopeParam: string,
  rows: AllocationGroupBy,
  cols: AllocationGroupBy,
) {
  return fetchJson<AllocationMatrix>('/portfolio/allocation-matrix', { scope: scopeParam, rows, cols })
}

// Month-end market value per group (plus today), values aligned with groups.
export type AllocationSeries = {
  groupBy: AllocationGroupBy
  groups: AllocationGroup[]
  points: { date: string; valuesGbp: string[] }[]
}

export function fetchAllocationSeries(scopeParam: string, groupBy: AllocationGroupBy) {
  return fetchJson<AllocationSeries>('/portfolio/allocation-series', { scope: scopeParam, groupBy })
}

export type DividendStatus = 'paid' | 'announced' | 'estimated'

// One dividend payment. date is the payment date (actual, announced or
// estimated); amountGbp is null for an announcement with no past payment
// to estimate it from; exDate is the announced ex-dividend date.
export type DividendEvent = {
  date: string
  instrumentId: string
  ticker: string
  name: string
  amountGbp: string | null
  status: DividendStatus
  exDate: string | null
}

// Paid dividends in the year before today, expected ones in the year after.
export type DividendCalendar = {
  today: string
  from: string
  to: string
  events: DividendEvent[]
}

export function fetchDividendCalendar(scopeParam: string) {
  return fetchJson<DividendCalendar>('/portfolio/dividend-calendar', { scope: scopeParam })
}

export function fetchMonthlyDividends(scopeParam: string) {
  return fetchJson<MonthlyDividend[]>('/portfolio/monthly-dividends', { scope: scopeParam })
}

export function fetchPortfolioIrr(scopeParam: string) {
  return fetchJson<PortfolioIrr>('/portfolio/irr', { scope: scopeParam })
}

export function fetchPassiveIncome(scopeParam: string) {
  return fetchJson<PassiveIncome>('/portfolio/passive-income', { scope: scopeParam })
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

export type InstrumentClassification = {
  category: string
  subCategory: string
  sector: string
  /** "" when not set */
  subSector: string
}

export type Instrument = InstrumentClassification & {
  id: string
  ticker: string
  name: string
}

export function fetchInstruments() {
  return fetchJson<Instrument[]>('/instruments')
}

export async function updateInstrumentClassification(
  instrumentId: string,
  classification: InstrumentClassification,
) {
  await sendJson<void>('PUT', `/instruments/${instrumentId}/classification`, classification)
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

export type CashIsa = {
  accountId: string
  accountName: string
  /** The balance as entered, on balanceDate (before interest since). */
  balanceGbp: string
  balanceDate: string
  /** Balance plus interest credited at each completed month end. */
  valueGbp: string
  accruedThisMonthGbp: string
  interestCreditedGbp: string
  currentRatePercent: string
  nextCreditDate: string
  rates: { effectiveFrom: string; ratePercent: string }[]
}

export type CashIsaSummary = {
  totalValueGbp: string
  totalAccruedThisMonthGbp: string
  asOf: string
  cashIsas: CashIsa[]
}

/** Cash ISAs in scope; omit scopeParam for every account. */
export function fetchCashIsas(scopeParam?: string) {
  return fetchJson<CashIsaSummary>('/cash-isas', scopeParam ? { scope: scopeParam } : undefined)
}

export async function saveCashIsa(accountId: string, balance: string, balanceDate: string, ratePercent: string) {
  await sendJson<void>('PUT', `/accounts/${accountId}/cash-isa`, { balance, balanceDate, ratePercent })
}

export async function setCashIsaRate(accountId: string, ratePercent: string, effectiveFrom: string) {
  await sendJson<void>('POST', `/accounts/${accountId}/cash-isa/rates`, { ratePercent, effectiveFrom })
}

export async function deleteCashIsa(accountId: string) {
  await sendJson<void>('DELETE', `/accounts/${accountId}/cash-isa`)
}

export type SubSector = {
  name: string
  instrumentCount: number
}

export type Sector = {
  name: string
  instrumentCount: number
  subSectors: SubSector[]
}

export function fetchSectors() {
  return fetchJson<Sector[]>('/sectors')
}

export function createSector(name: string) {
  return postJson<Sector>('/sectors', { name })
}

export async function deleteSector(name: string) {
  await sendJson<void>('DELETE', `/sectors/${encodeURIComponent(name)}`)
}

export function createSubSector(sector: string, name: string) {
  return postJson<{ sector: string; name: string }>(
    `/sectors/${encodeURIComponent(sector)}/sub-sectors`,
    { name },
  )
}

export async function deleteSubSector(sector: string, name: string) {
  await sendJson<void>(
    'DELETE',
    `/sectors/${encodeURIComponent(sector)}/sub-sectors/${encodeURIComponent(name)}`,
  )
}

export type SubCategory = {
  name: string
  instrumentCount: number
}

export type Category = {
  name: string
  instrumentCount: number
  subCategories: SubCategory[]
}

export function fetchCategories() {
  return fetchJson<Category[]>('/categories')
}

export function createCategory(name: string) {
  return postJson<{ name: string }>('/categories', { name })
}

export async function deleteCategory(name: string) {
  await sendJson<void>('DELETE', `/categories/${encodeURIComponent(name)}`)
}

export function createSubCategory(category: string, name: string) {
  return postJson<{ category: string; name: string }>(
    `/categories/${encodeURIComponent(category)}/sub-categories`,
    { name },
  )
}

export async function deleteSubCategory(category: string, name: string) {
  await sendJson<void>(
    'DELETE',
    `/categories/${encodeURIComponent(category)}/sub-categories/${encodeURIComponent(name)}`,
  )
}

export type ClassificationImportResult = {
  updated: number
  unchanged: number
  createdCategories: string[]
  createdSubCategories: string[]
  createdSectors: string[]
  createdSubSectors: string[]
  skipped: { line: number; ticker: string; reason: string }[]
}

export async function importClassifications(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  const res = await fetch('/api/instruments/classification-import', { method: 'POST', body: formData })
  if (!res.ok) {
    const data = await res.json().catch(() => null)
    throw new Error(data?.error ?? `Import failed: ${res.status}`)
  }
  return res.json() as Promise<ClassificationImportResult>
}
