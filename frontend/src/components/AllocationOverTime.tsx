import { useMemo } from 'react'
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { allocationKey, useAllocationSeries } from '@/lib/allocations'
import type { AllocationGroupBy } from '@/lib/api'
import { formatGbp } from '@/lib/format'
import { CHART_COLORS } from '@/lib/sector'

const HEIGHT = 300
// Seven palette hues plus a neutral band for everything folded into it.
const MAX_SERIES = CHART_COLORS.length - 1
const FOLDED_KEY = '__folded__'
const FOLDED_COLOR = 'var(--color-muted-foreground)'

type Series = { key: string; label: string; color: string }
type Point = { date: string; total: number } & Record<string, number | string>

const monthFormat = new Intl.DateTimeFormat('en-GB', { month: 'short', year: '2-digit' })
const longFormat = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })

function OverTimeTooltip({
  active,
  payload,
  series,
}: {
  active?: boolean
  payload?: { payload: Point }[]
  series: Series[]
}) {
  const point = payload?.[0]?.payload
  if (!active || !point) return null
  return (
    <div className="bg-popover text-popover-foreground min-w-48 rounded-md border px-3 py-2 text-xs shadow-md">
      <div className="text-muted-foreground mb-1">{longFormat.format(new Date(point.date))}</div>
      {[...series].reverse().map((s) => {
        const value = Number(point[s.key] ?? 0)
        return (
          <div key={s.key} className="flex items-center gap-2 py-0.5">
            <span className="h-0.5 w-3 rounded-full" style={{ backgroundColor: s.color }} aria-hidden />
            <span className="font-semibold tabular-nums">
              {point.total > 0 ? `${((value / point.total) * 100).toFixed(1)}%` : '—'}
            </span>
            <span className="text-muted-foreground min-w-0 flex-1 truncate">{s.label}</span>
            <span className="text-muted-foreground tabular-nums">{formatGbp(value)}</span>
          </div>
        )
      })}
    </div>
  )
}

/**
 * How the mix of groups has shifted, month by month. Holdings are those
 * held today, classified as they are today.
 */
export function AllocationOverTime({
  dimension,
  dimensionLabel,
  parent,
}: {
  dimension: AllocationGroupBy
  dimensionLabel: string
  // Restricts to one sector or category's sub-groups when drilled in.
  parent: string | null
}) {
  const { data, isLoading, error, isPlaceholderData } = useAllocationSeries(dimension)

  const { series, points } = useMemo(() => {
    if (!data) return { series: [] as Series[], points: [] as Point[] }
    const indexes = data.groups
      .map((g, i) => ({ g, i }))
      .filter(({ g }) => parent === null || g.parent === parent)
    // Groups arrive sorted by latest value; the largest keep their own
    // band and palette slot, the rest fold into one neutral band.
    const fold = indexes.length > MAX_SERIES + 1
    const kept = fold ? indexes.slice(0, MAX_SERIES) : indexes
    const folded = fold ? indexes.slice(MAX_SERIES) : []

    const series: Series[] = kept.map(({ g }, slot) => ({
      key: allocationKey(g),
      label: g.parent && parent === null ? `${g.name} (${g.parent})` : g.name,
      color: CHART_COLORS[slot],
    }))
    if (folded.length > 0) {
      series.push({ key: FOLDED_KEY, label: `${folded.length} other ${dimensionLabel.toLowerCase()}s`, color: FOLDED_COLOR })
    }

    const points: Point[] = data.points.map((p) => {
      const point: Point = { date: p.date, total: 0 }
      for (const { g, i } of kept) {
        point[allocationKey(g)] = Number(p.valuesGbp[i])
      }
      if (folded.length > 0) {
        point[FOLDED_KEY] = folded.reduce((sum, { i }) => sum + Number(p.valuesGbp[i]), 0)
      }
      point.total = series.reduce((sum, s) => sum + Number(point[s.key] ?? 0), 0)
      return point
    })
    // Drop leading months with nothing yet held in this selection.
    const first = points.findIndex((p) => p.total > 0)
    return { series, points: first < 0 ? [] : points.slice(first) }
  }, [data, parent, dimensionLabel])

  const latest = points.at(-1)

  return (
    <Card className={isPlaceholderData ? 'opacity-50 transition-opacity' : 'transition-opacity'}>
      <CardHeader>
        <CardTitle>{dimensionLabel} mix over time</CardTitle>
        <p className="text-muted-foreground text-xs">
          Share of market value at each month end, for holdings held today as they're classified today
        </p>
      </CardHeader>
      <CardContent>
        {isLoading && <p className="text-muted-foreground text-sm">Loading…</p>}
        {!!error && <p className="text-destructive text-sm">Failed to load the history.</p>}
        {data && points.length < 2 && (
          <p className="text-muted-foreground text-sm">Not enough price history to show a trend yet.</p>
        )}
        {points.length >= 2 && (
          <>
            <div style={{ height: HEIGHT }}>
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={points} stackOffset="expand" margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
                  <CartesianGrid vertical={false} stroke="var(--color-border)" />
                  <XAxis
                    dataKey="date"
                    tickFormatter={(d: string) => monthFormat.format(new Date(d))}
                    tick={{ fontSize: 12, fill: 'var(--color-muted-foreground)' }}
                    axisLine={false}
                    tickLine={false}
                    minTickGap={32}
                  />
                  <YAxis
                    tickFormatter={(v: number) => `${Math.round(v * 100)}%`}
                    ticks={[0, 0.25, 0.5, 0.75, 1]}
                    tick={{ fontSize: 12, fill: 'var(--color-muted-foreground)' }}
                    axisLine={false}
                    tickLine={false}
                    width={44}
                  />
                  <Tooltip
                    content={<OverTimeTooltip series={series} />}
                    cursor={{ stroke: 'var(--color-muted-foreground)', strokeWidth: 1 }}
                  />
                  {series.map((s) => (
                    <Area
                      key={s.key}
                      // Straight segments between month ends; a smoothed
                      // curve would invent values between them.
                      type="linear"
                      dataKey={s.key}
                      stackId="mix"
                      // The 1px surface stroke keeps a gap between bands.
                      stroke="var(--color-card)"
                      strokeWidth={1}
                      fill={s.color}
                      fillOpacity={s.key === FOLDED_KEY ? 0.45 : 0.9}
                      isAnimationActive={false}
                      activeDot={false}
                    />
                  ))}
                </AreaChart>
              </ResponsiveContainer>
            </div>
            <ul className="mt-3 flex flex-wrap gap-x-4 gap-y-1.5 text-xs" aria-label="Legend">
              {series.map((s) => (
                <li key={s.key} className="flex items-center gap-1.5">
                  <span
                    className="size-2.5 rounded-[3px]"
                    style={{ backgroundColor: s.color, opacity: s.key === FOLDED_KEY ? 0.45 : 0.9 }}
                    aria-hidden
                  />
                  <span>{s.label}</span>
                  {latest && latest.total > 0 && (
                    <span className="text-muted-foreground tabular-nums">
                      {((Number(latest[s.key] ?? 0) / latest.total) * 100).toFixed(1)}%
                    </span>
                  )}
                </li>
              ))}
            </ul>
          </>
        )}
      </CardContent>
    </Card>
  )
}
