import { CsvUploadCard } from '@/components/CsvUploadCard'
import { MarketPriceBackfillCard } from '@/components/MarketPriceBackfillCard'

export function AdminPage() {
  return (
    <div className="space-y-6">
      <CsvUploadCard />
      <MarketPriceBackfillCard />
    </div>
  )
}
