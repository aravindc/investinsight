import { useState } from 'react'
import { ResponsiveContainer, Treemap, type TreemapNode } from 'recharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { allocationKey } from '@/lib/allocations'
import { measureSpec, sumMeasure, type Measure } from '@/lib/allocation-view'
import type { Allocation } from '@/lib/api'
import { formatGbp } from '@/lib/format'

const HEIGHT = 340
// 2px surface gap between tiles: each tile is inset 1px on every side.
const GAP = 1
const TILE_COLOR = 'var(--color-chart-1)'

type Node = {
  name: string
  size: number
  id: string
  group: Allocation
}

// Rough width of 12px UI text; labels only render where they clearly fit,
// otherwise the tooltip and table carry the name.
function fits(text: string, width: number, fontSize: number) {
  return text.length * fontSize * 0.6 + 16 <= width
}

function sharePercent(size: number, total: number) {
  return total > 0 ? `${((size / total) * 100).toFixed(1)}%` : ''
}

/**
 * Part-to-whole view of the selected measure, one tile per group. A single
 * hue: there are more sectors than palette colours, so labels carry
 * identity and hue is never reused for two groups.
 */
export function AllocationTreemap({
  groups,
  measure,
  onSelect,
}: {
  groups: Allocation[]
  measure: Measure
  onSelect?: (group: Allocation) => void
}) {
  // The hovered or focused tile, with its position for the tooltip.
  const [active, setActive] = useState<{ key: string; x: number; y: number; width: number } | null>(
    null,
  )

  // Gain can be negative, which has no area; size by market value instead.
  const sizeSpec = measureSpec(measure === 'gain' ? 'market-value' : measure)
  const nodes: Node[] = groups
    .map((group) => ({
      name: group.name,
      size: Number(sizeSpec.amount(group) ?? 0),
      id: allocationKey(group),
      group,
    }))
    .filter((n) => n.size > 0)
  const omitted = groups.length - nodes.length
  const total = sumMeasure(groups, sizeSpec) ?? 0

  const activeNode = active && nodes.find((n) => n.id === active.key)

  function renderTile(props: TreemapNode) {
    const { x, y, width, height, depth } = props
    const group = props.group as Allocation | undefined
    if (depth === 0 || !group) return <g />
    const key = allocationKey(group)
    const w = Math.max(width - GAP * 2, 0)
    const h = Math.max(height - GAP * 2, 0)
    const size = Number(sizeSpec.amount(group) ?? 0)
    const share = sharePercent(size, total)
    const showName = h >= 22 && fits(group.name, w, 12)
    const showShare = showName && h >= 44 && fits(share, w, 11)
    const isActive = active?.key === key
    const activate = () => setActive({ key, x, y, width })

    return (
      <g
        role={onSelect ? 'button' : 'img'}
        tabIndex={0}
        aria-label={`${group.name}: ${formatGbp(size)}, ${share}`}
        className={onSelect ? 'cursor-pointer outline-none' : 'outline-none'}
        onMouseEnter={activate}
        onMouseLeave={() => setActive(null)}
        onFocus={activate}
        onBlur={() => setActive(null)}
        onClick={() => onSelect?.(group)}
        onKeyDown={(e) => {
          if (onSelect && (e.key === 'Enter' || e.key === ' ')) {
            e.preventDefault()
            onSelect(group)
          }
        }}
      >
        <rect
          x={x + GAP}
          y={y + GAP}
          width={w}
          height={h}
          rx={4}
          fill={TILE_COLOR}
          fillOpacity={active === null || isActive ? 1 : 0.55}
          className="transition-[fill-opacity]"
        />
        {showName && (
          <text x={x + GAP + 8} y={y + GAP + 18} fill="white" fontSize={12} fontWeight={500}>
            {group.name}
          </text>
        )}
        {showShare && (
          <text x={x + GAP + 8} y={y + GAP + 34} fill="white" fillOpacity={0.85} fontSize={11}>
            {share}
          </text>
        )}
      </g>
    )
  }

  const headline = sumMeasure(groups, measureSpec(measure))

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-muted-foreground text-sm font-medium">
          {measureSpec(measure).label}
        </CardTitle>
        <p className="text-3xl font-semibold">{headline === null ? 'N/A' : formatGbp(headline)}</p>
        <p className="text-muted-foreground text-xs">
          Tiles sized by {sizeSpec.label.toLowerCase()}
          {omitted > 0 && ` · ${omitted} with nothing to show left out`}
        </p>
      </CardHeader>
      <CardContent>
        {nodes.length === 0 ? (
          <p className="text-muted-foreground text-sm">Nothing to show for this measure.</p>
        ) : (
          <div className="relative" style={{ height: HEIGHT }}>
            <ResponsiveContainer width="100%" height="100%">
              <Treemap
                data={nodes}
                dataKey="size"
                nameKey="name"
                aspectRatio={4 / 3}
                isAnimationActive={false}
                content={renderTile}
              />
            </ResponsiveContainer>
            {active && activeNode && (
              <div
                role="tooltip"
                className="bg-popover text-popover-foreground pointer-events-none absolute z-10 min-w-36 -translate-x-1/2 -translate-y-full rounded-md border px-3 py-2 text-xs shadow-md"
                style={{ left: active.x + active.width / 2, top: Math.max(active.y, 0) }}
              >
                <div className="text-sm font-semibold">{formatGbp(activeNode.size)}</div>
                <div className="text-muted-foreground">
                  {activeNode.group.name}
                  {activeNode.group.parent && ` · ${activeNode.group.parent}`}
                </div>
                <div className="text-muted-foreground">
                  {sharePercent(activeNode.size, total)} · {activeNode.group.instrumentCount}{' '}
                  {activeNode.group.instrumentCount === 1 ? 'holding' : 'holdings'}
                </div>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
