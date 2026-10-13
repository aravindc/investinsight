import { useMemo } from 'react'
import {
  Bar,
  BarChart,
  CartesianGrid,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { DividendStatusMarker } from '@/components/DividendStatusMarker'
import type { DividendCalendar, DividendStatus } from '@/lib/api'
import {
  monthDate,
  monthKey,
  monthsBetween,
  STATUS_META,
  STATUS_ORDER,
  sumAmounts,
} from '@/lib/dividend-calendar'
import { formatGbp, formatGbpCompact } from '@/lib/format'

const HEIGHT = 220
const shortMonth = new Intl.DateTimeFormat('en-GB', { month: 'short', year: '2-digit', timeZone: 'UTC' })
const longMonth = new Intl.DateTimeFormat('en-GB', { month: 'long', year: 'numeric', timeZone: 'UTC' })

type Row = { month: string; total: number } & Record<DividendStatus, number>

function MonthTooltip({ active, payload }: { active?: boolean; payload?: { payload: Row }[] }) {
  const row = payload?.[0]?.payload
  if (!active || !row) return null
  return (
    <div className="bg-popover text-popover-foreground min-w-44 rounded-md border px-3 py-2 text-xs shadow-md">
      <div className="text-sm font-semibold">{formatGbp(row.total)}</div>
      <div className="text-muted-foreground mb-1">{longMonth.format(monthDate(row.month))}</div>
      {STATUS_ORDER.filter((s) => row[s] > 0).map((s) => (
        <div key={s} className="flex items-center gap-2 py-0.5">
          <DividendStatusMarker status={s} />
          <span className="text-muted-foreground flex-1">{STATUS_META[s].label}</span>
          <span className="tabular-nums">{formatGbp(row[s])}</span>
        </div>
      ))}
    </div>
  )
}

/**
 * Dividend income per month: the past year as paid, the next as announced
 * and estimated, stacked by how certain each is.
 */
export function DividendIncomeByMonth({ calendar }: { calendar: DividendCalendar }) {
  const { rows, received, expected } = useMemo(() => {
    const byMonth = new Map<string, Row>(
      monthsBetween(calendar.from, calendar.to).map((m) => [
        m,
        { month: m, total: 0, paid: 0, announced: 0, estimated: 0 },
      ]),
    )
    for (const e of calendar.events) {
      const row = byMonth.get(monthKey(e.date))
      if (!row || e.amountGbp === null) continue
      row[e.status] += Number(e.amountGbp)
      row.total += Number(e.amountGbp)
    }
    const paid = calendar.events.filter((e) => e.status === 'paid')
    const upcoming = calendar.events.filter((e) => e.status !== 'paid')
    return { rows: [...byMonth.values()], received: sumAmounts(paid), expected: sumAmounts(upcoming) }
  }, [calendar])

  const currentMonth = monthKey(calendar.today)

  return (
    <div>
      <div className="flex flex-wrap gap-x-10 gap-y-2">
        <div>
          <div className="text-muted-foreground text-xs">Received, last 12 months</div>
          <div className="text-2xl font-semibold">{formatGbp(received)}</div>
        </div>
        <div>
          <div className="text-muted-foreground text-xs">Expected, next 12 months</div>
          <div className="text-2xl font-semibold">{formatGbp(expected)}</div>
        </div>
      </div>
      <div className="mt-4" style={{ height: HEIGHT }}>
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={rows} margin={{ top: 16, right: 4, bottom: 0, left: 0 }} barCategoryGap="18%">
            <CartesianGrid vertical={false} stroke="var(--color-border)" />
            <XAxis
              dataKey="month"
              tickFormatter={(m: string) => shortMonth.format(monthDate(m))}
              tick={{ fontSize: 11, fill: 'var(--color-muted-foreground)' }}
              axisLine={false}
              tickLine={false}
              interval="preserveStartEnd"
              minTickGap={8}
            />
            <YAxis
              tickFormatter={(v: number) => formatGbpCompact(v)}
              tick={{ fontSize: 11, fill: 'var(--color-muted-foreground)' }}
              axisLine={false}
              tickLine={false}
              width={56}
            />
            <ReferenceLine
              x={currentMonth}
              stroke="var(--color-muted-foreground)"
              label={{ value: 'Today', position: 'top', fontSize: 11, fill: 'var(--color-muted-foreground)' }}
            />
            <Tooltip content={<MonthTooltip />} cursor={{ fill: 'var(--color-muted)', opacity: 0.5 }} />
            {STATUS_ORDER.map((s, i) => (
              <Bar
                key={s}
                dataKey={s}
                stackId="income"
                fill={STATUS_META[s].fill}
                // A 1px surface line separates stacked segments.
                stroke="var(--color-card)"
                strokeWidth={1}
                radius={i === STATUS_ORDER.length - 1 || s === 'paid' ? [4, 4, 0, 0] : 0}
                maxBarSize={24}
                isAnimationActive={false}
              />
            ))}
          </BarChart>
        </ResponsiveContainer>
      </div>
      <ul className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs" aria-label="Legend">
        {STATUS_ORDER.map((s) => (
          <li key={s} className="flex items-center gap-1.5">
            <DividendStatusMarker status={s} />
            {STATUS_META[s].label}
          </li>
        ))}
      </ul>
    </div>
  )
}
