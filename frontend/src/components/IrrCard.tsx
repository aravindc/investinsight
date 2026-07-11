import { useQuery } from '@tanstack/react-query'
import { HelpCircle, Percent } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { fetchPortfolioIrr } from '@/lib/api'

export function IrrCard() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['portfolio-irr'],
    queryFn: fetchPortfolioIrr,
  })

  const isPositive = (data?.irr ?? 0) >= 0

  return (
    <Card>
      <CardContent>
        <div className="flex items-center gap-2">
          <span className="bg-primary/10 text-primary flex size-8 items-center justify-center rounded-md">
            <Percent className="size-4" />
          </span>
          <span className="text-sm font-medium">IRR</span>
          <Tooltip>
            <TooltipTrigger asChild>
              <HelpCircle className="text-muted-foreground size-3.5" />
            </TooltipTrigger>
            <TooltipContent>
              Money-weighted annualized rate of return (XIRR) on invested
              holdings: every buy, sell, dividend, and interest payment as a
              dated cash flow, plus current market value as if liquidated
              today. Deposits and withdrawals aren't counted.
            </TooltipContent>
          </Tooltip>
        </div>

        {isLoading && (
          <p className="text-muted-foreground mt-3 text-sm">Loading…</p>
        )}
        {error && (
          <p className="text-destructive mt-3 text-sm">Failed to load IRR.</p>
        )}
        {data && data.irr === null && (
          <p className="text-muted-foreground mt-3 text-sm">
            Not enough activity yet to compute a rate of return.
          </p>
        )}
        {data && data.irr !== null && (
          <p
            className={
              isPositive
                ? 'mt-3 text-3xl font-semibold text-green-600 dark:text-green-400'
                : 'text-destructive mt-3 text-3xl font-semibold'
            }
          >
            {isPositive ? '+' : ''}
            {(data.irr * 100).toFixed(1)}%
          </p>
        )}
      </CardContent>
    </Card>
  )
}
