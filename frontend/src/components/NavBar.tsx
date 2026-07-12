import { Coins, LayoutDashboard, Receipt, Settings } from 'lucide-react'
import { NavLink } from 'react-router-dom'
import { AccountScopeSwitcher } from '@/components/AccountScopeSwitcher'
import { ThemeToggle } from '@/components/ThemeToggle'
import { cn } from '@/lib/utils'

const LINKS = [
  { to: '/', label: 'Holdings', icon: LayoutDashboard, end: true },
  { to: '/dividends', label: 'Dividends', icon: Coins, end: false },
  { to: '/transactions', label: 'Transactions', icon: Receipt, end: false },
  { to: '/admin', label: 'Admin', icon: Settings, end: false },
] as const

export function NavBar() {
  return (
    <header className="bg-card border-b">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-3">
        <div className="flex items-center gap-8">
          <span className="text-lg font-semibold">InvestInsight</span>
          <nav className="flex items-center gap-1">
            {LINKS.map(({ to, label, icon: Icon, end }) => (
              <NavLink
                key={to}
                to={to}
                end={end}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-primary text-primary-foreground'
                      : 'text-muted-foreground hover:text-foreground hover:bg-accent',
                  )
                }
              >
                <Icon className="size-4" />
                {label}
              </NavLink>
            ))}
          </nav>
        </div>
        <div className="flex items-center gap-3">
          <AccountScopeSwitcher />
          <ThemeToggle />
        </div>
      </div>
    </header>
  )
}
