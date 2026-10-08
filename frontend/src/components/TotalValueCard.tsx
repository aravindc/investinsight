import { useQuery } from '@tanstack/react-query'
import { HelpCircle, Landmark } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useAccountScope } from '@/lib/account-scope'
import { fetchHoldingsSummary, fetchTrading212Cash } from '@/lib/api'
import { formatGbp } from '@/lib/format'

/** Investment value plus Trading 212 cash and cash in pies. */
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
    staleTime: 60_000,
  })

  const isLoading = holdings.isLoading || cash.isLoading
  const error = holdings.error ?? cash.error
  const h = holdings.data
  const c = cash.data
  const total = h && c ? Number(h.marketValueGbp) + Number(c.cashGbp) + Number(c.cashInPiesGbp) : null

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
              Investment value plus uninvested cash, including cash sitting in pies. Cash is
              fetched live from Trading 212 for connected accounts.
            </TooltipContent>
          </Tooltip>
        </div>

        {isLoading && <p className="text-muted-foreground mt-3 text-sm">Loading…</p>}
        {!!error && (
          <p className="text-destructive mt-3 text-sm">Failed to load portfolio value.</p>
        )}
        {h && c && total !== null && (
          <>
            <p className="mt-3 text-3xl font-semibold">{formatGbp(total)}</p>
            <p className="text-muted-foreground mt-1 text-sm">
              {formatGbp(Number(c.cashGbp) + Number(c.cashInPiesGbp))} cash
              {Number(c.cashInPiesGbp) !== 0 && <> ({formatGbp(c.cashInPiesGbp)} in pies)</>}
            </p>
            {c.accountsExcluded.length > 0 && (
              <p className="text-destructive mt-1 text-xs">
                Cash not included for{' '}
                {c.accountsExcluded.map((a) => `${a.name} (${a.reason})`).join(', ')}.
              </p>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}
