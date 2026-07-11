import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { CsvUploadCard } from '@/components/CsvUploadCard'
import { DividendGrowthChart } from '@/components/DividendGrowthChart'
import { DividendsReceivedChart } from '@/components/DividendsReceivedChart'
import { InvestedValueChart } from '@/components/InvestedValueChart'
import { SectorAllocationChart } from '@/components/SectorAllocationChart'
import { ThemeToggle } from '@/components/ThemeToggle'
import { TotalProfitCard } from '@/components/TotalProfitCard'
import { TransactionsTable } from '@/components/TransactionsTable'
import { ValueCard } from '@/components/ValueCard'
import { TooltipProvider } from '@/components/ui/tooltip'
import { ThemeProvider } from '@/lib/theme'

const queryClient = new QueryClient()

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <TooltipProvider>
          <div className="mx-auto max-w-6xl space-y-6 p-6">
            <header className="flex items-start justify-between">
              <div>
                <h1 className="text-2xl font-semibold">InvestInsight</h1>
                <p className="text-muted-foreground text-sm">Portfolio dashboard</p>
              </div>
              <ThemeToggle />
            </header>
            <CsvUploadCard />
            <div className="grid grid-cols-3 gap-6">
              <ValueCard />
              <TotalProfitCard />
              <SectorAllocationChart />
            </div>
            <div className="grid grid-cols-2 gap-6">
              <DividendsReceivedChart />
              <DividendGrowthChart />
            </div>
            <InvestedValueChart />
            <TransactionsTable />
          </div>
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>
  )
}

export default App
