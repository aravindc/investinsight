import { useQuery } from '@tanstack/react-query'
import { HelpCircle, TrendingDown, TrendingUp } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  fetchHoldingsSummary,
  fetchMarketValueSeries,
} from '@/lib/api'
import { formatGbp } from '@/lib/format'

function formatSigned(value: number) {
  const formatted = formatGbp(Math.abs(value))
  return value < 0 ? `-${formatted}` : `+${formatted}`
}

function formatSignedPercent(value: number) {
  const formatted = `${Math.abs(value).toFixed(1)}%`
  return value < 0 ? `-${formatted}` : `+${formatted}`
}

export function TotalProfitCard() {
  const holdings = useQuery({
    queryKey: ['holdings-summary'],
    queryFn: fetchHoldingsSummary,
  })
  const marketSeries = useQuery({
    queryKey: ['market-value-series'],
    queryFn: fetchMarketValueSeries,
  })

  const isLoading = holdings.isLoading
  const error = holdings.error

  const invested = Number(holdings.data?.investedValueGbp ?? 0)
  const marketValue = Number(holdings.data?.marketValueGbp ?? 0)
  const profit = marketValue - invested
  const profitPercent = invested !== 0 ? (profit / invested) * 100 : 0

  // Daily change: diff of the last two days in the historical market-value
  // series. Invested capital doesn't move day-to-day absent new trades, so
  // the day's profit change equals its market-value change.
  const series = marketSeries.data ?? []
  const last = series.at(-1)
  const previous = series.at(-2)
  const dailyChange =
    last && previous ? Number(last.valueGbp) - Number(previous.valueGbp) : null
  const dailyChangePercent =
    dailyChange !== null && previous && Number(previous.valueGbp) !== 0
      ? (dailyChange / Number(previous.valueGbp)) * 100
      : null

  const isProfit = profit >= 0
  const TrendIcon = isProfit ? TrendingUp : TrendingDown

  return (
    <Card>
      <CardContent>
        <div className="flex items-center gap-2">
          <span
            className={
              isProfit
                ? 'flex size-8 items-center justify-center rounded-md bg-green-500/10 text-green-600 dark:text-green-400'
                : 'bg-destructive/10 text-destructive flex size-8 items-center justify-center rounded-md'
            }
          >
            <TrendIcon className="size-4" />
          </span>
          <span className="text-sm font-medium">Total profit</span>
          <Tooltip>
            <TooltipTrigger asChild>
              <HelpCircle className="text-muted-foreground size-3.5" />
            </TooltipTrigger>
            <TooltipContent>
              Market value minus invested (cost basis) value of current
              holdings.
            </TooltipContent>
          </Tooltip>
        </div>

        {isLoading && (
          <p className="text-muted-foreground mt-3 text-sm">Loading…</p>
        )}
        {error && (
          <p className="text-destructive mt-3 text-sm">
            Failed to load holdings summary.
          </p>
        )}
        {holdings.data && (
          <>
            <p className="mt-3 flex flex-wrap items-baseline gap-x-2 gap-y-0.5 text-3xl font-semibold">
              <span className="whitespace-nowrap">{formatSigned(profit)}</span>
              <span
                className={
                  isProfit
                    ? 'flex items-center gap-0.5 text-sm font-medium whitespace-nowrap text-green-600 dark:text-green-400'
                    : 'text-destructive flex items-center gap-0.5 text-sm font-medium whitespace-nowrap'
                }
              >
                <TrendIcon className="size-3.5" />
                {formatSignedPercent(profitPercent)}
              </span>
            </p>
            {dailyChange !== null && dailyChangePercent !== null && (
              <p className="text-muted-foreground mt-1 flex items-center gap-1 border-t pt-1 text-sm">
                <span
                  className={
                    dailyChange >= 0
                      ? 'text-green-600 dark:text-green-400'
                      : 'text-destructive'
                  }
                >
                  {formatSigned(dailyChange)}
                </span>
                <span
                  className={
                    dailyChange >= 0
                      ? 'flex items-center text-green-600 dark:text-green-400'
                      : 'text-destructive flex items-center'
                  }
                >
                  {dailyChange >= 0 ? (
                    <TrendingUp className="size-3.5" />
                  ) : (
                    <TrendingDown className="size-3.5" />
                  )}
                  {formatSignedPercent(dailyChangePercent)}
                </span>
                daily
              </p>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}
