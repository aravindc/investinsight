import { useEffect, useRef, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { allocationKey, needsClassifying } from '@/lib/allocations'
import { measureSpec, sumMeasure, type Measure } from '@/lib/allocation-view'
import type { Allocation, AllocationGroupBy } from '@/lib/api'
import { formatGbp } from '@/lib/format'
import { inset, squarify, type Rect } from '@/lib/treemap-layout'

const HEIGHT = 440
// Parents are inset 2px (a 4px gap between them), children 1px (2px gap).
const PARENT_INSET = 2
const CHILD_INSET = 1
const HEADER = 22
const TILE_COLOR = 'var(--color-chart-1)'
const UNCLASSIFIED_COLOR = 'var(--color-muted-foreground)'

type Parent = { name: string; size: number; children: Allocation[] }

type Hovered = { group: Allocation; x: number; y: number; width: number }

function fits(text: string, width: number, fontSize: number) {
  return text.length * fontSize * 0.6 + 16 <= width
}

function percentOf(part: number, whole: number) {
  return whole > 0 ? `${((part / whole) * 100).toFixed(1)}%` : '—'
}

function useWidth<T extends HTMLElement>() {
  const ref = useRef<T>(null)
  const [width, setWidth] = useState(0)
  useEffect(() => {
    const el = ref.current
    if (!el) return
    const observer = new ResizeObserver(([entry]) => setWidth(entry.contentRect.width))
    observer.observe(el)
    return () => observer.disconnect()
  }, [])
  return [ref, width] as const
}

/**
 * Nested treemap of sub-groups inside their parent (sub-sectors within
 * sectors, sub-categories within categories). Each parent is a labelled
 * block; clicking it, or any tile in it, zooms into that parent.
 */
export function AllocationHierarchy({
  groups,
  measure,
  parentLabel,
  dimension,
  onSelectParent,
}: {
  groups: Allocation[]
  measure: Measure
  parentLabel: string
  dimension: AllocationGroupBy
  onSelectParent: (parent: string) => void
}) {
  const [containerRef, width] = useWidth<HTMLDivElement>()
  const [hovered, setHovered] = useState<Hovered | null>(null)

  // Gain can be negative, which has no area; size by market value instead.
  const sizeSpec = measureSpec(measure === 'gain' ? 'market-value' : measure)
  const sizeOf = (g: Allocation) => Number(sizeSpec.amount(g) ?? 0)
  const total = groups.reduce((sum, g) => sum + Math.max(sizeOf(g), 0), 0)

  const parents = new Map<string, Parent>()
  for (const g of groups) {
    const name = g.parent ?? g.name
    const p = parents.get(name) ?? { name, size: 0, children: [] }
    p.size += Math.max(sizeOf(g), 0)
    p.children.push(g)
    parents.set(name, p)
  }

  const parentRects = squarify(
    [...parents.values()].map((p) => ({ value: p.size, item: p })),
    { x: 0, y: 0, width, height: HEIGHT },
  )
  const headline = sumMeasure(groups, measureSpec(measure))
  const hoveredParent = hovered?.group.parent ?? null

  function renderParent(rect: Rect, parent: Parent) {
    const outer = inset(rect, PARENT_INSET)
    const label = `${parent.name} · ${percentOf(parent.size, total)}`
    const headed = outer.height >= HEADER + 24 && fits(parent.name, outer.width, 12)
    const body: Rect = headed
      ? { ...outer, y: outer.y + HEADER, height: outer.height - HEADER }
      : outer
    const children = squarify(
      parent.children.map((g) => ({ value: sizeOf(g), item: g })),
      body,
    )
    const dimmed = hoveredParent !== null && hoveredParent !== parent.name

    return (
      <g key={parent.name} opacity={dimmed ? 0.45 : 1} className="transition-opacity">
        <rect {...outer} rx={6} fill="var(--color-muted)" />
        {headed && (
          <g
            role="button"
            tabIndex={0}
            aria-label={`Open ${parent.name}`}
            className="cursor-pointer outline-none [&:focus-visible>text]:underline"
            onClick={() => onSelectParent(parent.name)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault()
                onSelectParent(parent.name)
              }
            }}
          >
            <rect x={outer.x} y={outer.y} width={outer.width} height={HEADER} fill="transparent" />
            <text x={outer.x + 8} y={outer.y + 15} fontSize={12} fontWeight={600} className="fill-foreground">
              {fits(label, outer.width, 12) ? label : parent.name}
            </text>
          </g>
        )}
        {children.map(({ item: g, ...r }) => renderChild(inset(r, CHILD_INSET), g))}
      </g>
    )
  }

  function renderChild(r: Rect, g: Allocation) {
    const showName = r.height >= 22 && fits(g.name, r.width, 11)
    const share = percentOf(sizeOf(g), total)
    const showShare = showName && r.height >= 40 && fits(share, r.width, 11)
    const grey = needsClassifying(g, dimension)
    const ink = grey ? 'var(--color-foreground)' : 'white'
    const activate = () => setHovered({ group: g, x: r.x, y: r.y, width: r.width })

    return (
      <g
        key={allocationKey(g)}
        role="button"
        tabIndex={0}
        aria-label={`${g.name}, ${g.parent}: ${formatGbp(sizeOf(g))}, ${share} of total`}
        className="cursor-pointer outline-none [&:focus-visible>rect]:stroke-foreground [&:focus-visible>rect]:[stroke-width:2]"
        onMouseEnter={activate}
        onMouseLeave={() => setHovered(null)}
        onFocus={activate}
        onBlur={() => setHovered(null)}
        onClick={() => g.parent && onSelectParent(g.parent)}
        onKeyDown={(e) => {
          if (g.parent && (e.key === 'Enter' || e.key === ' ')) {
            e.preventDefault()
            onSelectParent(g.parent)
          }
        }}
      >
        <rect
          {...r}
          rx={4}
          fill={grey ? UNCLASSIFIED_COLOR : TILE_COLOR}
          fillOpacity={grey ? 0.35 : 1}
        />
        {showName && (
          <text
            x={r.x + 6}
            y={r.y + 15}
            fill={ink}
            fontSize={11}
            fontWeight={500}
            fontStyle={g.unclassified ? 'italic' : undefined}
          >
            {g.name}
          </text>
        )}
        {showShare && (
          <text x={r.x + 6} y={r.y + 29} fill={ink} fillOpacity={0.85} fontSize={11}>
            {share}
          </text>
        )}
      </g>
    )
  }

  const hoveredParentSize = hoveredParent ? (parents.get(hoveredParent)?.size ?? 0) : 0

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-muted-foreground text-sm font-medium">
          {measureSpec(measure).label}
        </CardTitle>
        <p className="text-3xl font-semibold">
          {headline === null
            ? 'N/A'
            : `${measure === 'gain' && headline > 0 ? '+' : ''}${formatGbp(headline)}`}
        </p>
        <p className="text-muted-foreground text-xs">
          Grouped by {parentLabel.toLowerCase()}, tiles sized by {sizeSpec.label.toLowerCase()}.
          Select a {parentLabel.toLowerCase()} to zoom in.
        </p>
      </CardHeader>
      <CardContent>
        <div ref={containerRef} className="relative" style={{ height: HEIGHT }}>
          {width > 0 && (
            <svg width={width} height={HEIGHT} role="group" aria-label={`${parentLabel} treemap`}>
              {parentRects.map(({ item, ...rect }) => renderParent(rect, item))}
            </svg>
          )}
          {hovered && (
            <div
              role="tooltip"
              className="bg-popover text-popover-foreground pointer-events-none absolute z-10 min-w-40 -translate-x-1/2 -translate-y-full rounded-md border px-3 py-2 text-xs shadow-md"
              style={{ left: hovered.x + hovered.width / 2, top: Math.max(hovered.y, 0) }}
            >
              <div className="text-sm font-semibold">{formatGbp(sizeOf(hovered.group))}</div>
              <div className="text-muted-foreground">
                {hovered.group.name} · {hovered.group.parent}
              </div>
              <div className="text-muted-foreground">
                {percentOf(sizeOf(hovered.group), total)} of total ·{' '}
                {percentOf(sizeOf(hovered.group), hoveredParentSize)} of {hovered.group.parent}
              </div>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
