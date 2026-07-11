import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { fetchInvestedValueSeries } from '@/lib/api'
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

function filterByPeriod(
  points: { date: string; valueGbp: string }[],
  period: PeriodKey,
) {
  if (period === 'all' || points.length === 0) return points

  const lastDate = new Date(points[points.length - 1].date)
  const cutoff = new Date(lastDate)
  cutoff.setDate(cutoff.getDate() - PERIOD_DAYS[period])

  const visible = points.filter((p) => new Date(p.date) >= cutoff)

  // Carry the last known cumulative value into the window so short periods
  // with no BUY/SELL activity don't render as empty.
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
  const { data, isLoading, error } = useQuery({
    queryKey: ['invested-value-series'],
    queryFn: fetchInvestedValueSeries,
  })
  const [period, setPeriod] = useState<PeriodKey>('all')

  const filtered = useMemo(() => filterByPeriod(data ?? [], period), [data, period])

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>Cumulative invested value</CardTitle>
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
        {error && (
          <p className="text-destructive text-sm">Failed to load invested value.</p>
        )}
        {data && filtered.length === 0 && (
          <p className="text-muted-foreground text-sm">No data for this period.</p>
        )}
        {data && filtered.length > 0 && (
          <ResponsiveContainer width="100%" height="100%">
            <LineChart
              data={filtered.map((point) => ({
                date: point.date,
                valueGbp: Number(point.valueGbp),
              }))}
            >
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
              <Line
                type="monotone"
                dataKey="valueGbp"
                stroke="var(--color-primary)"
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        )}
      </CardContent>
    </Card>
  )
}
