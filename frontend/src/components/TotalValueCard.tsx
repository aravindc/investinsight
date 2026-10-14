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
  const isaValue = Number(isa?.totalValueGbp ?? 0)
  const total =
    h && c && isa
      ? Number(h.marketValueGbp) + Number(c.cashGbp) + Number(c.cashInPiesGbp) + isaValue
      : null

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
        {total !== null && <p className="mt-3 text-3xl font-semibold">{formatGbp(total)}</p>}
      </CardContent>
    </Card>
  )
}
