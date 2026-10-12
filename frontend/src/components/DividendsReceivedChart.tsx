import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Area,
  AreaChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Tooltip as InfoTooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { HelpCircle } from 'lucide-react'
import { useAccountScope } from '@/lib/account-scope'
import { fetchMonthlyDividends } from '@/lib/api'
import { formatGbp } from '@/lib/format'

const MONTH_LABELS = [
  'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
  'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec',
]

type Point = {
  label: string
  cumulativeGbp: number
}

// Last 12 calendar months of data (not the full lifetime history), so the
// chart reflects trailing-year dividend income rather than the full history
// since inception. The running total still carries forward the lifetime
// total accrued before the window, rather than resetting to 0, so month 1
// starts from cumulative-to-date instead of looking like income started
// from scratch.
function lastTwelveMonths(
  months: { year: number; month: number; totalGbp: string }[],
): { carriedGbp: number; months: { year: number; month: number; totalGbp: string }[] } {
  const windowed = months.slice(-12)
  const carried = months
    .slice(0, months.length - windowed.length)
    .reduce((sum, m) => sum + Number(m.totalGbp), 0)
  return { carriedGbp: carried, months: windowed }
}

function toCumulativeSeries(
  months: { year: number; month: number; totalGbp: string }[],
  carriedGbp: number,
): Point[] {
  let running = carriedGbp
  return months.map((m) => {
    running += Number(m.totalGbp)
    return { label: `${MONTH_LABELS[m.month - 1]} ${String(m.year).slice(2)}`, cumulativeGbp: running }
  })
}

export function DividendsReceivedChart() {
  const { scopeParam } = useAccountScope()
  const { data, isLoading, error } = useQuery({
    queryKey: ['monthly-dividends', scopeParam],
    queryFn: () => fetchMonthlyDividends(scopeParam!),
    enabled: !!scopeParam,
  })

  const { carriedGbp, months } = useMemo(() => lastTwelveMonths(data ?? []), [data])
  const series = useMemo(() => toCumulativeSeries(months, carriedGbp), [months, carriedGbp])
  const total = series.at(-1)?.cumulativeGbp ?? 0

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="flex items-center gap-1.5">
          Dividends received
          <InfoTooltip>
            <TooltipTrigger asChild>
              <HelpCircle className="text-muted-foreground size-3.5" />
            </TooltipTrigger>
            <TooltipContent>
              Cumulative dividend income received to date, shown over the last 12 months.
            </TooltipContent>
          </InfoTooltip>
        </CardTitle>
        {!isLoading && !error && series.length > 0 && (
          <span className="bg-primary/10 text-primary rounded-md px-3 py-1 text-sm font-semibold">
            {formatGbp(total)}
          </span>
        )}
      </CardHeader>
      <CardContent className="h-72">
        {isLoading && <p className="text-muted-foreground text-sm">Loading…</p>}
        {error && (
          <p className="text-destructive text-sm">Failed to load dividend history.</p>
        )}
        {!isLoading && !error && series.length === 0 && (
          <p className="text-muted-foreground text-sm">No dividends received yet.</p>
        )}
        {!isLoading && series.length > 0 && (
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={series}>
              <defs>
                <linearGradient id="dividendsReceivedFill" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="var(--color-chart-7)" stopOpacity={0.35} />
                  <stop offset="100%" stopColor="var(--color-chart-7)" stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis
                dataKey="label"
                tick={{ fontSize: 12 }}
                interval="preserveStartEnd"
              />
              <YAxis tick={{ fontSize: 12 }} width={50} tickFormatter={(v) => formatGbp(v)} />
              <Tooltip formatter={(value) => formatGbp(Number(value))} />
              <Area
                type="monotone"
                dataKey="cumulativeGbp"
                name="Cumulative dividends"
                stroke="var(--color-chart-7)"
                strokeWidth={2}
                fill="url(#dividendsReceivedFill)"
                dot={false}
              />
            </AreaChart>
          </ResponsiveContainer>
        )}
      </CardContent>
    </Card>
  )
}
