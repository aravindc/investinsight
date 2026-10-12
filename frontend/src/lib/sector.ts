import {
  Boxes,
  Cpu,
  Factory,
  Fuel,
  HeartPulse,
  HelpCircle,
  House,
  Landmark,
  Layers,
  Mountain,
  Radio,
  ShoppingBag,
  ShoppingCart,
  Zap,
  type LucideIcon,
} from 'lucide-react'

// Fixed hue assignment for the built-in sectors (seeded by migration 00011),
// never reassigned based on sort order or filtering, so a given sector keeps
// the same color across renders. Sectors created from the Admin page get a
// color derived from their name instead. The list of valid sectors itself
// comes from the API (fetchSectors).
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

export const CHART_COLORS = [
  'var(--color-chart-1)',
  'var(--color-chart-2)',
  'var(--color-chart-3)',
  'var(--color-chart-4)',
  'var(--color-chart-5)',
  'var(--color-chart-6)',
  'var(--color-chart-7)',
  'var(--color-chart-8)',
]

export const MAX_SECTOR_SLICES = CHART_COLORS.length

const SECTOR_ICONS: Record<string, LucideIcon> = {
  Technology: Cpu,
  Financials: Landmark,
  Healthcare: HeartPulse,
  'Consumer Staples': ShoppingCart,
  'Consumer Discretionary': ShoppingBag,
  Energy: Fuel,
  Materials: Mountain,
  Industrials: Factory,
  Utilities: Zap,
  'Real Estate': House,
  'Communication Services': Radio,
  'Funds & ETFs': Layers,
  Other: HelpCircle,
}

export function colorForSector(sector: string): string {
  const idx = SECTOR_ORDER.indexOf(sector as (typeof SECTOR_ORDER)[number])
  if (idx >= 0) return CHART_COLORS[idx % CHART_COLORS.length]
  // Custom sector: a stable color from a hash of its name.
  let hash = 0
  for (const ch of sector) hash = (hash * 31 + ch.charCodeAt(0)) >>> 0
  return CHART_COLORS[hash % CHART_COLORS.length]
}

export function iconForSector(sector: string): LucideIcon {
  return SECTOR_ICONS[sector] ?? Boxes
}
