import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-react'
import { TableHead } from '@/components/ui/table'
import { cn } from '@/lib/utils'

export function SortableHead<K extends string>({
  label,
  sortKey,
  sort,
  onToggle,
  className,
}: {
  label: string
  sortKey: K
  sort: { key: K; dir: 'asc' | 'desc' } | null
  onToggle: (key: K) => void
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
