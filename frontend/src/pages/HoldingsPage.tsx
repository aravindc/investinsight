import { useState } from 'react'
import { CashCard } from '@/components/CashCard'
import { HoldingsTable } from '@/components/HoldingsTable'
import { InvestedValueChart } from '@/components/InvestedValueChart'
import { IrrCard } from '@/components/IrrCard'
import { PassiveIncomeCard } from '@/components/PassiveIncomeCard'
import { SectorAllocationSection } from '@/components/SectorAllocationSection'
import { TotalProfitCard } from '@/components/TotalProfitCard'
import { TotalValueCard } from '@/components/TotalValueCard'
import { ValueCard } from '@/components/ValueCard'

export function HoldingsPage() {
  const [selectedSector, setSelectedSector] = useState<string | null>(null)

  return (
    <div className="space-y-6">
      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
        <TotalValueCard />
        <ValueCard />
        <CashCard />
        <TotalProfitCard />
        <IrrCard />
        <PassiveIncomeCard />
      </div>
      <SectorAllocationSection
        selectedSector={selectedSector}
        onSelectSector={(sector) =>
          setSelectedSector((current) => (current === sector ? null : sector))
        }
      />
      <HoldingsTable
        selectedSector={selectedSector}
        onClearSector={() => setSelectedSector(null)}
      />
      <InvestedValueChart />
    </div>
  )
}
