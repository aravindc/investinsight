import { useSearchParams } from 'react-router-dom'
import { isDimension } from '@/lib/allocations'
import type { Allocation, AllocationFigures, AllocationGroupBy } from '@/lib/api'

export type Measure = 'market-value' | 'invested' | 'gain' | 'dividends'

export type MeasureSpec = {
  value: Measure
  label: string
  amount: (f: AllocationFigures) => string | null
}

export const MEASURES: MeasureSpec[] = [
  { value: 'market-value', label: 'Market value', amount: (f) => f.marketValueGbp },
  { value: 'invested', label: 'Invested', amount: (f) => f.investedGbp },
  { value: 'gain', label: 'Gain', amount: (f) => f.gainGbp },
  { value: 'dividends', label: 'Dividends (12m)', amount: (f) => f.trailingYearDividendsGbp },
]

export function isMeasure(value: string | null): value is Measure {
  return MEASURES.some((m) => m.value === value)
}

export function measureSpec(measure: Measure): MeasureSpec {
  return MEASURES.find((m) => m.value === measure)!
}

// Drilling into a sector or category shows its sub-groups.
const CHILD: Partial<Record<AllocationGroupBy, AllocationGroupBy>> = {
  sector: 'sub-sector',
  category: 'sub-category',
}
const PARENT: Partial<Record<AllocationGroupBy, AllocationGroupBy>> = {
  'sub-sector': 'sector',
  'sub-category': 'category',
}

export function canDrill(dimension: AllocationGroupBy): boolean {
  return dimension in CHILD
}

export const ALL_LABEL: Partial<Record<AllocationGroupBy, string>> = {
  'sub-sector': 'All sectors',
  'sub-category': 'All categories',
}

/**
 * The Analytics allocation view's state, kept in the URL so a link or a
 * refresh reopens the same view: `dimension`, `measure`, and `parent` (the
 * sector or category drilled into, only with a sub-level dimension).
 */
export function useAllocationView() {
  const [params, setParams] = useSearchParams()
  const rawDimension = params.get('dimension')
  const rawMeasure = params.get('measure')
  const dimension: AllocationGroupBy = isDimension(rawDimension) ? rawDimension : 'sector'
  const measure: Measure = isMeasure(rawMeasure) ? rawMeasure : 'market-value'
  const parent = dimension in PARENT ? params.get('parent') : null

  const update = (changes: Record<string, string | null>) =>
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        for (const [key, value] of Object.entries(changes)) {
          if (value === null) next.delete(key)
          else next.set(key, value)
        }
        return next
      },
      { replace: true },
    )

  return {
    dimension,
    measure,
    parent,
    setDimension: (d: AllocationGroupBy) => update({ dimension: d, parent: null }),
    setMeasure: (m: Measure) => update({ measure: m }),
    drillInto: (group: Allocation) => {
      const child = CHILD[dimension]
      if (child) update({ dimension: child, parent: group.name })
    },
    // Zoom a sub-level view into one parent.
    drillTo: (parentName: string) => update({ parent: parentName }),
    drillOut: () => update({ dimension: PARENT[dimension] ?? dimension, parent: null }),
  }
}

/** Sums a measure across groups; null if any group lacks a value. */
export function sumMeasure(groups: AllocationFigures[], spec: MeasureSpec): number | null {
  let total = 0
  for (const g of groups) {
    const amount = spec.amount(g)
    if (amount === null) return null
    total += Number(amount)
  }
  return total
}
