import { useMemo, useState } from 'react'

export type SortKey = 'name' | 'count'
export type SortState = { key: SortKey; dir: 'asc' | 'desc' } | null

/**
 * Sorts rows by name or instrument count. Starts unsorted (the server's
 * display order); the first click on a column sorts names A→Z and counts
 * largest-first, further clicks flip the direction.
 */
export function useNameCountSort<T extends { name: string; instrumentCount: number }>(rows: T[]) {
  const [sort, setSort] = useState<SortState>(null)

  const sorted = useMemo(() => {
    if (!sort) return rows
    const factor = sort.dir === 'asc' ? 1 : -1
    return [...rows].sort((a, b) => {
      const byKey =
        sort.key === 'name'
          ? a.name.localeCompare(b.name)
          : a.instrumentCount - b.instrumentCount
      return byKey * factor || a.name.localeCompare(b.name)
    })
  }, [rows, sort])

  const toggle = (key: SortKey) =>
    setSort((current) =>
      current?.key === key
        ? { key, dir: current.dir === 'asc' ? 'desc' : 'asc' }
        : { key, dir: key === 'name' ? 'asc' : 'desc' },
    )

  return { sorted, sort, toggle }
}
