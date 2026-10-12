import { useSearchParams } from 'react-router-dom'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  allocationKey,
  DIMENSIONS,
  isDimension,
  useAllocations,
  useGroupColor,
} from '@/lib/allocations'
import type { Allocation, AllocationFigures, AllocationGroupBy } from '@/lib/api'
import { formatGbp } from '@/lib/format'

type Measure = 'market-value' | 'invested' | 'gain' | 'dividends'

type MeasureSpec = {
  value: Measure
  label: string
  amount: (f: AllocationFigures) => string | null
  // Secondary percentage beside each amount, and what it means.
  percentLabel: string
  percent: (a: Allocation, totals: AllocationFigures) => number | null
}

function ratio(part: string, whole: string): number | null {
  const w = Number(whole)
  return w === 0 ? null : (Number(part) / w) * 100
}

function nullableNumber(value: string | null): number | null {
  return value === null ? null : Number(value)
}

const MEASURES: MeasureSpec[] = [
  {
    value: 'market-value',
    label: 'Market value',
    amount: (f) => f.marketValueGbp,
    percentLabel: 'of value',
    percent: (a) => Number(a.sharePercent),
  },
  {
    value: 'invested',
    label: 'Invested',
    amount: (f) => f.investedGbp,
    percentLabel: 'of invested',
    percent: (a, totals) => ratio(a.investedGbp, totals.investedGbp),
  },
  {
    value: 'gain',
    label: 'Gain',
    amount: (f) => f.gainGbp,
    percentLabel: 'return',
    percent: (a) => nullableNumber(a.gainPercent),
  },
  {
    value: 'dividends',
    label: 'Dividends (12m)',
    amount: (f) => f.trailingYearDividendsGbp,
    percentLabel: 'yield',
    percent: (a) => nullableNumber(a.yieldPercent),
  },
]

function isMeasure(value: string | null): value is Measure {
  return MEASURES.some((m) => m.value === value)
}

/** Reads and writes the selected dimension and measure in the URL. */
function useBreakdownParams() {
  const [params, setParams] = useSearchParams()
  const rawDimension = params.get('dimension')
  const rawMeasure = params.get('measure')
  const dimension: AllocationGroupBy = isDimension(rawDimension) ? rawDimension : 'sector'
  const measure: Measure = isMeasure(rawMeasure) ? rawMeasure : 'market-value'

  const update = (key: 'dimension' | 'measure', value: string) =>
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        next.set(key, value)
        return next
      },
      { replace: true },
    )

  return {
    dimension,
    measure,
    setDimension: (d: AllocationGroupBy) => update('dimension', d),
    setMeasure: (m: Measure) => update('measure', m),
  }
}

function formatPercent(value: number | null): string {
  return value === null ? '—' : `${value.toFixed(1)}%`
}

function formatAmount(value: string | null, signed: boolean): string {
  if (value === null) return 'N/A'
  const n = Number(value)
  return signed && n > 0 ? `+${formatGbp(n)}` : formatGbp(n)
}

function BreakdownList({
  dimension,
  measure,
}: {
  dimension: AllocationGroupBy
  measure: MeasureSpec
}) {
  const { data, isLoading, error } = useAllocations(dimension)
  const colorFor = useGroupColor(dimension)

  if (isLoading) return <p className="text-muted-foreground text-sm">Loading…</p>
  if (error) return <p className="text-destructive text-sm">Failed to load allocations.</p>
  if (!data || data.groups.length === 0) {
    return <p className="text-muted-foreground text-sm">No holdings to break down.</p>
  }

  const rows = data.groups
    .map((group) => ({ group, amount: nullableNumber(measure.amount(group)) }))
    // Groups without a value (unpriced gain) sink to the bottom.
    .sort((a, b) => (b.amount ?? -Infinity) - (a.amount ?? -Infinity))
  const maxAbs = Math.max(...rows.map((r) => Math.abs(r.amount ?? 0)), 0)
  const signed = measure.value === 'gain'

  return (
    <div>
      <ul className="divide-y">
        {rows.map(({ group, amount }) => {
          const width = amount === null || maxAbs === 0 ? 0 : (Math.abs(amount) / maxAbs) * 100
          return (
            <li key={allocationKey(group)} className="py-2.5">
              <div className="flex items-center gap-3 text-sm">
                <span
                  aria-hidden
                  className="size-2.5 shrink-0 rounded-full"
                  style={{ backgroundColor: colorFor(group) }}
                />
                <span className="min-w-0 flex-1 truncate">
                  <span className={group.unclassified ? 'text-muted-foreground italic' : 'font-medium'}>
                    {group.name}
                  </span>
                  {group.parent !== null && (
                    <span className="text-muted-foreground"> · {group.parent}</span>
                  )}
                  <span className="text-muted-foreground text-xs">
                    {' '}
                    ({group.instrumentCount})
                  </span>
                </span>
                <span className="font-medium tabular-nums">
                  {formatAmount(measure.amount(group), signed)}
                </span>
                <span className="text-muted-foreground w-24 text-right text-xs tabular-nums">
                  {formatPercent(measure.percent(group, data.totals))} {measure.percentLabel}
                </span>
              </div>
              <div aria-hidden className="bg-muted mt-1.5 ml-5.5 h-1.5 overflow-hidden rounded-full">
                <div
                  className="h-full rounded-full"
                  style={{
                    width: `${width}%`,
                    backgroundColor: colorFor(group),
                    opacity: amount !== null && amount < 0 ? 0.45 : 1,
                  }}
                />
              </div>
            </li>
          )
        })}
      </ul>
      <div className="mt-2 flex justify-between border-t pt-3 text-sm">
        <span className="text-muted-foreground">
          Total · {data.totals.instrumentCount} holdings
          {data.totals.unpricedCount > 0 && <> · {data.totals.unpricedCount} unpriced</>}
        </span>
        <span className="font-semibold tabular-nums">
          {formatAmount(measure.amount(data.totals), signed)}
        </span>
      </div>
    </div>
  )
}

/** Portfolio breakdown by sector, sub-sector, category or sub-category. */
export function AllocationBreakdown() {
  const { dimension, measure, setDimension, setMeasure } = useBreakdownParams()
  const measureSpec = MEASURES.find((m) => m.value === measure)!

  return (
    <Card>
      <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-3">
        <CardTitle>Breakdown</CardTitle>
        <div className="flex flex-wrap items-center gap-3">
          <Tabs value={dimension} onValueChange={(v) => isDimension(v) && setDimension(v)}>
            <TabsList>
              {DIMENSIONS.map((d) => (
                <TabsTrigger key={d.value} value={d.value}>
                  {d.label}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
          <Select value={measure} onValueChange={(v) => isMeasure(v) && setMeasure(v)}>
            <SelectTrigger className="w-40" aria-label="Measure">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {MEASURES.map((m) => (
                <SelectItem key={m.value} value={m.value}>
                  {m.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </CardHeader>
      <CardContent>
        <BreakdownList dimension={dimension} measure={measureSpec} />
      </CardContent>
    </Card>
  )
}
