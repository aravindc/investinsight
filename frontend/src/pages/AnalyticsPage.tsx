import { AllocationAnalysis } from '@/components/AllocationAnalysis'
import { AssetCategoryFilter } from '@/components/AssetCategoryFilter'
import { AnalyticsFilterProvider } from '@/lib/analytics-filter'

export function AnalyticsPage() {
  return (
    <AnalyticsFilterProvider>
      <div className="space-y-6">
        <AssetCategoryFilter />
        <AllocationAnalysis />
      </div>
    </AnalyticsFilterProvider>
  )
}
