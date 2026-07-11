import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { fetchDividendsSummary } from '@/lib/api'
import { formatGbp } from '@/lib/format'

export function DividendsSummaryCard() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['dividends-summary'],
    queryFn: fetchDividendsSummary,
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>Total dividends</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading && <p className="text-muted-foreground text-sm">Loading…</p>}
        {error && (
          <p className="text-destructive text-sm">Failed to load dividends summary.</p>
        )}
        {data && (
          <div>
            <p className="text-muted-foreground text-sm">Received to date</p>
            <p className="text-2xl font-semibold">
              {formatGbp(data.totalDividendsGbp)}
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
