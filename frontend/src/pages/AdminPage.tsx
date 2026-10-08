import { CashIsaAdminCard } from '@/components/CashIsaAdminCard'
import { CsvUploadCard } from '@/components/CsvUploadCard'
import { HouseholdAccountCard } from '@/components/HouseholdAccountCard'
import { InstrumentSectorCard } from '@/components/InstrumentSectorCard'
import { LogoBackfillCard } from '@/components/LogoBackfillCard'
import { MarketPriceBackfillCard } from '@/components/MarketPriceBackfillCard'
import { Trading212Card } from '@/components/Trading212Card'

export function AdminPage() {
  return (
    <div className="space-y-6">
      <HouseholdAccountCard />
      <Trading212Card />
      <CashIsaAdminCard />
      <CsvUploadCard />
      <div className="grid gap-6 md:grid-cols-2">
        <MarketPriceBackfillCard />
        <LogoBackfillCard />
      </div>
      <InstrumentSectorCard />
    </div>
  )
}
