import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { allocationKey, DIMENSIONS, useAllocationMatrix } from '@/lib/allocations'
import type { AllocationGroupBy, AllocationMatrix } from '@/lib/api'
import { formatGbp } from '@/lib/format'

type Basis = 'row' | 'portfolio'
type Cell = AllocationMatrix['cells'][number]

// Sequential, one hue: the share is mixed from the card surface (near zero)
// up to the full palette blue (100%), so low values recede in both themes.
const MIN_MIX = 12
function cellColor(percent: number) {
  const mix = MIN_MIX + (100 - MIN_MIX) * Math.min(Math.max(percent, 0), 100) / 100
  return `color-mix(in oklab, var(--color-chart-1) ${mix.toFixed(1)}%, var(--color-card))`
}
// White text once the fill is dark enough; ink below that.
const WHITE_TEXT_FROM = 55

function label(name: string, parent: string | null) {
  return parent ? `${name} (${parent})` : name
}

/**
 * How the current dimension's groups split across another dimension, e.g.
 * how much of each sector is held as stocks, ETFs or investment trusts.
 */
export function AllocationHeatmap({
  rowsBy,
  colsBy,
  parent,
}: {
  rowsBy: AllocationGroupBy
  colsBy: AllocationGroupBy
  // Restricts rows to one sector or category when drilled in.
  parent: string | null
}) {
  const { data, isLoading, error, isPlaceholderData } = useAllocationMatrix(rowsBy, colsBy)
  const [basis, setBasis] = useState<Basis>('row')
  const [inspected, setInspected] = useState<Cell | null>(null)

  const rowLabel = DIMENSIONS.find((d) => d.value === rowsBy)!.label
  const colLabel = DIMENSIONS.find((d) => d.value === colsBy)!.label

  const content = () => {
    if (isLoading) return <p className="text-muted-foreground text-sm">Loading…</p>
    if (error || !data) return <p className="text-destructive text-sm">Failed to load the breakdown.</p>

    const rowIndexes = data.rows
      .map((r, i) => ({ r, i }))
      .filter(({ r }) => parent === null || r.parent === parent)
    const byCell = new Map(data.cells.map((c) => [`${c.row}:${c.col}`, c]))
    const rowValue = new Map<number, number>()
    const colValue = new Map<number, number>()
    for (const c of data.cells) {
      rowValue.set(c.row, (rowValue.get(c.row) ?? 0) + Number(c.marketValueGbp))
      colValue.set(c.col, (colValue.get(c.col) ?? 0) + Number(c.marketValueGbp))
    }
    const total = Number(data.totals.marketValueGbp)
    const percentOf = (cell: Cell) =>
      basis === 'portfolio'
        ? Number(cell.sharePercent)
        : (Number(cell.marketValueGbp) / (rowValue.get(cell.row) || 1)) * 100
    // Shade relative to the largest cell shown, so a dominant column
    // doesn't wash every other cell out to the surface.
    const shown = rowIndexes.flatMap(({ i }) =>
      data.cols.map((_, j) => byCell.get(`${i}:${j}`)).filter((c): c is Cell => !!c),
    )
    const maxPercent = Math.max(...shown.map(percentOf), 1)

    return (
      <>
        <div className="overflow-x-auto">
          <table className="w-full border-separate border-spacing-0.5 text-xs">
            <thead>
              <tr>
                <th scope="col" className="text-muted-foreground px-2 py-1 text-left font-medium">
                  {rowLabel}
                </th>
                {data.cols.map((c) => (
                  <th
                    key={allocationKey(c)}
                    scope="col"
                    className="text-muted-foreground min-w-14 px-1 py-1 text-center align-bottom font-medium"
                  >
                    {label(c.name, c.parent)}
                  </th>
                ))}
                <th scope="col" className="text-muted-foreground px-2 py-1 text-right font-medium">
                  Of portfolio
                </th>
              </tr>
            </thead>
            <tbody>
              {rowIndexes.map(({ r, i }) => (
                <tr key={allocationKey(r)}>
                  <th scope="row" className="px-2 py-1 text-left font-medium whitespace-nowrap">
                    {r.name}
                    {r.parent && parent === null && (
                      <span className="text-muted-foreground font-normal"> · {r.parent}</span>
                    )}
                  </th>
                  {data.cols.map((c, j) => {
                    const cell = byCell.get(`${i}:${j}`)
                    if (!cell) {
                      return (
                        <td key={allocationKey(c)} className="text-muted-foreground/50 h-8 text-center">
                          ·
                        </td>
                      )
                    }
                    const pct = percentOf(cell)
                    const shade = (pct / maxPercent) * 100
                    const description = `${r.name} × ${c.name}: ${pct.toFixed(1)}% of ${
                      basis === 'row' ? r.name : 'portfolio'
                    }`
                    return (
                      <td
                        key={allocationKey(c)}
                        tabIndex={0}
                        aria-label={description}
                        onMouseEnter={() => setInspected(cell)}
                        onFocus={() => setInspected(cell)}
                        onMouseLeave={() => setInspected(null)}
                        onBlur={() => setInspected(null)}
                        className="focus-visible:ring-ring h-8 rounded-[4px] text-center tabular-nums outline-none focus-visible:ring-2"
                        style={{
                          backgroundColor: cellColor(shade),
                          color: shade >= WHITE_TEXT_FROM ? 'white' : 'var(--color-foreground)',
                        }}
                      >
                        {pct < 0.05 ? '<0.1%' : `${pct.toFixed(pct < 10 ? 1 : 0)}%`}
                      </td>
                    )
                  })}
                  <td className="px-2 text-right tabular-nums">
                    {total > 0 ? `${(((rowValue.get(i) ?? 0) / total) * 100).toFixed(1)}%` : '—'}
                  </td>
                </tr>
              ))}
            </tbody>
            {parent === null && (
              <tfoot>
                <tr>
                  <th scope="row" className="text-muted-foreground px-2 py-1 text-left font-medium">
                    Of portfolio
                  </th>
                  {data.cols.map((c, j) => (
                    <td key={allocationKey(c)} className="px-1 py-1 text-center tabular-nums">
                      {total > 0 ? `${(((colValue.get(j) ?? 0) / total) * 100).toFixed(1)}%` : '—'}
                    </td>
                  ))}
                  <td />
                </tr>
              </tfoot>
            )}
          </table>
        </div>

        <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
          <p className="text-muted-foreground min-h-4 text-xs" aria-live="polite">
            {inspected ? (
              <>
                <span className="text-foreground font-medium">
                  {data.rows[inspected.row].name} × {data.cols[inspected.col].name}
                </span>{' '}
                · {formatGbp(inspected.marketValueGbp)} · {inspected.instrumentCount}{' '}
                {inspected.instrumentCount === 1 ? 'holding' : 'holdings'} ·{' '}
                {Number(inspected.sharePercent).toFixed(1)}% of portfolio
              </>
            ) : (
              'Hover or focus a cell for its value.'
            )}
          </p>
          <div className="text-muted-foreground flex items-center gap-2 text-xs" aria-hidden>
            <span>0%</span>
            <span
              className="h-2 w-28 rounded-full"
              style={{
                background: `linear-gradient(to right, ${cellColor(0)}, ${cellColor(100)})`,
              }}
            />
            <span>{maxPercent.toFixed(0)}%</span>
          </div>
        </div>
      </>
    )
  }

  return (
    <Card className={isPlaceholderData ? 'opacity-50 transition-opacity' : 'transition-opacity'}>
      <CardHeader className="flex flex-row flex-wrap items-start justify-between gap-3">
        <div>
          <CardTitle>
            {rowLabel} × {colLabel}
          </CardTitle>
          <p className="text-muted-foreground mt-1 text-xs">
            Market value in each combination,{' '}
            {basis === 'row' ? `as a share of each ${rowLabel.toLowerCase()}` : 'as a share of the portfolio'}
          </p>
        </div>
        <Tabs value={basis} onValueChange={(v) => setBasis(v as Basis)}>
          <TabsList>
            <TabsTrigger value="row">% of {rowLabel.toLowerCase()}</TabsTrigger>
            <TabsTrigger value="portfolio">% of portfolio</TabsTrigger>
          </TabsList>
        </Tabs>
      </CardHeader>
      <CardContent>{content()}</CardContent>
    </Card>
  )
}
