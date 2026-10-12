import { Info } from 'lucide-react'
import { Link } from 'react-router-dom'
import { needsClassifying } from '@/lib/allocations'
import type { Allocation, AllocationGroupBy } from '@/lib/api'

const REASON: Record<AllocationGroupBy, string> = {
  sector: 'in the Other sector',
  'sub-sector': 'with no sub-sector',
  category: 'in the Other category',
  'sub-category': 'in the Other category',
}

/**
 * Flags holdings still to be classified for this dimension (the "Other"
 * catch-all, or no sub-sector) and links to where they can be classified.
 */
export function ClassificationCallout({
  groups,
  dimension,
}: {
  groups: Allocation[]
  dimension: AllocationGroupBy
}) {
  const flagged = groups.filter((g) => needsClassifying(g, dimension))
  if (flagged.length === 0) return null

  const holdings = flagged.reduce((n, g) => n + g.instrumentCount, 0)
  const share = flagged.reduce((n, g) => n + Number(g.sharePercent), 0)

  return (
    <div
      role="status"
      className="bg-muted/50 flex flex-wrap items-center gap-x-3 gap-y-1 rounded-lg border px-4 py-3 text-sm"
    >
      <Info className="text-muted-foreground size-4 shrink-0" aria-hidden />
      <span className="flex-1">
        <span className="font-medium">
          {holdings} {holdings === 1 ? 'holding' : 'holdings'} {REASON[dimension]}
        </span>
        <span className="text-muted-foreground"> · {share.toFixed(1)}% of market value, shown in grey</span>
      </span>
      <Link to="/admin?tab=categories" className="font-medium underline-offset-4 hover:underline">
        Classify in Admin
      </Link>
    </div>
  )
}
