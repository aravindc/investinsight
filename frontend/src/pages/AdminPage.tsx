import { CsvUploadCard } from '@/components/CsvUploadCard'
import { HouseholdAccountCard } from '@/components/HouseholdAccountCard'
import { LogoBackfillCard } from '@/components/LogoBackfillCard'
import { MarketPriceBackfillCard } from '@/components/MarketPriceBackfillCard'

export function AdminPage() {
  return (
    <div className="space-y-6">
      <HouseholdAccountCard />
      <CsvUploadCard />
      <MarketPriceBackfillCard />
      <LogoBackfillCard />
    </div>
  )
}
