import { cn } from '@/lib/utils'
import { STATUS_META } from '@/lib/dividend-calendar'
import type { DividendStatus } from '@/lib/api'

/** Paid ●, announced ◆, estimated ○: shape and shade both carry status. */
export function DividendStatusMarker({
  status,
  className,
}: {
  status: DividendStatus
  className?: string
}) {
  const fill = STATUS_META[status].fill
  return (
    <span
      aria-hidden
      className={cn(
        'inline-block size-2 shrink-0',
        status === 'announced' ? 'rotate-45 rounded-[1px]' : 'rounded-full',
        className,
      )}
      style={
        status === 'estimated'
          ? { boxShadow: 'inset 0 0 0 1.5px var(--color-chart-1)' }
          : { backgroundColor: fill }
      }
    />
  )
}
