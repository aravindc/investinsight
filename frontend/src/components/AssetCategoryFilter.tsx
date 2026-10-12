import { useMemo, useRef, useState } from 'react'
import { X } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import {
  Popover,
  PopoverAnchor,
  PopoverContent,
} from '@/components/ui/popover'
import { colorForSector, iconForSector } from '@/lib/sector'
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
  const boxRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

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
      {/* An anchor, not a trigger: a trigger toggles on every click, which
          fought the input's open-on-focus and closed the list again. */}
      <PopoverAnchor asChild>
        <div
          ref={boxRef}
          className="border-input dark:bg-input/30 flex min-h-8 w-full max-w-xl flex-wrap items-center gap-1.5 rounded-lg border bg-transparent px-2.5 py-1.5 text-sm"
          // Anywhere in the box behaves like the input.
          onClick={() => {
            inputRef.current?.focus()
            setOpen(true)
          }}
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
            ref={inputRef}
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
      </PopoverAnchor>
      <PopoverContent
        className="w-(--radix-popover-trigger-width) p-0"
        align="start"
        onOpenAutoFocus={(e) => e.preventDefault()}
        // Clicks in the box (typing, removing a badge) aren't "outside".
        onInteractOutside={(e) => {
          if (boxRef.current?.contains(e.target as Node)) e.preventDefault()
        }}
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
          {filteredCategories.map((cat) => {
            const Icon = iconForSector(cat.sector)
            const color = colorForSector(cat.sector)
            return (
              <div key={cat.sector} className="mb-1">
                <button
                  type="button"
                  onClick={() => toggle({ type: 'category', sector: cat.sector })}
                  className="hover:bg-accent flex w-full items-center justify-between rounded px-2 py-1 text-left text-sm font-medium"
                >
                  <span className="flex items-center gap-2">
                    <span
                      className="flex size-6 items-center justify-center rounded-md"
                      style={{ backgroundColor: `color-mix(in srgb, ${color} 15%, transparent)`, color }}
                    >
                      <Icon className="size-3.5" />
                    </span>
                    {cat.sector}
                  </span>
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
            )
          })}
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
