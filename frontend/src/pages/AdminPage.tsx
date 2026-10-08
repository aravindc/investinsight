import { CsvUploadCard } from '@/components/CsvUploadCard'
import { HouseholdAccountCard } from '@/components/HouseholdAccountCard'
import { InstrumentSectorCard } from '@/components/InstrumentSectorCard'
import { LogoBackfillCard } from '@/components/LogoBackfillCard'
import { MarketPriceBackfillCard } from '@/components/MarketPriceBackfillCard'

export function AdminPage() {
  return (
    <div className="space-y-6">
      <HouseholdAccountCard />
      <CsvUploadCard />
          <div className="grid grid-cols-2 space-x-6">
            <MarketPriceBackfillCard />
            <LogoBackfillCard />
          </div>
      <InstrumentSectorCard />
    </div>

  )
}
