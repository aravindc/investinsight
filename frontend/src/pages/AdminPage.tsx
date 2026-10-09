import { useSearchParams } from 'react-router-dom'
import { CashIsaAdminCard } from '@/components/CashIsaAdminCard'
import { CategoryAdminCard } from '@/components/CategoryAdminCard'
import { CsvUploadCard } from '@/components/CsvUploadCard'
import { HouseholdAccountCard } from '@/components/HouseholdAccountCard'
import { InstrumentClassificationCard } from '@/components/InstrumentClassificationCard'
import { LogoBackfillCard } from '@/components/LogoBackfillCard'
import { MarketPriceBackfillCard } from '@/components/MarketPriceBackfillCard'
import { SectorAdminCard } from '@/components/SectorAdminCard'
import { Trading212Card } from '@/components/Trading212Card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

const TABS = [
  { value: 'accounts', label: 'Accounts' },
  { value: 'import', label: 'Import' },
  { value: 'cash-isas', label: 'Cash ISAs' },
  { value: 'market-data', label: 'Market data' },
  { value: 'categories', label: 'Categories' },
] as const

type TabValue = (typeof TABS)[number]['value']

function isTab(value: string | null): value is TabValue {
  return TABS.some((t) => t.value === value)
}

export function AdminPage() {
  // The tab lives in the URL (?tab=…) so reloads and links keep it.
  const [searchParams, setSearchParams] = useSearchParams()
  const requested = searchParams.get('tab')
  const tab: TabValue = isTab(requested) ? requested : 'accounts'

  return (
    <Tabs
      value={tab}
      onValueChange={(value) => setSearchParams({ tab: value }, { replace: true })}
    >
      <TabsList>
        {TABS.map((t) => (
          <TabsTrigger key={t.value} value={t.value}>
            {t.label}
          </TabsTrigger>
        ))}
      </TabsList>

      <TabsContent value="accounts" className="space-y-6">
        <HouseholdAccountCard />
      </TabsContent>

      <TabsContent value="import" className="space-y-6">
        <Trading212Card />
        <CsvUploadCard />
      </TabsContent>

      <TabsContent value="cash-isas" className="space-y-6">
        <CashIsaAdminCard />
      </TabsContent>

      <TabsContent value="market-data">
        <div className="grid gap-6 md:grid-cols-2">
          <MarketPriceBackfillCard />
          <LogoBackfillCard />
        </div>
      </TabsContent>

      <TabsContent value="categories" className="space-y-6">
        <div className="grid items-start gap-6 lg:grid-cols-2">
          <CategoryAdminCard />
          <SectorAdminCard />
        </div>
        <InstrumentClassificationCard />
      </TabsContent>
    </Tabs>
  )
}
