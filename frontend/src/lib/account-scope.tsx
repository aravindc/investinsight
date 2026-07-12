import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react'
import { fetchAccounts, fetchHouseholds, type Account, type Household } from '@/lib/api'

export type AccountScope =
  | { kind: 'account'; accountId: string }
  | { kind: 'household'; householdId: string }

const STORAGE_KEY = 'accountScope'

function scopeToParam(scope: AccountScope): string {
  return scope.kind === 'household' ? `household:${scope.householdId}` : scope.accountId
}

function loadStoredScope(): AccountScope | null {
  const raw = localStorage.getItem(STORAGE_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as AccountScope
  } catch {
    return null
  }
}

function scopeIsValid(scope: AccountScope, accounts: Account[]): boolean {
  if (scope.kind === 'household') {
    return accounts.some((a) => a.householdId === scope.householdId)
  }
  return accounts.some((a) => a.id === scope.accountId)
}

type AccountScopeContextValue = {
  scope: AccountScope | null
  setScope: (scope: AccountScope) => void
  accounts: Account[]
  households: Household[]
  scopeParam: string | null
}

const AccountScopeContext = createContext<AccountScopeContextValue | null>(null)

export function AccountProvider({ children }: { children: ReactNode }) {
  const [accounts, setAccounts] = useState<Account[]>([])
  const [households, setHouseholds] = useState<Household[]>([])
  const [scope, setScopeState] = useState<AccountScope | null>(null)

  useEffect(() => {
    Promise.all([fetchAccounts(), fetchHouseholds()]).then(([fetchedAccounts, fetchedHouseholds]) => {
      setAccounts(fetchedAccounts)
      setHouseholds(fetchedHouseholds)
      if (fetchedAccounts.length === 0) return

      const stored = loadStoredScope()
      if (stored && scopeIsValid(stored, fetchedAccounts)) {
        setScopeState(stored)
      } else {
        setScopeState({ kind: 'household', householdId: fetchedAccounts[0].householdId })
      }
    })
  }, [])

  const setScope = (next: AccountScope) => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
    setScopeState(next)
  }

  return (
    <AccountScopeContext.Provider
      value={{
        scope,
        setScope,
        accounts,
        households,
        scopeParam: scope ? scopeToParam(scope) : null,
      }}
    >
      {children}
    </AccountScopeContext.Provider>
  )
}

export function useAccountScope() {
  const ctx = useContext(AccountScopeContext)
  if (!ctx) throw new Error('useAccountScope must be used within an AccountProvider')
  return ctx
}
