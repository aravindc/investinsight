import { useQuery } from '@tanstack/react-query'
import { HelpCircle, PiggyBank } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useAccountScope } from '@/lib/account-scope'
import { fetchCashIsas } from '@/lib/api'
import { formatGbp } from '@/lib/format'

export function CashIsaCard() {
  const { scopeParam } = useAccountScope()
  const { data, isLoading, error } = useQuery({
    queryKey: ['cash-isas', scopeParam],
    queryFn: () => fetchCashIsas(scopeParam!),
    enabled: !!scopeParam,
  })

  const isas = data?.cashIsas ?? []
  // A single rate is shown when every Cash ISA in scope pays the same.
  const rates = [...new Set(isas.map((i) => i.currentRatePercent))]

  return (
    <Card>
      <CardContent>
        <div className="flex items-center gap-2">
          <span className="bg-primary/10 text-primary flex size-8 items-center justify-center rounded-md">
            <PiggyBank className="size-4" />
          </span>
          <span className="text-sm font-medium">Cash ISA</span>
          <Tooltip>
            <TooltipTrigger asChild>
              <HelpCircle className="text-muted-foreground size-3.5" />
            </TooltipTrigger>
            <TooltipContent className="max-w-xs">
              Cash ISA balance including interest credited at each month end. Interest accrues
              daily and is added at the end of the month. Set balances and rates on the Admin page.
            </TooltipContent>
          </Tooltip>
        </div>

        {isLoading && <p className="text-muted-foreground mt-3 text-sm">Loading…</p>}
        {!!error && <p className="text-destructive mt-3 text-sm">Failed to load Cash ISA.</p>}
        {data && isas.length === 0 && (
          <p className="text-muted-foreground mt-3 text-sm">No Cash ISA set up. Add one on the Admin page.</p>
        )}
        {data && isas.length > 0 && (
          <>
            <p className="mt-3 text-3xl font-semibold">{formatGbp(data.totalValueGbp)}</p>
            <p className="text-muted-foreground mt-1 text-sm">
              {rates.length === 1 ? `${rates[0]}% · ` : ''}
              {formatGbp(data.totalAccruedThisMonthGbp)} accrued this month
            </p>
          </>
        )}
      </CardContent>
    </Card>
  )
}
