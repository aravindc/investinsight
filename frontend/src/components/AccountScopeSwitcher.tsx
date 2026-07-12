import { useAccountScope } from '@/lib/account-scope'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

export function AccountScopeSwitcher() {
  const { scope, setScope, accounts, households } = useAccountScope()

  if (!scope || accounts.length === 0) return null

  const selectedHouseholdId =
    scope.kind === 'household'
      ? scope.householdId
      : (accounts.find((a) => a.id === scope.accountId)?.householdId ?? households[0]?.id)

  const accountsInHousehold = accounts.filter((a) => a.householdId === selectedHouseholdId)
  const accountValue = scope.kind === 'account' ? scope.accountId : 'household'

  return (
    <div className="flex items-center gap-2">
      <Select
        value={selectedHouseholdId}
        onValueChange={(householdId) => setScope({ kind: 'household', householdId })}
      >
        <SelectTrigger className="w-36">
          <SelectValue placeholder="Household" />
        </SelectTrigger>
        <SelectContent>
          {households.map((h) => (
            <SelectItem key={h.id} value={h.id}>
              {h.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Select
        value={accountValue}
        onValueChange={(v) =>
          setScope(
            v === 'household'
              ? { kind: 'household', householdId: selectedHouseholdId }
              : { kind: 'account', accountId: v },
          )
        }
      >
        <SelectTrigger className="w-40">
          <SelectValue placeholder="Account" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="household">All accounts</SelectItem>
          {accountsInHousehold.map((a) => (
            <SelectItem key={a.id} value={a.id}>
              {a.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
