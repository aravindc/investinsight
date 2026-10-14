import { useQuery } from '@tanstack/react-query'
import { HelpCircle, Landmark } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useAccountScope } from '@/lib/account-scope'
import { fetchCashIsas, fetchHoldingsSummary, fetchTrading212Cash } from '@/lib/api'
import { formatGbp } from '@/lib/format'

/** Investment value plus Trading 212 cash, cash in pies and Cash ISAs. */
const SEGMENTS = {
  investments: { label: 'Investments', color: 'var(--color-chart-1)' },
  cash: { label: 'Cash', color: 'var(--color-chart-2)' },
}

function percent(part: number, total: number) {
  const p = (part / total) * 100
  return p > 0 && p < 1 ? '<1%' : `${Math.round(p)}%`
}

/**
 * Investments against cash as a share of the total, on one line with each
 * share labelled at its own end of the bar, so the card stays as short as
 * its neighbours.
 */
function SplitBar({ investments, cash, total }: { investments: number; cash: number; total: number }) {
  const parts = [
    { key: 'investments' as const, value: investments },
    { key: 'cash' as const, value: cash },
  ].filter((p) => p.value > 0)
  const [first, last] = [parts[0], parts.at(-1)!]
  const end = (p: (typeof parts)[number]) => (
    <span className="shrink-0 text-xs">
      <span className="text-muted-foreground">{SEGMENTS[p.key].label}</span>{' '}
      <span className="font-medium tabular-nums">{percent(p.value, total)}</span>
    </span>
  )

  return (
    <div className="mt-2.5 flex items-center gap-2">
      {end(first)}
      <div
        role="img"
        aria-label={parts.map((p) => `${SEGMENTS[p.key].label} ${percent(p.value, total)}`).join(', ')}
        // A 2px surface gap separates the segments.
        className="flex h-2 min-w-12 flex-1 gap-0.5 overflow-hidden rounded-full"
      >
        {parts.map((p) => (
          <div
            key={p.key}
            className="h-full min-w-1 first:rounded-l-full last:rounded-r-full"
            style={{ width: `${(p.value / total) * 100}%`, backgroundColor: SEGMENTS[p.key].color }}
          />
        ))}
      </div>
      {parts.length > 1 && end(last)}
    </div>
  )
}

export function TotalValueCard() {
  const { scopeParam } = useAccountScope()
  // Same query key as ValueCard, so the holdings summary is fetched once.
  const holdings = useQuery({
    queryKey: ['holdings-summary', scopeParam],
    queryFn: () => fetchHoldingsSummary(scopeParam!),
    enabled: !!scopeParam,
  })
  const cash = useQuery({
    queryKey: ['trading212-cash', scopeParam],
    queryFn: () => fetchTrading212Cash(scopeParam!),
    enabled: !!scopeParam,
    // The server caches Trading 212 cash for up to an hour; no need to
    // re-ask it on every visit.
    staleTime: 5 * 60_000,
  })

  // Same query keys as CashCard.
  const cashIsas = useQuery({
    queryKey: ['cash-isas', scopeParam],
    queryFn: () => fetchCashIsas(scopeParam!),
    enabled: !!scopeParam,
  })

  const isLoading = holdings.isLoading || cash.isLoading || cashIsas.isLoading
  const error = holdings.error ?? cash.error ?? cashIsas.error
  const h = holdings.data
  const c = cash.data
  const isa = cashIsas.data
  const investments = Number(h?.marketValueGbp ?? 0)
  const cashTotal =
    Number(c?.cashGbp ?? 0) + Number(c?.cashInPiesGbp ?? 0) + Number(isa?.totalValueGbp ?? 0)
  const total = h && c && isa ? investments + cashTotal : null

  return (
    <Card>
      <CardContent>
        <div className="flex items-center gap-2">
          <span className="bg-primary/10 text-primary flex size-8 items-center justify-center rounded-md">
            <Landmark className="size-4" />
          </span>
          <span className="text-sm font-medium">Total portfolio value</span>
          <Tooltip>
            <TooltipTrigger asChild>
              <HelpCircle className="text-muted-foreground size-3.5" />
            </TooltipTrigger>
            <TooltipContent className="max-w-xs">
              Investment value plus cash: uninvested cash, cash in pies and Cash ISAs. The Cash
              card breaks it down.
            </TooltipContent>
          </Tooltip>
        </div>

        {isLoading && <p className="text-muted-foreground mt-3 text-sm">Loading…</p>}
        {!!error && (
          <p className="text-destructive mt-3 text-sm">Failed to load portfolio value.</p>
        )}
        {total !== null && (
          <>
            <p className="mt-3 text-3xl font-semibold">{formatGbp(total)}</p>
            {total > 0 && <SplitBar investments={investments} cash={cashTotal} total={total} />}
          </>
        )}
      </CardContent>
    </Card>
  )
}
