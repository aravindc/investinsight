import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { CsvUploadCard } from '@/components/CsvUploadCard'
import { DividendsSummaryCard } from '@/components/DividendsSummaryCard'
import { HoldingsSummaryCard } from '@/components/HoldingsSummaryCard'
import { TransactionsTable } from '@/components/TransactionsTable'

const queryClient = new QueryClient()

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <div className="mx-auto max-w-6xl space-y-6 p-6">
        <header>
          <h1 className="text-2xl font-semibold">InvestInsight</h1>
          <p className="text-muted-foreground text-sm">Portfolio dashboard</p>
        </header>
        <CsvUploadCard />
        <div className="grid grid-cols-2 gap-6">
          <HoldingsSummaryCard />
          <DividendsSummaryCard />
        </div>
        <TransactionsTable />
      </div>
    </QueryClientProvider>
  )
}

export default App
