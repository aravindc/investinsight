import { useAccountScope } from '@/lib/account-scope'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

export function AccountScopeSwitcher() {
  const { scope, setScope, accounts } = useAccountScope()

  if (!scope || accounts.length === 0) return null

  const householdId = accounts[0].householdId
  const value = scope.kind === 'household' ? 'household' : scope.accountId

  return (
    <Select
      value={value}
      onValueChange={(v) =>
        setScope(
          v === 'household'
            ? { kind: 'household', householdId }
            : { kind: 'account', accountId: v },
        )
      }
    >
      <SelectTrigger className="w-44">
        <SelectValue placeholder="Account" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="household">All accounts</SelectItem>
        {accounts.map((a) => (
          <SelectItem key={a.id} value={a.id}>
            {a.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
