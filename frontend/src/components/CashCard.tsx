import { useQuery } from '@tanstack/react-query'
import { Banknote, HelpCircle } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useAccountScope } from '@/lib/account-scope'
import { fetchCashIsas, fetchTrading212Cash } from '@/lib/api'
import { formatGbp, formatGbpWhole } from '@/lib/format'

/** Uninvested cash, cash held in pies, and Cash ISAs. */
export function CashCard() {
  const { scopeParam } = useAccountScope()
  // Same query keys as TotalValueCard, so each is fetched once.
  const cash = useQuery({
    queryKey: ['trading212-cash', scopeParam],
    queryFn: () => fetchTrading212Cash(scopeParam!),
    enabled: !!scopeParam,
    // The server caches Trading 212 cash for up to an hour.
    staleTime: 5 * 60_000,
  })
  const cashIsas = useQuery({
    queryKey: ['cash-isas', scopeParam],
    queryFn: () => fetchCashIsas(scopeParam!),
    enabled: !!scopeParam,
  })

  const c = cash.data
  const isa = cashIsas.data
  const isas = isa?.cashIsas ?? []
  // A single rate is shown when every Cash ISA in scope pays the same.
  const rates = [...new Set(isas.map((i) => i.currentRatePercent))]
  const total =
    c && isa ? Number(c.cashGbp) + Number(c.cashInPiesGbp) + Number(isa.totalValueGbp) : null
  // Parts that are zero are left out of the breakdown line, which is in
  // whole pounds so it fits on one line like the other cards' detail.
  const parts =
    c && isa
      ? [
          { label: 'cash', value: c.cashGbp },
          { label: 'in pies', value: c.cashInPiesGbp },
          { label: 'ISA', value: isa.totalValueGbp },
        ].filter((p) => Number(p.value) !== 0)
      : []
  const asOf = c?.oldestFetchedAt
    ? new Date(c.oldestFetchedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    : null

  return (
    <Card>
      <CardContent>
        <div className="flex items-center gap-2">
          <span className="bg-primary/10 text-primary flex size-8 items-center justify-center rounded-md">
            <Banknote className="size-4" />
          </span>
          <span className="text-sm font-medium">Cash</span>
          <Tooltip>
            <TooltipTrigger asChild>
              <HelpCircle className="text-muted-foreground size-3.5" />
            </TooltipTrigger>
            <TooltipContent className="max-w-xs space-y-1">
              <p>
                Uninvested cash and cash in pies come from Trading 212 for connected accounts,
                refreshed at most hourly and after each sync{asOf && ` (last at ${asOf})`}.
              </p>
              <p>
                Cash ISAs include interest credited at each month end
                {isa && isas.length > 0 && (
                  <>
                    {' '}
                    ({rates.length === 1 ? `${rates[0]}%, ` : ''}
                    {formatGbp(isa.totalAccruedThisMonthGbp)} accrued this month)
                  </>
                )}
                . Set balances and rates on the Admin page.
              </p>
            </TooltipContent>
          </Tooltip>
        </div>

        {(cash.isLoading || cashIsas.isLoading) && (
          <p className="text-muted-foreground mt-3 text-sm">Loading…</p>
        )}
        {!!(cash.error ?? cashIsas.error) && (
          <p className="text-destructive mt-3 text-sm">Failed to load cash.</p>
        )}
        {total !== null && (
          <>
            <p className="mt-3 text-3xl font-semibold">{formatGbp(total)}</p>
            <p className="text-muted-foreground mt-1 text-sm">
              {parts.length === 0
                ? 'No cash'
                : parts.map((p, i) => (
                    <span key={p.label}>
                      {i > 0 && ' · '}
                      {formatGbpWhole(p.value)} {p.label}
                    </span>
                  ))}
            </p>
            {c && c.accountsExcluded.length > 0 && (
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
