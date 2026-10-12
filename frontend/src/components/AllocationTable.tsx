import { useMemo, useState } from 'react'
import { ChevronRight } from 'lucide-react'
import { SortableHead } from '@/components/SortableHead'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { allocationKey } from '@/lib/allocations'
import { sumMeasure, measureSpec, type Measure } from '@/lib/allocation-view'
import type { Allocation } from '@/lib/api'
import { formatGbp } from '@/lib/format'

type SortKey =
  | 'name'
  | 'instrumentCount'
  | 'marketValue'
  | 'share'
  | 'invested'
  | 'gain'
  | 'dividends'
  | 'yield'

type SortState = { key: SortKey; dir: 'asc' | 'desc' } | null

const SORT_VALUE: Record<Exclude<SortKey, 'name'>, (a: Allocation) => number | null> = {
  instrumentCount: (a) => a.instrumentCount,
  marketValue: (a) => Number(a.marketValueGbp),
  share: (a) => Number(a.sharePercent),
  invested: (a) => Number(a.investedGbp),
  gain: (a) => (a.gainGbp === null ? null : Number(a.gainGbp)),
  dividends: (a) => Number(a.trailingYearDividendsGbp),
  yield: (a) => (a.yieldPercent === null ? null : Number(a.yieldPercent)),
}

const MEASURE_SORT: Record<Measure, SortKey> = {
  'market-value': 'marketValue',
  invested: 'invested',
  gain: 'gain',
  dividends: 'dividends',
}

function percent(value: string | null, signed = false) {
  if (value === null) return '—'
  const n = Number(value)
  return `${signed && n > 0 ? '+' : ''}${n.toFixed(1)}%`
}

function GainCell({ gbp, pct }: { gbp: string | null; pct: string | null }) {
  if (gbp === null) return <span className="text-muted-foreground">N/A</span>
  const n = Number(gbp)
  const color = n >= 0 ? 'text-green-600 dark:text-green-400' : 'text-destructive'
  return (
    <div className={color}>
      <div className="font-medium">
        {n > 0 ? '+' : ''}
        {formatGbp(n)}
      </div>
      <div className="text-xs">{percent(pct, true)}</div>
    </div>
  )
}

/** Every figure per group; doubles as the table view of the charts. */
export function AllocationTable({
  groups,
  measure,
  dimensionLabel,
  onSelect,
  selectLabel,
}: {
  groups: Allocation[]
  measure: Measure
  dimensionLabel: string
  onSelect?: (group: Allocation) => void
  selectLabel?: string
}) {
  // Starts sorted by the selected measure; the parent remounts this on change.
  const [sort, setSort] = useState<SortState>({ key: MEASURE_SORT[measure], dir: 'desc' })

  const sorted = useMemo(() => {
    if (!sort) return groups
    const factor = sort.dir === 'asc' ? 1 : -1
    return [...groups].sort((a, b) => {
      if (sort.key === 'name') return a.name.localeCompare(b.name) * factor
      const av = SORT_VALUE[sort.key](a)
      const bv = SORT_VALUE[sort.key](b)
      // Missing values always sort last.
      if (av === null || bv === null) return av === bv ? 0 : av === null ? 1 : -1
      return (av - bv) * factor || a.name.localeCompare(b.name)
    })
  }, [groups, sort])

  const toggle = (key: SortKey) =>
    setSort((current) =>
      current?.key === key
        ? { key, dir: current.dir === 'asc' ? 'desc' : 'asc' }
        : { key, dir: key === 'name' ? 'asc' : 'desc' },
    )

  const total = (m: Measure) => sumMeasure(groups, measureSpec(m))
  const totalMarket = total('market-value') ?? 0
  const totalInvested = total('invested') ?? 0
  const totalGain = total('gain')
  const totalDividends = total('dividends') ?? 0
  const head = (label: string, key: SortKey, className = 'text-right') => (
    <SortableHead label={label} sortKey={key} sort={sort} onToggle={toggle} className={className} />
  )

  return (
    <Card>
      <CardHeader>
        <CardTitle>{dimensionLabel} breakdown</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              {head(dimensionLabel, 'name', '')}
              {head('Holdings', 'instrumentCount')}
              {head('Market value', 'marketValue')}
              {head('Share', 'share')}
              {head('Invested', 'invested')}
              {head('Gain', 'gain')}
              {head('Dividends (12m)', 'dividends')}
              {head('Yield', 'yield')}
              {onSelect && <TableHead className="w-8" />}
            </TableRow>
          </TableHeader>
          <TableBody>
            {sorted.map((g) => (
              <TableRow
                key={allocationKey(g)}
                className={onSelect ? 'cursor-pointer' : undefined}
                onClick={onSelect ? () => onSelect(g) : undefined}
              >
                <TableCell>
                  <div className={g.unclassified ? 'text-muted-foreground italic' : 'font-medium'}>
                    {g.name}
                  </div>
                  {g.parent && <div className="text-muted-foreground text-xs">{g.parent}</div>}
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {g.instrumentCount}
                  {g.unpricedCount > 0 && (
                    <div className="text-muted-foreground text-xs">{g.unpricedCount} unpriced</div>
                  )}
                </TableCell>
                <TableCell className="text-right font-medium tabular-nums">
                  {formatGbp(g.marketValueGbp)}
                </TableCell>
                <TableCell className="text-right tabular-nums">{percent(g.sharePercent)}</TableCell>
                <TableCell className="text-right tabular-nums">{formatGbp(g.investedGbp)}</TableCell>
                <TableCell className="text-right tabular-nums">
                  <GainCell gbp={g.gainGbp} pct={g.gainPercent} />
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {formatGbp(g.trailingYearDividendsGbp)}
                </TableCell>
                <TableCell className="text-right tabular-nums">{percent(g.yieldPercent)}</TableCell>
                {onSelect && (
                  <TableCell>
                    <button
                      type="button"
                      aria-label={`${selectLabel ?? 'Open'}: ${g.name}`}
                      className="text-muted-foreground hover:text-foreground rounded p-0.5"
                      onClick={(e) => {
                        e.stopPropagation()
                        onSelect(g)
                      }}
                    >
                      <ChevronRight className="size-4" />
                    </button>
                  </TableCell>
                )}
              </TableRow>
            ))}
          </TableBody>
          <TableFooter>
            <TableRow>
              <TableCell className="font-medium">Total</TableCell>
              <TableCell className="text-right tabular-nums">
                {groups.reduce((n, g) => n + g.instrumentCount, 0)}
              </TableCell>
              <TableCell className="text-right font-medium tabular-nums">
                {formatGbp(totalMarket)}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {percent(String(groups.reduce((n, g) => n + Number(g.sharePercent), 0)))}
              </TableCell>
              <TableCell className="text-right tabular-nums">{formatGbp(totalInvested)}</TableCell>
              <TableCell className="text-right tabular-nums">
                <GainCell
                  gbp={totalGain === null ? null : String(totalGain)}
                  pct={totalGain === null || totalInvested === 0 ? null : String((totalGain / totalInvested) * 100)}
                />
              </TableCell>
              <TableCell className="text-right tabular-nums">{formatGbp(totalDividends)}</TableCell>
              <TableCell className="text-right tabular-nums">
                {totalGain === null || totalMarket === 0
                  ? '—'
                  : percent(String((totalDividends / totalMarket) * 100))}
              </TableCell>
              {onSelect && <TableCell />}
            </TableRow>
          </TableFooter>
        </Table>
      </CardContent>
    </Card>
  )
}
