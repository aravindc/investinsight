import { useQuery } from '@tanstack/react-query'
import { HelpCircle, PiggyBank, TrendingDown, TrendingUp } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useAccountScope } from '@/lib/account-scope'
import { fetchPassiveIncome, type PassiveIncome } from '@/lib/api'
import { formatGbp } from '@/lib/format'

function formatSignedPercent(value: number) {
  const formatted = `${Math.abs(value).toFixed(1)}%`
  return value < 0 ? `-${formatted}` : `+${formatted}`
}

function PassiveIncomeCardView({
  data,
  isLoading,
  error,
}: {
  data: PassiveIncome | undefined
  isLoading: boolean
  error: unknown
}) {
  const changePercent = data?.changePercent !== null && data?.changePercent !== undefined
    ? Number(data.changePercent)
    : null
  const isIncrease = (changePercent ?? 0) >= 0
  const TrendIcon = isIncrease ? TrendingUp : TrendingDown

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
              as a percentage of current market value. The change compares
              that to the preceding 12-month period.
            </TooltipContent>
          </Tooltip>
        </div>

        {isLoading && (
          <p className="text-muted-foreground mt-3 text-sm">Loading…</p>
        )}
        {!!error && (
          <p className="text-destructive mt-3 text-sm">
            Failed to load passive income.
          </p>
        )}
        {data && (
          <>
            <p className="mt-3 flex flex-wrap items-baseline gap-x-2 gap-y-0.5 text-3xl font-semibold">
              <span className="whitespace-nowrap">{data.yieldPercent}%</span>
              {changePercent !== null && (
                <span
                  className={
                    isIncrease
                      ? 'flex items-center gap-0.5 text-sm font-medium whitespace-nowrap text-green-600 dark:text-green-400'
                      : 'text-destructive flex items-center gap-0.5 text-sm font-medium whitespace-nowrap'
                  }
                >
                  <TrendIcon className="size-3.5" />
                  {formatSignedPercent(changePercent)}
                </span>
              )}
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

export function PassiveIncomeCard() {
  const { scopeParam } = useAccountScope()
  const { data, isLoading, error } = useQuery({
    queryKey: ['passive-income', scopeParam],
    queryFn: () => fetchPassiveIncome(scopeParam!),
    enabled: !!scopeParam,
  })

  return <PassiveIncomeCardView data={data} isLoading={isLoading} error={error} />
}
