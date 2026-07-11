import { useMutation, useQueryClient } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { backfillMarketPrices } from '@/lib/api'

export function MarketPriceBackfillCard() {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: backfillMarketPrices,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['holdings-summary'] })
      queryClient.invalidateQueries({ queryKey: ['market-value-series'] })
    },
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>Market prices</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-muted-foreground text-sm">
          Fetch and store daily closing prices from Yahoo Finance for every
          held instrument, from its first buy date to today. Safe to run
          repeatedly — already up-to-date instruments are skipped.
        </p>
        <Button onClick={() => mutation.mutate()} disabled={mutation.isPending}>
          <RefreshCw className={mutation.isPending ? 'size-4 animate-spin' : 'size-4'} />
          {mutation.isPending ? 'Backfilling…' : 'Run backfill'}
        </Button>

        {mutation.isError && (
          <p className="text-destructive text-sm">
            {mutation.error instanceof Error
              ? mutation.error.message
              : 'Backfill failed.'}
          </p>
        )}

        {mutation.data && (
          <div className="text-sm">
            <p>
              <span className="font-semibold">
                {mutation.data.instrumentsProcessed}
              </span>{' '}
              instrument{mutation.data.instrumentsProcessed === 1 ? '' : 's'}{' '}
              processed.
            </p>
            {mutation.data.failed.length > 0 && (
              <p className="text-muted-foreground mt-1">
                Couldn't resolve prices for: {mutation.data.failed.join(', ')}.
              </p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
