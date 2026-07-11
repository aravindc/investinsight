import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { fetchHoldingsSummary } from '@/lib/api'
import { formatGbp } from '@/lib/format'

export function HoldingsSummaryCard() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['holdings-summary'],
    queryFn: fetchHoldingsSummary,
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>Current holdings</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading && <p className="text-muted-foreground text-sm">Loading…</p>}
        {error && (
          <p className="text-destructive text-sm">Failed to load holdings summary.</p>
        )}
        {data && (
          <div className="flex gap-8">
            <div>
              <p className="text-muted-foreground text-sm">Invested value</p>
              <p className="text-2xl font-semibold">
                {formatGbp(data.investedValueGbp)}
              </p>
            </div>
            <div>
              <p className="text-muted-foreground text-sm">Market value</p>
              <p className="text-2xl font-semibold">
                {formatGbp(data.marketValueGbp)}
              </p>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
