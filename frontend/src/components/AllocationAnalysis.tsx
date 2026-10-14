import { useEffect, useRef } from 'react'
import { ChevronRight } from 'lucide-react'
import { AllocationHeatmap } from '@/components/AllocationHeatmap'
import { AllocationHierarchy } from '@/components/AllocationHierarchy'
import { AllocationPerformanceChart } from '@/components/AllocationPerformanceChart'
import { AllocationTable } from '@/components/AllocationTable'
import { AllocationOverTime } from '@/components/AllocationOverTime'
import { AllocationTreemap } from '@/components/AllocationTreemap'
import { Button } from '@/components/ui/button'
import { ClassificationCallout } from '@/components/ClassificationCallout'
import { ConcentrationCard } from '@/components/ConcentrationCard'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { DIMENSIONS, isDimension, useAllocations } from '@/lib/allocations'
import {
  ALL_LABEL,
  canDrill,
  isMeasure,
  MEASURES,
  parentDimension,
  useAllocationView,
} from '@/lib/allocation-view'

/**
 * Allocation analysis by sector, sub-sector, category or sub-category. One
 * control row scopes every chart and the table below it; clicking a sector
 * or category drills into its sub-groups.
 */
export function AllocationAnalysis() {
  const view = useAllocationView()
  const { dimension, measure, parent } = view
  const { data, isLoading, error, isPlaceholderData, refetch, isFetching } = useAllocations(dimension)
  // Drilled into one sector or category, concentration describes that
  // parent: its own figures come from the parent dimension's allocations
  // (usually cached from the tab the drill started on).
  const parentDim = parentDimension(dimension) ?? dimension
  const parentAllocations = useAllocations(parentDim, parent !== null)
  const parentFigures = parent
    ? parentAllocations.data?.groups.find((g) => g.parent === null && g.name === parent)
    : undefined

  // Move focus with a drill so keyboard and screen-reader users land on
  // the new view: the breadcrumb going in, the heading coming back out.
  const headingRef = useRef<HTMLHeadingElement>(null)
  const crumbRef = useRef<HTMLSpanElement>(null)
  const previousParent = useRef(parent)
  useEffect(() => {
    if (previousParent.current !== parent) {
      ;(parent ? crumbRef : headingRef).current?.focus()
      previousParent.current = parent
    }
  }, [parent])

  const dimensionLabel = DIMENSIONS.find((d) => d.value === dimension)!.label
  // While another dimension loads, the previous one stays on screen
  // (dimmed) and its groups can't be filtered by the new parent.
  const groups =
    data && parent && data.groupBy === dimension
      ? data.groups.filter((g) => g.parent === parent)
      : (data?.groups ?? [])
  const drill = canDrill(dimension) ? view.drillInto : undefined
  const childLabel = dimension === 'sector' ? 'View sub-sectors' : 'View sub-categories'
  // A sub-level before drilling in shows sub-groups nested in their parents.
  const parentLabel =
    dimension === 'sub-sector' ? 'Sector' : dimension === 'sub-category' ? 'Category' : null
  const nested = parentLabel !== null && parent === null && data?.groupBy === dimension
  // The heatmap's default columns are the other family: sectors by how
  // they're held (sub-category), categories by sector. Its picker switches.
  const crossBy = dimension === 'sector' || dimension === 'sub-sector' ? 'sub-category' : 'sector'

  return (
    <section className="space-y-4" aria-label="Allocation analysis">
      <div className="flex flex-wrap items-center gap-3">
        <h2 ref={headingRef} tabIndex={-1} className="mr-auto text-lg font-semibold outline-none">
          Allocation
        </h2>
        <Tabs value={dimension} onValueChange={(v) => isDimension(v) && view.setDimension(v)}>
          <TabsList>
            {DIMENSIONS.map((d) => (
              <TabsTrigger key={d.value} value={d.value}>
                {d.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
        <Select value={measure} onValueChange={(v) => isMeasure(v) && view.setMeasure(v)}>
          <SelectTrigger className="w-40" aria-label="Measure">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {MEASURES.map((m) => (
              <SelectItem key={m.value} value={m.value}>
                {m.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {parent && (
        <nav aria-label="Breadcrumb" className="flex items-center gap-1 text-sm">
          <button
            type="button"
            onClick={view.drillOut}
            className="text-muted-foreground hover:text-foreground underline-offset-4 hover:underline"
          >
            {ALL_LABEL[dimension]}
          </button>
          <ChevronRight className="text-muted-foreground size-3.5" />
          <span ref={crumbRef} tabIndex={-1} className="font-medium outline-none" aria-current="page">
            {parent}
          </span>
        </nav>
      )}

      {isLoading && (
        // First load only: placeholders the size of the charts, so the page
        // doesn't jump. Later loads keep the previous charts, dimmed.
        <div className="grid gap-6 lg:grid-cols-5" aria-busy="true">
          <span className="sr-only">Loading allocation…</span>
          <div className="bg-muted h-[440px] animate-pulse rounded-xl lg:col-span-3" />
          <div className="bg-muted h-[440px] animate-pulse rounded-xl lg:col-span-2" />
        </div>
      )}
      {!!error && !data && (
        <div className="flex items-center gap-3 text-sm" role="alert">
          <span className="text-destructive">Failed to load allocations.</span>
          <Button variant="outline" size="sm" disabled={isFetching} onClick={() => refetch()}>
            Try again
          </Button>
        </div>
      )}
      {data && groups.length === 0 && !isPlaceholderData && (
        <p className="text-muted-foreground text-sm">No holdings to break down.</p>
      )}
      {data && groups.length > 0 && (
        <div
          className={`space-y-6 transition-opacity ${isPlaceholderData ? 'opacity-50' : ''}`}
          aria-busy={isPlaceholderData}
        >
          {data.groupBy === dimension && (
            <ClassificationCallout groups={groups} dimension={dimension} />
          )}
          {nested ? (
            <div className="grid gap-6 lg:grid-cols-5">
              <div className="lg:col-span-3">
                <AllocationHierarchy
                  groups={groups}
                  measure={measure}
                  parentLabel={parentLabel}
                  dimension={dimension}
                  onSelectParent={(name) => view.drillTo(name)}
                />
              </div>
              <div className="lg:col-span-2">
                <AllocationPerformanceChart groups={groups} />
              </div>
            </div>
          ) : (
            <div className="grid gap-6 lg:grid-cols-5">
              <div className="lg:col-span-3">
                <AllocationTreemap
                  groups={groups}
                  measure={measure}
                  dimension={data.groupBy}
                  onSelect={drill}
                />
              </div>
              <div className="lg:col-span-2">
                <AllocationPerformanceChart groups={groups} />
              </div>
            </div>
          )}
          <AllocationHeatmap
            key={dimension}
            rowsBy={dimension}
            defaultColsBy={crossBy}
            parent={parent}
          />
          <div className="grid gap-6 lg:grid-cols-5">
            <div className="lg:col-span-2">
              <ConcentrationCard
                groups={groups}
                totals={parentFigures ?? data.totals}
                scopeLabel={parentFigures ? parentFigures.name : null}
                dimensionLabel={dimensionLabel}
                warnOnGroupSize={dimension === 'sector' || dimension === 'sub-sector'}
              />
            </div>
            <div className="min-w-0 lg:col-span-3">
              <AllocationOverTime
                dimension={dimension}
                dimensionLabel={dimensionLabel}
                parent={parent}
              />
            </div>
          </div>
          <AllocationTable
            key={`${dimension}:${measure}`}
            groups={groups}
            measure={measure}
            dimensionLabel={dimensionLabel}
            parentLabel={nested ? parentLabel : undefined}
            onSelect={drill}
            selectLabel={childLabel}
          />
        </div>
      )}
    </section>
  )
}
