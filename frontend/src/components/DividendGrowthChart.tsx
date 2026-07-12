import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Bar,
  BarChart,
  Legend,
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

const YEAR_COLORS = [
  'var(--color-chart-1)',
  'var(--color-chart-2)',
  'var(--color-chart-3)',
  'var(--color-chart-4)',
  'var(--color-chart-5)',
  'var(--color-chart-6)',
  'var(--color-chart-7)',
  'var(--color-chart-8)',
]

type MonthRow = Record<string, number | string> & { month: string }

function toByMonthRows(
  months: { year: number; month: number; totalGbp: string }[],
): { rows: MonthRow[]; years: number[] } {
  const years = Array.from(new Set(months.map((m) => m.year))).sort()

  const rows: MonthRow[] = MONTH_LABELS.map((label) => ({ month: label }))
  for (const m of months) {
    rows[m.month - 1][String(m.year)] = Number(m.totalGbp)
  }

  return { rows, years }
}

export function DividendGrowthChart() {
  const { scopeParam } = useAccountScope()
  const { data, isLoading, error } = useQuery({
    queryKey: ['monthly-dividends', scopeParam],
    queryFn: () => fetchMonthlyDividends(scopeParam!),
    enabled: !!scopeParam,
  })

  const { rows, years } = useMemo(() => toByMonthRows(data ?? []), [data])

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-1.5">
          Dividend growth
          <InfoTooltip>
            <TooltipTrigger asChild>
              <HelpCircle className="text-muted-foreground size-3.5" />
            </TooltipTrigger>
            <TooltipContent>
              Dividend income by calendar month, compared year over year.
            </TooltipContent>
          </InfoTooltip>
        </CardTitle>
      </CardHeader>
      <CardContent className="h-72">
        {isLoading && <p className="text-muted-foreground text-sm">Loading…</p>}
        {error && (
          <p className="text-destructive text-sm">Failed to load dividend history.</p>
        )}
        {!isLoading && !error && years.length === 0 && (
          <p className="text-muted-foreground text-sm">No dividends received yet.</p>
        )}
        {!isLoading && years.length > 0 && (
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={rows}>
              <XAxis dataKey="month" tick={{ fontSize: 12 }} />
              <YAxis tick={{ fontSize: 12 }} width={50} tickFormatter={(v) => formatGbp(v)} />
              <Tooltip formatter={(value) => formatGbp(Number(value))} />
              <Legend />
              {years.map((year, idx) => (
                <Bar
                  key={year}
                  dataKey={String(year)}
                  name={String(year)}
                  fill={YEAR_COLORS[idx % YEAR_COLORS.length]}
                  radius={[4, 4, 0, 0]}
                />
              ))}
            </BarChart>
          </ResponsiveContainer>
        )}
      </CardContent>
    </Card>
  )
}
