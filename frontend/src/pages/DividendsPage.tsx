import { DividendGrowthChart } from '@/components/DividendGrowthChart'
import { DividendsReceivedChart } from '@/components/DividendsReceivedChart'

export function DividendsPage() {
  return (
    <div className="grid gap-6">
      <DividendsReceivedChart />
      <DividendGrowthChart />
    </div>
  )
}
