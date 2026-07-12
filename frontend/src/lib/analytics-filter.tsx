import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { useQuery } from '@tanstack/react-query'
import { useAccountScope } from '@/lib/account-scope'
import { fetchAssetCategories, type AssetCategory } from '@/lib/api'

export type FilterEntry =
  | { type: 'category'; sector: string }
  | { type: 'asset'; instrumentId: string; ticker: string }

type AnalyticsFilterContextValue = {
  selected: FilterEntry[]
  setSelected: (entries: FilterEntry[]) => void
  categories: AssetCategory[]
  isLoading: boolean
  resolvedInstrumentIds: string[] | null // null = no filter active
  isFilterActive: boolean
}

const AnalyticsFilterContext = createContext<AnalyticsFilterContextValue | null>(null)

export function AnalyticsFilterProvider({ children }: { children: ReactNode }) {
  const { scopeParam } = useAccountScope()
  const [selected, setSelected] = useState<FilterEntry[]>([])

  const { data: categories, isLoading } = useQuery({
    queryKey: ['asset-categories', scopeParam],
    queryFn: () => fetchAssetCategories(scopeParam!),
    enabled: !!scopeParam,
  })

  // A sector/asset selection made under one account/household scope may not
  // exist (or mean something different) under another — reset on change.
  useEffect(() => {
    setSelected([])
  }, [scopeParam])

  const resolvedInstrumentIds = useMemo(() => {
    if (selected.length === 0) return null

    const ids = new Set<string>()
    for (const entry of selected) {
      if (entry.type === 'asset') {
        ids.add(entry.instrumentId)
      } else {
        const category = categories?.find((c) => c.sector === entry.sector)
        category?.instruments.forEach((i) => ids.add(i.instrumentId))
      }
    }
    return Array.from(ids)
  }, [selected, categories])

  return (
    <AnalyticsFilterContext.Provider
      value={{
        selected,
        setSelected,
        categories: categories ?? [],
        isLoading,
        resolvedInstrumentIds,
        isFilterActive: selected.length > 0,
      }}
    >
      {children}
    </AnalyticsFilterContext.Provider>
  )
}

export function useAnalyticsFilter() {
  const ctx = useContext(AnalyticsFilterContext)
  if (!ctx) throw new Error('useAnalyticsFilter must be used within an AnalyticsFilterProvider')
  return ctx
}
