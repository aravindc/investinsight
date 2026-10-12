import { Info } from 'lucide-react'
import { Link } from 'react-router-dom'
import type { Allocation } from '@/lib/api'

/**
 * Flags holdings with no sub-sector, which the sub-sector views can only
 * show as "Unclassified", and links to where they can be classified.
 */
export function UnclassifiedCallout({ groups }: { groups: Allocation[] }) {
  const unclassified = groups.filter((g) => g.unclassified)
  if (unclassified.length === 0) return null

  const holdings = unclassified.reduce((n, g) => n + g.instrumentCount, 0)
  const share = unclassified.reduce((n, g) => n + Number(g.sharePercent), 0)

  return (
    <div
      role="status"
      className="bg-muted/50 flex flex-wrap items-center gap-x-3 gap-y-1 rounded-lg border px-4 py-3 text-sm"
    >
      <Info className="text-muted-foreground size-4 shrink-0" aria-hidden />
      <span className="flex-1">
        <span className="font-medium">
          {holdings} {holdings === 1 ? 'holding has' : 'holdings have'} no sub-sector
        </span>
        <span className="text-muted-foreground">
          {' '}
          · {share.toFixed(1)}% of market value, shown as Unclassified
        </span>
      </span>
      <Link
        to="/admin?tab=categories"
        className="font-medium underline-offset-4 hover:underline"
      >
        Classify in Admin
      </Link>
    </div>
  )
}
