import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { useAnalyticsFilter, type FilterEntry } from '@/lib/analytics-filter'

export function AssetCategoryFilter() {
  const { selected, setSelected, categories, isLoading } = useAnalyticsFilter()

  const isSectorChecked = (sector: string) =>
    selected.some((e) => e.type === 'category' && e.sector === sector)
  const isAssetChecked = (instrumentId: string) =>
    selected.some((e) => e.type === 'asset' && e.instrumentId === instrumentId)

  function toggle(entry: FilterEntry, checked: boolean) {
    const matches = (e: FilterEntry) =>
      entry.type === 'category'
        ? e.type === 'category' && e.sector === entry.sector
        : e.type === 'asset' && e.instrumentId === entry.instrumentId

    setSelected(checked ? [...selected, entry] : selected.filter((e) => !matches(e)))
  }

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline">
          Choose a category or an asset
          {selected.length > 0 ? ` (${selected.length})` : ''}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80 p-0" align="start">
        <div className="thin-scrollbar max-h-96 overflow-y-auto p-2">
          {isLoading && (
            <p className="text-muted-foreground p-2 text-sm">Loading…</p>
          )}
          {!isLoading && categories.length === 0 && (
            <p className="text-muted-foreground p-2 text-sm">No current holdings.</p>
          )}
          {categories.map((cat) => (
            <div key={cat.sector} className="mb-2">
              <label className="hover:bg-accent flex items-center gap-2 rounded px-2 py-1 text-sm font-medium">
                <Checkbox
                  checked={isSectorChecked(cat.sector)}
                  onCheckedChange={(checked) =>
                    toggle({ type: 'category', sector: cat.sector }, !!checked)
                  }
                />
                {cat.sector}
              </label>
              <div className="ml-6">
                {cat.instruments.map((inst) => (
                  <label
                    key={inst.instrumentId}
                    className="hover:bg-accent flex items-center gap-2 rounded px-2 py-1 text-sm"
                  >
                    <Checkbox
                      checked={isAssetChecked(inst.instrumentId)}
                      onCheckedChange={(checked) =>
                        toggle(
                          { type: 'asset', instrumentId: inst.instrumentId, ticker: inst.ticker },
                          !!checked,
                        )
                      }
                    />
                    {inst.name} ({inst.ticker})
                  </label>
                ))}
              </div>
            </div>
          ))}
        </div>
        {selected.length > 0 && (
          <div className="border-t p-2">
            <Button variant="ghost" size="sm" onClick={() => setSelected([])}>
              Clear filters
            </Button>
          </div>
        )}
      </PopoverContent>
    </Popover>
  )
}
