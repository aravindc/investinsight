import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-react'
import { TableHead } from '@/components/ui/table'
import type { SortKey, SortState } from '@/lib/use-name-count-sort'
import { cn } from '@/lib/utils'

export function SortableHead({
  label,
  sortKey,
  sort,
  onToggle,
  className,
}: {
  label: string
  sortKey: SortKey
  sort: SortState
  onToggle: (key: SortKey) => void
  className?: string
}) {
  const active = sort?.key === sortKey
  return (
    <TableHead
      className={className}
      aria-sort={active ? (sort.dir === 'asc' ? 'ascending' : 'descending') : 'none'}
    >
      <button
        type="button"
        onClick={() => onToggle(sortKey)}
        className={cn(
          'hover:text-foreground inline-flex items-center gap-1',
          className?.includes('text-right') && 'flex-row-reverse',
        )}
      >
        {label}
        {active && sort.dir === 'asc' && <ArrowUp className="size-3" />}
        {active && sort.dir === 'desc' && <ArrowDown className="size-3" />}
        {!active && <ArrowUpDown className="text-muted-foreground/50 size-3" />}
      </button>
    </TableHead>
  )
}
