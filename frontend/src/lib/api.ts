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

export type InvestedValuePoint = {
  date: string
  valueGbp: string
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

export function fetchInvestedValueSeries() {
  return fetchJson<InvestedValuePoint[]>('/portfolio/invested-value-series')
}

export async function uploadCsv(file: File, hasHeader: boolean) {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('hasHeader', String(hasHeader))

  const res = await fetch('/api/csv-upload', {
    method: 'POST',
    body: formData,
  })
  if (!res.ok) {
    throw new Error(`CSV upload failed: ${res.status}`)
  }
  return res.json() as Promise<CsvUploadResult>
}
