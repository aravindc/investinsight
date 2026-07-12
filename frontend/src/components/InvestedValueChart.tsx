import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useAccountScope } from '@/lib/account-scope'
import {
  fetchInvestedValueSeries,
  fetchMarketValueSeries,
  type InvestedValuePoint,
} from '@/lib/api'
import { formatGbp } from '@/lib/format'

const PERIODS = [
  { key: 'all', label: 'All' },
  { key: '1y', label: '1yr' },
  { key: '3m', label: '3mth' },
  { key: '1m', label: '1mth' },
  { key: '1w', label: '1wk' },
  { key: '1d', label: '1day' },
] as const

type PeriodKey = (typeof PERIODS)[number]['key']

const PERIOD_DAYS: Record<Exclude<PeriodKey, 'all'>, number> = {
  '1y': 365,
  '3m': 90,
  '1m': 30,
  '1w': 7,
  '1d': 1,
}

type MergedPoint = {
  date: string
  investedGbp: number | null
  marketGbp: number | null
}

function mergeSeries(
  invested: InvestedValuePoint[],
  market: InvestedValuePoint[],
): MergedPoint[] {
  const marketByDate = new Map(market.map((p) => [p.date, Number(p.valueGbp)]))
  const allDates = Array.from(
    new Set([...invested.map((p) => p.date), ...market.map((p) => p.date)]),
  ).sort()

  const investedByDate = new Map(invested.map((p) => [p.date, Number(p.valueGbp)]))

  let lastInvested: number | null = null
  let lastMarket: number | null = null
  return allDates.map((date) => {
    if (investedByDate.has(date)) lastInvested = investedByDate.get(date)!
    if (marketByDate.has(date)) lastMarket = marketByDate.get(date)!
    return { date, investedGbp: lastInvested, marketGbp: lastMarket }
  })
}

function filterByPeriod(points: MergedPoint[], period: PeriodKey): MergedPoint[] {
  if (period === 'all' || points.length === 0) return points

  const lastDate = new Date(points[points.length - 1].date)
  const cutoff = new Date(lastDate)
  cutoff.setDate(cutoff.getDate() - PERIOD_DAYS[period])

  const visible = points.filter((p) => new Date(p.date) >= cutoff)

  // Carry the last known values into the window so short periods with no
  // activity don't render as empty.
  const before = points.filter((p) => new Date(p.date) < cutoff)
  if (visible.length === 0 && before.length > 0) {
    return [before[before.length - 1]]
  }
  if (before.length > 0 && (visible.length === 0 || new Date(visible[0].date) > cutoff)) {
    return [before[before.length - 1], ...visible]
  }
  return visible
}

export function InvestedValueChart() {
  const { scopeParam } = useAccountScope()
  const invested = useQuery({
    queryKey: ['invested-value-series', scopeParam],
    queryFn: () => fetchInvestedValueSeries(scopeParam!),
    enabled: !!scopeParam,
  })
  const market = useQuery({
    queryKey: ['market-value-series', scopeParam],
    queryFn: () => fetchMarketValueSeries(scopeParam!),
    enabled: !!scopeParam,
  })
  const [period, setPeriod] = useState<PeriodKey>('all')

  const merged = useMemo(
    () => mergeSeries(invested.data ?? [], market.data ?? []),
    [invested.data, market.data],
  )
  const filtered = useMemo(() => filterByPeriod(merged, period), [merged, period])

  const isLoading = invested.isLoading
  const hasError = invested.error
  const marketDataStartsLate =
    (market.data?.length ?? 0) > 0 &&
    (invested.data?.length ?? 0) > 0 &&
    market.data![0].date > invested.data![0].date

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>Cumulative invested vs. market value</CardTitle>
        <div className="flex gap-1">
          {PERIODS.map((p) => (
            <Button
              key={p.key}
              size="sm"
              variant={period === p.key ? 'default' : 'outline'}
              onClick={() => setPeriod(p.key)}
            >
              {p.label}
            </Button>
          ))}
        </div>
      </CardHeader>
      <CardContent className="h-80">
        {isLoading && <p className="text-muted-foreground text-sm">Loading…</p>}
        {hasError && (
          <p className="text-destructive text-sm">Failed to load invested value.</p>
        )}
        {!isLoading && filtered.length === 0 && (
          <p className="text-muted-foreground text-sm">No data for this period.</p>
        )}
        {!isLoading && filtered.length > 0 && (
          <>
            {marketDataStartsLate && (
              <p className="text-muted-foreground mb-2 text-xs">
                Market value starts from {market.data![0].date} — the earliest
                date with any price history for a currently-held instrument.
                Some early days may under-count holdings whose price history
                hadn't started yet.
              </p>
            )}
            <ResponsiveContainer width="100%" height="90%">
              <LineChart data={filtered}>
                <XAxis
                  dataKey="date"
                  tick={{ fontSize: 12 }}
                  tickFormatter={(value) => new Date(value).toLocaleDateString('en-GB')}
                />
                <YAxis
                  tick={{ fontSize: 12 }}
                  width={80}
                  tickFormatter={(value) => formatGbp(value)}
                />
                <Tooltip
                  labelFormatter={(value) => new Date(value).toLocaleDateString('en-GB')}
                  formatter={(value) => formatGbp(Number(value))}
                />
                <Legend />
                <Line
                  type="monotone"
                  dataKey="investedGbp"
                  name="Invested value"
                  stroke="var(--color-chart-1)"
                  strokeWidth={2}
                  dot={false}
                  connectNulls
                />
                <Line
                  type="monotone"
                  dataKey="marketGbp"
                  name="Market value"
                  stroke="var(--color-chart-2)"
                  strokeWidth={2}
                  dot={false}
                  connectNulls
                />
              </LineChart>
            </ResponsiveContainer>
          </>
        )}
      </CardContent>
    </Card>
  )
}
