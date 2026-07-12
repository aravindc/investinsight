import { useMemo, useState } from 'react'
import { X } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { useAnalyticsFilter, type FilterEntry } from '@/lib/analytics-filter'

function entryKey(entry: FilterEntry): string {
  return entry.type === 'category' ? `category:${entry.sector}` : `asset:${entry.instrumentId}`
}

function entryLabel(entry: FilterEntry): string {
  return entry.type === 'category' ? entry.sector : entry.ticker
}

export function AssetCategoryFilter() {
  const { selected, setSelected, categories, isLoading } = useAnalyticsFilter()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')

  const isSelected = (entry: FilterEntry) => selected.some((e) => entryKey(e) === entryKey(entry))

  function toggle(entry: FilterEntry) {
    setSelected(
      isSelected(entry)
        ? selected.filter((e) => entryKey(e) !== entryKey(entry))
        : [...selected, entry],
    )
  }

  function remove(entry: FilterEntry) {
    setSelected(selected.filter((e) => entryKey(e) !== entryKey(entry)))
  }

  const filteredCategories = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return categories

    return categories
      .map((cat) => ({
        ...cat,
        matchesSector: cat.sector.toLowerCase().includes(q),
        instruments: cat.instruments.filter(
          (inst) =>
            inst.name.toLowerCase().includes(q) || inst.ticker.toLowerCase().includes(q),
        ),
      }))
      .filter((cat) => cat.matchesSector || cat.instruments.length > 0)
  }, [categories, query])

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <div
          className="border-input dark:bg-input/30 flex min-h-8 w-full max-w-xl flex-wrap items-center gap-1.5 rounded-lg border bg-transparent px-2.5 py-1.5 text-sm"
          onClick={() => setOpen(true)}
        >
          {selected.map((entry) => (
            <Badge key={entryKey(entry)} variant="secondary" className="gap-1">
              {entryLabel(entry)}
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation()
                  remove(entry)
                }}
                className="hover:text-foreground"
              >
                <X className="size-3" />
              </button>
            </Badge>
          ))}
          <input
            value={query}
            onChange={(e) => {
              setQuery(e.target.value)
              setOpen(true)
            }}
            onFocus={() => setOpen(true)}
            placeholder={selected.length === 0 ? 'Choose a category or an asset…' : ''}
            className="text-muted-foreground min-w-32 flex-1 bg-transparent outline-none placeholder:text-muted-foreground"
          />
        </div>
      </PopoverTrigger>
      <PopoverContent
        className="w-(--radix-popover-trigger-width) p-0"
        align="start"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <div className="thin-scrollbar max-h-96 overflow-y-auto p-2">
          {isLoading && (
            <p className="text-muted-foreground p-2 text-sm">Loading…</p>
          )}
          {!isLoading && categories.length === 0 && (
            <p className="text-muted-foreground p-2 text-sm">No current holdings.</p>
          )}
          {!isLoading && categories.length > 0 && filteredCategories.length === 0 && (
            <p className="text-muted-foreground p-2 text-sm">No matches.</p>
          )}
          {filteredCategories.map((cat) => (
            <div key={cat.sector} className="mb-1">
              <button
                type="button"
                onClick={() => toggle({ type: 'category', sector: cat.sector })}
                className="hover:bg-accent flex w-full items-center justify-between rounded px-2 py-1 text-left text-sm font-medium"
              >
                {cat.sector}
                {isSelected({ type: 'category', sector: cat.sector }) && (
                  <span className="text-primary text-xs">Selected</span>
                )}
              </button>
              <div className="ml-2">
                {cat.instruments.map((inst) => (
                  <button
                    type="button"
                    key={inst.instrumentId}
                    onClick={() =>
                      toggle({ type: 'asset', instrumentId: inst.instrumentId, ticker: inst.ticker })
                    }
                    className="hover:bg-accent flex w-full items-center justify-between rounded px-2 py-1 text-left text-sm"
                  >
                    <span>
                      {inst.name} ({inst.ticker})
                    </span>
                    {isSelected({ type: 'asset', instrumentId: inst.instrumentId, ticker: inst.ticker }) && (
                      <span className="text-primary text-xs">Selected</span>
                    )}
                  </button>
                ))}
              </div>
            </div>
          ))}
        </div>
        {selected.length > 0 && (
          <div className="border-t p-2">
            <button
              type="button"
              onClick={() => setSelected([])}
              className="text-muted-foreground hover:text-foreground text-sm"
            >
              Clear filters
            </button>
          </div>
        )}
      </PopoverContent>
    </Popover>
  )
}
