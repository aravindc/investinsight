import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { fetchSectorAllocations } from '@/lib/api'
import { formatGbp } from '@/lib/format'

const INNER_RADIUS = '55%'
const OUTER_RADIUS = '80%'
// Guide circle sits just outside the ring, echoing the reference design.
const GUIDE_RADIUS = '84%'

// Fixed hue assignment by sector identity, matching the backend's declared
// sector order — never reassigned based on sort order or filtering, so a
// given sector keeps the same color across renders.
const SECTOR_ORDER = [
  'Technology',
  'Financials',
  'Healthcare',
  'Consumer Staples',
  'Consumer Discretionary',
  'Energy',
  'Materials',
  'Industrials',
  'Utilities',
  'Real Estate',
  'Communication Services',
  'Funds & ETFs',
  'Other',
] as const

const CHART_COLORS = [
  'var(--color-chart-1)',
  'var(--color-chart-2)',
  'var(--color-chart-3)',
  'var(--color-chart-4)',
  'var(--color-chart-5)',
  'var(--color-chart-6)',
  'var(--color-chart-7)',
  'var(--color-chart-8)',
]

const MAX_SLICES = CHART_COLORS.length

type Slice = {
  sector: string
  investedGbp: number
}

function colorForSector(sector: string): string {
  const idx = SECTOR_ORDER.indexOf(sector as (typeof SECTOR_ORDER)[number])
  return CHART_COLORS[idx % CHART_COLORS.length] ?? CHART_COLORS[CHART_COLORS.length - 1]
}

// Collapses beyond the first (MAX_SLICES - 1) sectors into "Other" so the
// pie never exceeds the validated categorical palette size.
function capSlices(allocations: Slice[]): Slice[] {
  if (allocations.length <= MAX_SLICES) return allocations

  const kept = allocations.slice(0, MAX_SLICES - 1)
  const overflow = allocations.slice(MAX_SLICES - 1)
  const overflowTotal = overflow.reduce((sum, a) => sum + a.investedGbp, 0)

  const existingOther = kept.find((a) => a.sector === 'Other')
  if (existingOther) {
    existingOther.investedGbp += overflowTotal
    return kept
  }
  return [...kept, { sector: 'Other', investedGbp: overflowTotal }]
}

export function SectorAllocationChart() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['sector-allocations'],
    queryFn: fetchSectorAllocations,
  })

  const slices = useMemo(() => {
    const allocations = (data ?? []).map((a) => ({
      sector: a.sector,
      investedGbp: Number(a.investedGbp),
    }))
    return capSlices(allocations)
  }, [data])

  const total = slices.reduce((sum, s) => sum + s.investedGbp, 0)

  return (
    <Card>
      <CardHeader>
        <CardTitle>Holdings by sector</CardTitle>
      </CardHeader>
      <CardContent className="h-40">
        {isLoading && <p className="text-muted-foreground text-sm">Loading…</p>}
        {error && (
          <p className="text-destructive text-sm">Failed to load sector allocations.</p>
        )}
        {!isLoading && !error && slices.length === 0 && (
          <p className="text-muted-foreground text-sm">No current holdings.</p>
        )}
        {!isLoading && slices.length > 0 && (
          <ResponsiveContainer width="100%" height="100%">
            <PieChart>
              <Pie
                data={[{ sector: 'guide', investedGbp: 1 }]}
                dataKey="investedGbp"
                innerRadius={GUIDE_RADIUS}
                outerRadius={GUIDE_RADIUS}
                stroke="var(--muted-foreground)"
                strokeDasharray="4 4"
                strokeOpacity={0.4}
                fill="none"
                isAnimationActive={false}
                legendType="none"
                tooltipType="none"
              />
              <Pie
                data={slices}
                dataKey="investedGbp"
                nameKey="sector"
                innerRadius={INNER_RADIUS}
                outerRadius={OUTER_RADIUS}
                cornerRadius={8}
                paddingAngle={3}
                stroke="none"
              >
                {slices.map((s) => (
                  <Cell key={s.sector} fill={colorForSector(s.sector)} />
                ))}
              </Pie>
              <Tooltip
                formatter={(value, _name, item) => [
                  formatGbp(String(value)),
                  `${item.payload.sector} (${((Number(value) / total) * 100).toFixed(0)}%)`,
                ]}
              />
            </PieChart>
          </ResponsiveContainer>
        )}
      </CardContent>
    </Card>
  )
}
