import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  createAccount,
  createHousehold,
  fetchHouseholds,
  type Household,
} from '@/lib/api'

export function HouseholdAccountCard() {
  const queryClient = useQueryClient()

  const { data: households = [] } = useQuery({
    queryKey: ['households'],
    queryFn: fetchHouseholds,
  })

  const [householdName, setHouseholdName] = useState('')
  const [accountName, setAccountName] = useState('')
  const [selectedHouseholdId, setSelectedHouseholdId] = useState<string>('')

  const effectiveHouseholdId = selectedHouseholdId || households[0]?.id || ''

  const createHouseholdMutation = useMutation({
    mutationFn: (name: string) => createHousehold(name),
    onSuccess: (household: Household) => {
      queryClient.invalidateQueries({ queryKey: ['households'] })
      setSelectedHouseholdId(household.id)
      setHouseholdName('')
    },
  })

  const createAccountMutation = useMutation({
    mutationFn: () => createAccount(effectiveHouseholdId, accountName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['accounts'] })
      setAccountName('')
    },
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>Households &amp; accounts</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="space-y-2">
          <p className="text-sm font-medium">New household</p>
          <div className="flex gap-3">
            <Input
              value={householdName}
              onChange={(e) => setHouseholdName(e.target.value)}
              placeholder="e.g. Smith Family"
              className="max-w-sm"
            />
            <Button
              onClick={() => createHouseholdMutation.mutate(householdName)}
              disabled={!householdName.trim() || createHouseholdMutation.isPending}
            >
              {createHouseholdMutation.isPending ? 'Creating…' : 'Create household'}
            </Button>
          </div>
          {createHouseholdMutation.isError && (
            <p className="text-destructive text-sm">
              {createHouseholdMutation.error instanceof Error
                ? createHouseholdMutation.error.message
                : 'Failed to create household.'}
            </p>
          )}
        </div>

        <div className="space-y-2 border-t pt-4">
          <p className="text-sm font-medium">New account</p>
          <div className="flex gap-3">
            <Select value={effectiveHouseholdId} onValueChange={setSelectedHouseholdId}>
              <SelectTrigger className="w-48">
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
            <Input
              value={accountName}
              onChange={(e) => setAccountName(e.target.value)}
              placeholder="e.g. Ara ISA"
              className="max-w-sm"
            />
            <Button
              onClick={() => createAccountMutation.mutate()}
              disabled={
                !accountName.trim() ||
                !effectiveHouseholdId ||
                createAccountMutation.isPending
              }
            >
              {createAccountMutation.isPending ? 'Creating…' : 'Create account'}
            </Button>
          </div>
          {households.length === 0 && (
            <p className="text-muted-foreground text-sm">
              Create a household first.
            </p>
          )}
          {createAccountMutation.isError && (
            <p className="text-destructive text-sm">
              {createAccountMutation.error instanceof Error
                ? createAccountMutation.error.message
                : 'Failed to create account.'}
            </p>
          )}
          {createAccountMutation.isSuccess && (
            <p className="text-sm">
              <span className="font-semibold">{createAccountMutation.data.name}</span>{' '}
              created.
            </p>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
