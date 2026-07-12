import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useAccountScope } from '@/lib/account-scope'
import { fetchHoldingsTable, fetchSectorAllocations, type SectorAllocation } from '@/lib/api'
import { formatGbp } from '@/lib/format'
import { colorForSector, iconForSector, MAX_SECTOR_SLICES } from '@/lib/sector'

const INNER_RADIUS = '55%'
const OUTER_RADIUS = '80%'
// Guide circle sits just outside the ring, echoing the reference design.
const GUIDE_RADIUS = '84%'

type Slice = {
  key: string
  label: string
  investedGbp: number
  color: string
}

// Collapses beyond the first (MAX_SECTOR_SLICES - 1) slices into "Other" so the
// pie never exceeds the validated categorical palette size.
function capSlices(slices: Slice[], otherColor: string): Slice[] {
  if (slices.length <= MAX_SECTOR_SLICES) return slices

  const kept = slices.slice(0, MAX_SECTOR_SLICES - 1)
  const overflow = slices.slice(MAX_SECTOR_SLICES - 1)
  const overflowTotal = overflow.reduce((sum, s) => sum + s.investedGbp, 0)

  const existingOther = kept.find((s) => s.label === 'Other')
  if (existingOther) {
    existingOther.investedGbp += overflowTotal
    return kept
  }
  return [...kept, { key: 'other', label: 'Other', investedGbp: overflowTotal, color: otherColor }]
}

export function SectorAllocationSection({
  selectedSector,
  onSelectSector,
}: {
  selectedSector: string | null
  onSelectSector: (sector: string) => void
}) {
  const { scopeParam } = useAccountScope()
  const { data, isLoading, error } = useQuery({
    queryKey: ['sector-allocations', scopeParam],
    queryFn: () => fetchSectorAllocations(scopeParam!),
    enabled: !!scopeParam,
  })

  // Shares the ['holdings-table', scopeParam] cache entry with HoldingsTable
  // — only fetched once a sector is actually selected, and reuses the same
  // network response either way (React Query dedupes by key).
  const { data: holdingsData } = useQuery({
    queryKey: ['holdings-table', scopeParam],
    queryFn: () => fetchHoldingsTable(scopeParam!),
    enabled: !!scopeParam && !!selectedSector,
  })

  const allocations = data ?? []

  const slices = useMemo(() => {
    if (selectedSector) {
      const holdingsInSector = (holdingsData ?? []).filter((h) => h.sector === selectedSector)
      const sectorColor = colorForSector(selectedSector)
      const raw = holdingsInSector
        .map((h, idx) => ({
          key: h.instrumentId,
          label: h.name,
          investedGbp: Number(h.costBasisGbp),
          // Distinct shades of the sector's color, cycling if there are
          // more holdings than shades.
          color: `color-mix(in srgb, ${sectorColor} ${100 - (idx % 5) * 15}%, var(--card))`,
        }))
        .sort((a, b) => b.investedGbp - a.investedGbp)
      return capSlices(raw, sectorColor)
    }

    const raw = allocations.map((a) => ({
      key: a.sector,
      label: a.sector,
      investedGbp: Number(a.investedGbp),
      color: colorForSector(a.sector),
    }))
    return capSlices(raw, colorForSector('Other'))
  }, [allocations, holdingsData, selectedSector])

  const total = slices.reduce((sum, s) => sum + s.investedGbp, 0)
  const grandTotalInvested = allocations.reduce(
    (sum, a) => sum + Number(a.investedGbp),
    0,
  )

  return (
    <div className="grid grid-cols-[1fr_2fr] gap-6">
      <Card>
        <CardHeader>
          <CardTitle>{selectedSector ? selectedSector : 'Portfolio'}</CardTitle>
        </CardHeader>
        <CardContent className="h-88">
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
                  data={[{ key: 'guide', investedGbp: 1 }]}
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
                  nameKey="label"
                  innerRadius={INNER_RADIUS}
                  outerRadius={OUTER_RADIUS}
                  cornerRadius={8}
                  paddingAngle={3}
                  stroke="none"
                >
                  {slices.map((s) => (
                    <Cell key={s.key} fill={s.color} />
                  ))}
                </Pie>
                <Tooltip
                  formatter={(value, _name, item) => [
                    formatGbp(String(value)),
                    `${item.payload.label} (${((Number(value) / total) * 100).toFixed(0)}%)`,
                  ]}
                />
              </PieChart>
            </ResponsiveContainer>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardContent className="thin-scrollbar h-88 overflow-y-auto p-0">
          {isLoading && (
            <p className="text-muted-foreground p-6 text-sm">Loading…</p>
          )}
          {error && (
            <p className="text-destructive p-6 text-sm">
              Failed to load sector allocations.
            </p>
          )}
          {!isLoading && !error && allocations.length === 0 && (
            <p className="text-muted-foreground p-6 text-sm">
              No current holdings.
            </p>
          )}
          {!isLoading && allocations.length > 0 && (
            <Table>
              <TableHeader className="bg-card sticky top-0 z-10">
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead className="text-right">
                    Value
                    <br />
                    Invested
                  </TableHead>
                  <TableHead className="text-right">Gain/loss</TableHead>
                  <TableHead className="text-right">Allocation</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {allocations.map((a) => (
                  <SectorRow
                    key={a.sector}
                    allocation={a}
                    selected={a.sector === selectedSector}
                    onSelect={() => onSelectSector(a.sector)}
                    shareOfPortfolio={
                      grandTotalInvested > 0
                        ? Number(a.investedGbp) / grandTotalInvested
                        : 0
                    }
                  />
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function SectorRow({
  allocation,
  selected,
  onSelect,
  shareOfPortfolio,
}: {
  allocation: SectorAllocation
  selected: boolean
  onSelect: () => void
  shareOfPortfolio: number
}) {
  const Icon = iconForSector(allocation.sector)
  const color = colorForSector(allocation.sector)
  const isPriced = Number(allocation.marketValueGbp) > 0
  const gain = Number(allocation.gainGbp)
  const isGain = gain >= 0

  return (
    <TableRow
      onClick={onSelect}
      aria-selected={selected}
      className={`cursor-pointer ${selected ? 'bg-accent' : ''}`}
    >
      <TableCell className="relative pl-4">
        <span
          className="absolute top-2 bottom-2 left-1.5 w-1 rounded-full"
          style={{ backgroundColor: color }}
        />
        <div className="flex items-center gap-3">
          <span
            className="flex size-9 items-center justify-center rounded-md"
            style={{ backgroundColor: `color-mix(in srgb, ${color} 15%, transparent)`, color }}
          >
            <Icon className="size-4" />
          </span>
          <div>
            <p className="font-medium">{allocation.sector}</p>
            <p className="text-muted-foreground text-xs">
              {allocation.instrumentCount} item
              {allocation.instrumentCount === 1 ? '' : 's'}
            </p>
          </div>
        </div>
      </TableCell>
      <TableCell className="text-right">
        {isPriced && (
          <p className="font-medium">{formatGbp(allocation.marketValueGbp)}</p>
        )}
        <p className="text-muted-foreground text-xs">
          {formatGbp(allocation.investedGbp)}
        </p>
      </TableCell>
      <TableCell className="text-right">
        {isPriced ? (
          <>
            <p
              className={
                isGain
                  ? 'font-medium text-green-600 dark:text-green-400'
                  : 'text-destructive font-medium'
              }
            >
              {isGain ? '+' : ''}
              {formatGbp(allocation.gainGbp)}
            </p>
            <p
              className={
                isGain
                  ? 'text-xs text-green-600 dark:text-green-400'
                  : 'text-destructive text-xs'
              }
            >
              {isGain ? '+' : ''}
              {allocation.gainPercent}%
            </p>
          </>
        ) : (
          <p className="text-muted-foreground text-xs">Unpriced</p>
        )}
      </TableCell>
      <TableCell className="text-right font-medium">
        {(shareOfPortfolio * 100).toFixed(0)}%
      </TableCell>
    </TableRow>
  )
}
