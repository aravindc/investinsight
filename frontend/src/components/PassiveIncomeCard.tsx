import { useQuery } from '@tanstack/react-query'
import { HelpCircle, PiggyBank } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { fetchPassiveIncome } from '@/lib/api'
import { formatGbp } from '@/lib/format'

export function PassiveIncomeCard() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['passive-income'],
    queryFn: fetchPassiveIncome,
  })

  return (
    <Card>
      <CardContent>
        <div className="flex items-center gap-2">
          <span className="bg-primary/10 text-primary flex size-8 items-center justify-center rounded-md">
            <PiggyBank className="size-4" />
          </span>
          <span className="text-sm font-medium">Passive income</span>
          <Tooltip>
            <TooltipTrigger asChild>
              <HelpCircle className="text-muted-foreground size-3.5" />
            </TooltipTrigger>
            <TooltipContent>
              Dividend and interest income received in the last 12 months,
              as a percentage of current market value.
            </TooltipContent>
          </Tooltip>
        </div>

        {isLoading && (
          <p className="text-muted-foreground mt-3 text-sm">Loading…</p>
        )}
        {error && (
          <p className="text-destructive mt-3 text-sm">
            Failed to load passive income.
          </p>
        )}
        {data && (
          <>
            <p className="mt-3 text-3xl font-semibold">
              {data.yieldPercent}%
            </p>
            <p className="text-muted-foreground mt-1 text-sm">
              {formatGbp(data.trailingYearGbp)} annually
            </p>
          </>
        )}
      </CardContent>
    </Card>
  )
}
