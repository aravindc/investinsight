import { CsvUploadCard } from '@/components/CsvUploadCard'
import { HouseholdAccountCard } from '@/components/HouseholdAccountCard'
import { MarketPriceBackfillCard } from '@/components/MarketPriceBackfillCard'

export function AdminPage() {
  return (
    <div className="space-y-6">
      <HouseholdAccountCard />
      <CsvUploadCard />
      <MarketPriceBackfillCard />
    </div>
  )
}
