import { useMutation, useQueryClient } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { backfillLogos } from '@/lib/api'

export function LogoBackfillCard() {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: backfillLogos,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['holdings-table'] })
    },
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>Instrument logos</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-muted-foreground text-sm">
          Fetch and store logo images from logo.dev for every held
          instrument that doesn't have one yet. Safe to run repeatedly —
          only newly added holdings are fetched.
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
                Couldn't find a logo for: {mutation.data.failed.join(', ')}.
              </p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
