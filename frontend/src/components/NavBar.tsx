import { useEffect, useRef } from 'react'
import {
  ChartNoAxesCombined,
  Coins,
  LayoutDashboard,
  Receipt,
  Settings,
} from 'lucide-react'
import { NavLink, useLocation } from 'react-router-dom'
import { AccountScopeSwitcher } from '@/components/AccountScopeSwitcher'
import { ThemeToggle } from '@/components/ThemeToggle'
import { cn } from '@/lib/utils'

const LINKS = [
  { to: '/', label: 'Holdings', icon: LayoutDashboard, end: true },
  { to: '/dividends', label: 'Dividends', icon: Coins, end: false },
  { to: '/transactions', label: 'Transactions', icon: Receipt, end: false },
  { to: '/analytics', label: 'Analytics', icon: ChartNoAxesCombined, end: false },
  { to: '/admin', label: 'Admin', icon: Settings, end: false },
] as const

function NavItem({ to, label, icon: Icon, end }: (typeof LINKS)[number]) {
  return (
    <NavLink
      to={to}
      end={end}
      className={({ isActive }) =>
        cn(
          'flex shrink-0 items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
          isActive
            ? 'bg-primary text-primary-foreground'
            : 'text-muted-foreground hover:text-foreground hover:bg-accent',
        )
      }
    >
      <Icon className="size-4" />
      {label}
    </NavLink>
  )
}

export function NavBar() {
  // On narrow screens the link row scrolls; keep the current page's link in view.
  const scrollingNav = useRef<HTMLElement>(null)
  const { pathname } = useLocation()
  useEffect(() => {
    scrollingNav.current
      ?.querySelector('[aria-current="page"]')
      ?.scrollIntoView({ block: 'nearest', inline: 'center' })
  }, [pathname])

  return (
    <header className="bg-card border-b">
      {/* On narrow screens the links wrap onto their own row and scroll. */}
      <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-x-8 gap-y-2 px-4 py-3 sm:px-6">
        <div className="flex min-w-0 max-w-full items-center gap-8">
          <span className="text-lg font-semibold">InvestInsight</span>
          <nav className="flex min-w-0 items-center gap-1 overflow-x-auto max-lg:hidden">
            {LINKS.map((link) => (
              <NavItem key={link.to} {...link} />
            ))}
          </nav>
        </div>
        <div className="flex items-center gap-3">
          <AccountScopeSwitcher />
          <ThemeToggle />
        </div>
        <nav ref={scrollingNav} className="-mx-1 flex w-full items-center gap-1 overflow-x-auto lg:hidden">
          {LINKS.map((link) => (
            <NavItem key={link.to} {...link} />
          ))}
        </nav>
      </div>
    </header>
  )
}
