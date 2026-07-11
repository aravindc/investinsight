import { InvestedValueChart } from '@/components/InvestedValueChart'
import { SectorAllocationChart } from '@/components/SectorAllocationChart'
import { TotalProfitCard } from '@/components/TotalProfitCard'
import { TransactionsTable } from '@/components/TransactionsTable'
import { ValueCard } from '@/components/ValueCard'

export function HoldingsPage() {
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-3 gap-6">
        <ValueCard />
        <TotalProfitCard />
        <SectorAllocationChart />
      </div>
      <InvestedValueChart />
      <TransactionsTable />
    </div>
  )
}
