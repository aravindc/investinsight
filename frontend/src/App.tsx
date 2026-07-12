import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { NavBar } from '@/components/NavBar'
import { TooltipProvider } from '@/components/ui/tooltip'
import { AccountProvider } from '@/lib/account-scope'
import { ThemeProvider } from '@/lib/theme'
import { AdminPage } from '@/pages/AdminPage'
import { DividendsPage } from '@/pages/DividendsPage'
import { HoldingsPage } from '@/pages/HoldingsPage'
import { TransactionsPage } from '@/pages/TransactionsPage'

const queryClient = new QueryClient()

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <AccountProvider>
          <TooltipProvider>
            <BrowserRouter>
              <div className="bg-background min-h-screen">
                <NavBar />
                <main className="mx-auto max-w-6xl space-y-6 p-6">
                  <Routes>
                    <Route path="/" element={<HoldingsPage />} />
                    <Route path="/dividends" element={<DividendsPage />} />
                    <Route path="/transactions" element={<TransactionsPage />} />
                    <Route path="/admin" element={<AdminPage />} />
                  </Routes>
                </main>
              </div>
            </BrowserRouter>
          </TooltipProvider>
        </AccountProvider>
      </ThemeProvider>
    </QueryClientProvider>
  )
}

export default App
