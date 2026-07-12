import { AssetCategoryFilter } from '@/components/AssetCategoryFilter'
import { AnalyticsIrrCard } from '@/components/IrrCard'
import { AnalyticsPassiveIncomeCard } from '@/components/PassiveIncomeCard'
import { AnalyticsTotalProfitCard } from '@/components/TotalProfitCard'
import { AnalyticsValueCard } from '@/components/ValueCard'
import { AnalyticsFilterProvider } from '@/lib/analytics-filter'

export function AnalyticsPage() {
  return (
    <AnalyticsFilterProvider>
      <div className="space-y-6">
        <AssetCategoryFilter />
        <div className="grid grid-cols-4 gap-6">
          <AnalyticsValueCard />
          <AnalyticsTotalProfitCard />
          <AnalyticsIrrCard />
          <AnalyticsPassiveIncomeCard />
        </div>
      </div>
    </AnalyticsFilterProvider>
  )
}
