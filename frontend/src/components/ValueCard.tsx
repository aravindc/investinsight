import { useQuery } from '@tanstack/react-query'
import { HelpCircle, Wallet } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { fetchHoldingsSummary } from '@/lib/api'
import { formatGbp } from '@/lib/format'

export function ValueCard() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['holdings-summary'],
    queryFn: fetchHoldingsSummary,
  })

  return (
    <Card>
      <CardContent>
        <div className="flex items-center gap-2">
          <span className="bg-primary/10 text-primary flex size-8 items-center justify-center rounded-md">
            <Wallet className="size-4" />
          </span>
          <span className="text-sm font-medium">Value</span>
          <Tooltip>
            <TooltipTrigger asChild>
              <HelpCircle className="text-muted-foreground size-3.5" />
            </TooltipTrigger>
            <TooltipContent>
              Current market value of instruments you hold today.
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
        {data && (
          <>
            <p className="mt-3 text-3xl font-semibold">
              {formatGbp(data.marketValueGbp)}
            </p>
            <p className="text-muted-foreground mt-1 text-sm">
              {formatGbp(data.investedValueGbp)} invested
            </p>
          </>
        )}
      </CardContent>
    </Card>
  )
}
