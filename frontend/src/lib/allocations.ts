import { useCallback } from 'react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useAccountScope } from '@/lib/account-scope'
import { useAnalyticsFilter } from '@/lib/analytics-filter'
import {
  fetchAllocationMatrix,
  fetchAllocations,
  fetchAllocationSeries,
  fetchCategories,
  fetchSectors,
  type Allocation,
  type AllocationGroupBy,
} from '@/lib/api'
import { CHART_COLORS, colorForSector } from '@/lib/sector'

export const DIMENSIONS: { value: AllocationGroupBy; label: string }[] = [
  { value: 'sector', label: 'Sector' },
  { value: 'sub-sector', label: 'Sub-sector' },
  { value: 'category', label: 'Category' },
  { value: 'sub-category', label: 'Sub-category' },
]

export function isDimension(value: string | null): value is AllocationGroupBy {
  return DIMENSIONS.some((d) => d.value === value)
}

/** Allocations for the current account scope and Analytics filter. */
export function useAllocations(groupBy: AllocationGroupBy, enabled = true) {
  const { scopeParam } = useAccountScope()
  const { resolvedInstrumentIds } = useAnalyticsFilter()
  return useQuery({
    queryKey: ['allocations', scopeParam, groupBy, resolvedInstrumentIds],
    queryFn: () => fetchAllocations(scopeParam!, groupBy, resolvedInstrumentIds ?? undefined),
    enabled: !!scopeParam && enabled,
    // Keep the current charts on screen while a new dimension or filter loads.
    placeholderData: keepPreviousData,
  })
}

// The protected catch-all sector and category new holdings land in.
const CATCH_ALL = 'Other'

/**
 * Whether a group holds instruments still to be classified: those in the
 * "Other" sector or category, or with no sub-sector.
 */
export function needsClassifying(a: Allocation, groupBy: AllocationGroupBy): boolean {
  switch (groupBy) {
    case 'sector':
    case 'category':
      return a.name === CATCH_ALL
    case 'sub-sector':
      return a.unclassified || a.parent === CATCH_ALL
    case 'sub-category':
      return a.parent === CATCH_ALL
  }
}

/** Holdings cross-tabulated by two dimensions, for the current scope and filter. */
export function useAllocationMatrix(rows: AllocationGroupBy, cols: AllocationGroupBy) {
  const { scopeParam } = useAccountScope()
  const { resolvedInstrumentIds } = useAnalyticsFilter()
  return useQuery({
    queryKey: ['allocation-matrix', scopeParam, rows, cols, resolvedInstrumentIds],
    queryFn: () =>
      fetchAllocationMatrix(scopeParam!, rows, cols, resolvedInstrumentIds ?? undefined),
    enabled: !!scopeParam,
    placeholderData: keepPreviousData,
  })
}

/** Month-end value per group, for the current scope and filter. */
export function useAllocationSeries(groupBy: AllocationGroupBy) {
  const { scopeParam } = useAccountScope()
  const { resolvedInstrumentIds } = useAnalyticsFilter()
  return useQuery({
    queryKey: ['allocation-series', scopeParam, groupBy, resolvedInstrumentIds],
    queryFn: () => fetchAllocationSeries(scopeParam!, groupBy, resolvedInstrumentIds ?? undefined),
    enabled: !!scopeParam,
    placeholderData: keepPreviousData,
  })
}

/** Stable identity for a group; sub-level names repeat across parents. */
export function allocationKey(a: Pick<Allocation, 'name' | 'parent'>): string {
  return a.parent === null ? a.name : `${a.parent}/${a.name}`
}

// The catch-all category is neutral rather than taking a palette hue.
const OTHER_CATEGORY = 'Other'
const OTHER_COLOR = 'var(--color-muted-foreground)'

// A sub-group is a shade of its parent's colour, picked by its position in
// the parent's configured list (not by rank), so it never repaints when the
// data or filter changes. Unclassified sits below every configured shade.
const SUB_GROUP_SHADES = [100, 72, 50, 34]
const UNCLASSIFIED_SHADE = 20

function shade(color: string, percent: number): string {
  return percent === 100 ? color : `color-mix(in srgb, ${color} ${percent}%, var(--color-card))`
}

/**
 * Returns a colour function for groups of the given dimension. Sectors keep
 * their Holdings-page colours; categories take palette hues in their
 * configured order.
 */
export function useGroupColor(groupBy: AllocationGroupBy) {
  const bySector = groupBy === 'sector' || groupBy === 'sub-sector'
  const { data: sectors } = useQuery({
    queryKey: ['sectors'],
    queryFn: fetchSectors,
    enabled: bySector,
  })
  const { data: categories } = useQuery({
    queryKey: ['categories'],
    queryFn: fetchCategories,
    enabled: !bySector,
  })

  const categoryColor = useCallback(
    (name: string) => {
      if (name === OTHER_CATEGORY) return OTHER_COLOR
      const configured = (categories ?? []).filter((c) => c.name !== OTHER_CATEGORY)
      const idx = configured.findIndex((c) => c.name === name)
      return CHART_COLORS[Math.max(idx, 0) % CHART_COLORS.length]
    },
    [categories],
  )

  return useCallback(
    (a: Pick<Allocation, 'name' | 'parent' | 'unclassified'>): string => {
      if (a.parent === null) {
        return bySector ? colorForSector(a.name) : categoryColor(a.name)
      }
      const parentColor = bySector ? colorForSector(a.parent) : categoryColor(a.parent)
      if (a.unclassified) return shade(parentColor, UNCLASSIFIED_SHADE)

      const siblings = bySector
        ? sectors?.find((s) => s.name === a.parent)?.subSectors
        : categories?.find((c) => c.name === a.parent)?.subCategories
      const idx = Math.max(siblings?.findIndex((s) => s.name === a.name) ?? 0, 0)
      return shade(parentColor, SUB_GROUP_SHADES[idx % SUB_GROUP_SHADES.length])
    },
    [bySector, sectors, categories, categoryColor],
  )
}
