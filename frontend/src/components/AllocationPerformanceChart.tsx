import {
  Bar,
  BarChart,
  CartesianGrid,
  LabelList,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { allocationKey } from '@/lib/allocations'
import type { Allocation } from '@/lib/api'
import { formatGbp } from '@/lib/format'

const ROW_HEIGHT = 32
const AXIS_HEIGHT = 32
const BAR_SIZE = 14
const RADIUS = 4
// Room right of the plot for the label on the longest bar.
const LABEL_ROOM = 52
const POSITIVE = 'var(--color-green-600)'
const NEGATIVE = 'var(--color-destructive)'

type Row = {
  key: string
  label: string
  gainPercent: number
  group: Allocation
}

function formatSignedPercent(value: number) {
  return `${value > 0 ? '+' : ''}${value.toFixed(1)}%`
}

type ShapeProps = { x?: number; y?: number; width?: number; height?: number; payload?: Row }

// A bar grows from the zero baseline: square there, 4px round at its data end.
function DivergingBar({ x = 0, y = 0, width = 0, height = 0, payload }: ShapeProps) {
  const left = Math.min(x, x + width)
  const w = Math.abs(width)
  if (w === 0 || !payload) return null
  const r = Math.min(RADIUS, w, height / 2)
  const fill = payload.gainPercent >= 0 ? POSITIVE : NEGATIVE
  const right = left + w
  const bottom = y + height
  const d =
    payload.gainPercent >= 0
      ? `M${left},${y} H${right - r} Q${right},${y} ${right},${y + r} V${bottom - r} Q${right},${bottom} ${right - r},${bottom} H${left} Z`
      : `M${right},${y} H${left + r} Q${left},${y} ${left},${y + r} V${bottom - r} Q${left},${bottom} ${left + r},${bottom} H${right} Z`
  return <path d={d} fill={fill} />
}

type LabelProps = { x?: number | string; y?: number | string; width?: number | string; height?: number | string; value?: number | string }

// Value in text ink (never the bar colour), just right of the bar's right
// edge: the tip for a gain, the zero baseline for a loss. Left of a loss
// would collide with the category labels.
function TipLabel({ x = 0, y = 0, width = 0, height = 0, value }: LabelProps) {
  const v = Number(value)
  const [nx, ny, nw, nh] = [Number(x), Number(y), Number(width), Number(height)]
  return (
    <text
      x={Math.max(nx, nx + nw) + 6}
      y={ny + nh / 2}
      dominantBaseline="central"
      textAnchor="start"
      fontSize={12}
      className="fill-foreground tabular-nums"
    >
      {formatSignedPercent(v)}
    </text>
  )
}

function PerformanceTooltip({ active, payload }: { active?: boolean; payload?: { payload: Row }[] }) {
  const row = payload?.[0]?.payload
  if (!active || !row) return null
  const gain = Number(row.group.gainGbp)
  return (
    <div className="bg-popover text-popover-foreground rounded-md border px-3 py-2 text-xs shadow-md">
      <div className="text-sm font-semibold tabular-nums">
        {formatSignedPercent(row.gainPercent)}{' '}
        <span className="font-normal">
          ({gain > 0 ? '+' : ''}
          {formatGbp(gain)})
        </span>
      </div>
      <div className="text-muted-foreground">
        {row.group.name}
        {row.group.parent && ` · ${row.group.parent}`}
      </div>
    </div>
  )
}

// Round tick values (…, -10, 0, 10, 20, …) spanning min..max, about 4 steps.
function niceTicks(min: number, max: number): number[] {
  const raw = Math.max(max - min, 1) / 4
  const magnitude = 10 ** Math.floor(Math.log10(raw))
  const step = [1, 2, 2.5, 5, 10].find((m) => m * magnitude >= raw)! * magnitude
  const ticks: number[] = []
  for (let t = Math.floor(min / step) * step; t <= Math.ceil(max / step) * step + step / 2; t += step) {
    ticks.push(Number(t.toFixed(6)))
  }
  return ticks
}

/** Return on cost per group, diverging from a zero baseline. */
export function AllocationPerformanceChart({ groups }: { groups: Allocation[] }) {
  const rows: Row[] = groups
    .filter((g) => g.gainPercent !== null)
    .map((g) => ({
      key: allocationKey(g),
      label: g.name,
      gainPercent: Number(g.gainPercent),
      group: g,
    }))
    .sort((a, b) => b.gainPercent - a.gainPercent)
  const unpriced = groups.filter((g) => g.gainPercent === null)

  const values = rows.map((r) => r.gainPercent)
  const ticks = niceTicks(Math.min(0, ...values), Math.max(0, ...values))
  const labelWidth = Math.min(180, Math.max(...rows.map((r) => r.label.length), 4) * 7 + 12)

  return (
    <Card>
      <CardHeader>
        <CardTitle>Return on cost</CardTitle>
        <p className="text-muted-foreground text-xs">Unrealised gain as a share of invested value</p>
      </CardHeader>
      <CardContent>
        {rows.length === 0 ? (
          <p className="text-muted-foreground text-sm">No fully priced groups to compare.</p>
        ) : (
          <div style={{ height: rows.length * ROW_HEIGHT + AXIS_HEIGHT }}>
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={rows} layout="vertical" margin={{ top: 0, right: LABEL_ROOM, bottom: 0, left: 0 }}>
                <CartesianGrid horizontal={false} stroke="var(--color-border)" />
                <XAxis
                  type="number"
                  domain={[ticks[0], ticks.at(-1)!]}
                  ticks={ticks}
                  tickFormatter={(v: number) => `${v}%`}
                  tick={{ fontSize: 12, fill: 'var(--color-muted-foreground)' }}
                  axisLine={false}
                  tickLine={false}
                  height={AXIS_HEIGHT}
                />
                <YAxis
                  type="category"
                  dataKey="label"
                  width={labelWidth}
                  tick={{ fontSize: 12, fill: 'var(--color-foreground)' }}
                  axisLine={false}
                  tickLine={false}
                  interval={0}
                />
                <ReferenceLine x={0} stroke="var(--color-muted-foreground)" />
                <Tooltip content={<PerformanceTooltip />} cursor={{ fill: 'var(--color-muted)', opacity: 0.5 }} />
                <Bar dataKey="gainPercent" barSize={BAR_SIZE} shape={<DivergingBar />} isAnimationActive={false}>
                  <LabelList dataKey="gainPercent" content={<TipLabel />} />
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
        {unpriced.length > 0 && (
          <p className="text-muted-foreground mt-2 text-xs">
            Not shown, as some holdings have no price: {unpriced.map((g) => g.name).join(', ')}.
          </p>
        )}
      </CardContent>
    </Card>
  )
}
