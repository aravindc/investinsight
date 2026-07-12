import { HoldingsTable } from '@/components/HoldingsTable'
import { InvestedValueChart } from '@/components/InvestedValueChart'
import { IrrCard } from '@/components/IrrCard'
import { PassiveIncomeCard } from '@/components/PassiveIncomeCard'
import { SectorAllocationSection } from '@/components/SectorAllocationSection'
import { TotalProfitCard } from '@/components/TotalProfitCard'
import { ValueCard } from '@/components/ValueCard'

export function HoldingsPage() {
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-4 gap-6">
        <ValueCard />
        <TotalProfitCard />
        <IrrCard />
        <PassiveIncomeCard />
      </div>
      <SectorAllocationSection />
      <InvestedValueChart />
      <HoldingsTable />
    </div>
  )
}
