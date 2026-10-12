import { TriangleAlert } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { allocationKey } from '@/lib/allocations'
import type { Allocation, AllocationFigures } from '@/lib/api'

// Common rules of thumb; past these a holding or group gets a warning.
const HOLDING_LIMIT = 10 // % of the selection in one holding
// % of the selection in one sector or sub-sector. Not applied to asset
// categories, where most of a portfolio in one (e.g. equities) is normal.
const GROUP_LIMIT = 30
const DEPENDENCE_LIMIT = 50 // % of a group in its largest holding
const MAX_DEPENDENT_LISTED = 5

function Warning({ children }: { children: React.ReactNode }) {
  return (
    <span className="mt-1 flex items-center gap-1 text-xs font-medium text-amber-700 dark:text-amber-400">
      <TriangleAlert className="size-3.5 shrink-0" aria-hidden />
      {children}
    </span>
  )
}

function Figure({
  label,
  value,
  detail,
  warning,
}: {
  label: string
  value: string
  detail: string
  warning?: string
}) {
  return (
    <div className="min-w-0">
      <div className="text-muted-foreground text-xs">{label}</div>
      <div className="mt-0.5 text-2xl font-semibold">{value}</div>
      <div className="text-muted-foreground truncate text-xs" title={detail}>
        {detail}
      </div>
      {warning && <Warning>{warning}</Warning>}
    </div>
  )
}

/**
 * How concentrated the current selection is: across holdings overall, and
 * within the groups of the current dimension.
 */
export function ConcentrationCard({
  groups,
  totals,
  dimensionLabel,
  warnOnGroupSize,
}: {
  groups: Allocation[]
  totals: AllocationFigures
  dimensionLabel: string
  warnOnGroupSize: boolean
}) {
  const effective = totals.effectiveHoldings === null ? null : Number(totals.effectiveHoldings)
  const largest = totals.largestHolding
  const largestShare = largest ? Number(largest.sharePercent) : null
  const topGroup = [...groups].sort((a, b) => Number(b.sharePercent) - Number(a.sharePercent))[0]
  const topGroupShare = topGroup ? Number(topGroup.sharePercent) : null
  const dependent = groups
    .filter(
      (g) =>
        g.instrumentCount > 1 &&
        g.largestHolding !== null &&
        Number(g.largestHolding.sharePercent) >= DEPENDENCE_LIMIT,
    )
    .sort((a, b) => Number(b.sharePercent) - Number(a.sharePercent))

  return (
    <Card className="h-full">
      <CardHeader>
        <CardTitle>Concentration</CardTitle>
        <p className="text-muted-foreground text-xs">
          By market value. Effective holdings is how many equal-sized holdings would be as
          diversified.
        </p>
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="grid grid-cols-3 gap-4">
          <Figure
            label="Effective holdings"
            value={effective === null ? '—' : effective.toFixed(effective < 10 ? 1 : 0)}
            detail={`of ${totals.instrumentCount} holdings`}
          />
          <Figure
            label="Largest holding"
            value={largestShare === null ? '—' : `${largestShare.toFixed(1)}%`}
            detail={largest ? `${largest.ticker} · ${largest.name}` : '—'}
            warning={
              largestShare !== null && largestShare > HOLDING_LIMIT ? `Over ${HOLDING_LIMIT}%` : undefined
            }
          />
          <Figure
            label={`Largest ${dimensionLabel.toLowerCase()}`}
            value={topGroupShare === null ? '—' : `${topGroupShare.toFixed(1)}%`}
            detail={topGroup ? topGroup.name : '—'}
            warning={
              warnOnGroupSize && topGroupShare !== null && topGroupShare > GROUP_LIMIT
                ? `Over ${GROUP_LIMIT}%`
                : undefined
            }
          />
        </div>

        <div>
          <div className="text-sm font-medium">Reliant on one holding</div>
          <p className="text-muted-foreground text-xs">
            {dimensionLabel} groups with {DEPENDENCE_LIMIT}% or more in their largest holding
          </p>
          {dependent.length === 0 ? (
            <p className="text-muted-foreground mt-2 text-sm">None.</p>
          ) : (
            <ul className="mt-2 divide-y text-sm">
              {dependent.slice(0, MAX_DEPENDENT_LISTED).map((g) => (
                <li key={allocationKey(g)} className="flex items-baseline justify-between gap-3 py-1.5">
                  <span className="min-w-0 truncate">
                    <span className="font-medium">{g.name}</span>
                    <span className="text-muted-foreground"> · {g.largestHolding!.ticker}</span>
                  </span>
                  <span className="shrink-0 tabular-nums">
                    {Number(g.largestHolding!.sharePercent).toFixed(0)}%
                    <span className="text-muted-foreground"> of {g.instrumentCount}</span>
                  </span>
                </li>
              ))}
              {dependent.length > MAX_DEPENDENT_LISTED && (
                <li className="text-muted-foreground py-1.5 text-xs">
                  and {dependent.length - MAX_DEPENDENT_LISTED} more
                </li>
              )}
            </ul>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
