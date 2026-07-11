import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { CsvUploadCard } from '@/components/CsvUploadCard'
import { InvestedValueChart } from '@/components/InvestedValueChart'
import { SectorAllocationChart } from '@/components/SectorAllocationChart'
import { TotalProfitCard } from '@/components/TotalProfitCard'
import { TransactionsTable } from '@/components/TransactionsTable'
import { ValueCard } from '@/components/ValueCard'
import { TooltipProvider } from '@/components/ui/tooltip'

const queryClient = new QueryClient()

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <div className="mx-auto max-w-6xl space-y-6 p-6">
          <header>
            <h1 className="text-2xl font-semibold">InvestInsight</h1>
            <p className="text-muted-foreground text-sm">Portfolio dashboard</p>
          </header>
          <CsvUploadCard />
          <div className="grid grid-cols-3 gap-6">
            <ValueCard />
            <TotalProfitCard />
            <SectorAllocationChart />
          </div>
          <InvestedValueChart />
          <TransactionsTable />
        </div>
      </TooltipProvider>
    </QueryClientProvider>
  )
}

export default App
