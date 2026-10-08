import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  deleteCashIsa,
  fetchAccounts,
  fetchCashIsas,
  saveCashIsa,
  setCashIsaRate,
  type Account,
  type CashIsa,
} from '@/lib/api'
import { formatGbp } from '@/lib/format'

const CASH_ISAS_KEY = 'cash-isas'

function todayIso() {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function formatDate(iso: string) {
  return new Date(`${iso}T00:00:00`).toLocaleDateString(undefined, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

export function CashIsaAdminCard() {
  const { data: accounts = [] } = useQuery({ queryKey: ['accounts'], queryFn: fetchAccounts })
  const { data } = useQuery({ queryKey: [CASH_ISAS_KEY, 'all'], queryFn: () => fetchCashIsas() })
  const byAccount = new Map((data?.cashIsas ?? []).map((c) => [c.accountId, c]))

  return (
    <Card>
      <CardHeader>
        <CardTitle>Cash ISAs</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-muted-foreground text-sm">
          Enter each account's Cash ISA balance and interest rate. Interest accrues daily
          (rate ÷ 365) and is added to the balance at the end of each month. After a deposit or
          withdrawal, update the balance with the date it applies to.
        </p>
        {accounts.length === 0 && (
          <p className="text-muted-foreground text-sm">Create an account first.</p>
        )}
        <div className="divide-y">
          {accounts.map((account) => (
            <CashIsaRow key={account.id} account={account} isa={byAccount.get(account.id)} />
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

type Mode = 'none' | 'balance' | 'rate'

function CashIsaRow({ account, isa }: { account: Account; isa: CashIsa | undefined }) {
  const queryClient = useQueryClient()
  const [mode, setMode] = useState<Mode>('none')
  const [balance, setBalance] = useState('')
  const [date, setDate] = useState(todayIso())
  const [rate, setRate] = useState('')

  const refresh = () => {
    setMode('none')
    // Cash ISA values feed the Holdings cards too.
    queryClient.invalidateQueries({ queryKey: [CASH_ISAS_KEY] })
  }

  const save = useMutation({ mutationFn: () => saveCashIsa(account.id, balance, date, rate), onSuccess: refresh })
  const changeRate = useMutation({ mutationFn: () => setCashIsaRate(account.id, rate, date), onSuccess: refresh })
  const remove = useMutation({ mutationFn: () => deleteCashIsa(account.id), onSuccess: refresh })
  const error = save.error ?? changeRate.error ?? remove.error

  const open = (next: Mode) => {
    setDate(todayIso())
    setBalance(next === 'balance' && isa ? isa.valueGbp : '')
    setRate(isa ? isa.currentRatePercent : '')
    setMode(mode === next ? 'none' : next)
  }

  return (
    <div className="space-y-3 py-4 first:pt-0 last:pb-0">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
        <div className="min-w-0 flex-1">
          <p className="font-medium">{account.name}</p>
          {isa ? (
            <p className="text-muted-foreground text-sm">
              {formatGbp(isa.valueGbp)} at {isa.currentRatePercent}% ·{' '}
              {formatGbp(isa.accruedThisMonthGbp)} accrued, credited {formatDate(isa.nextCreditDate)}
            </p>
          ) : (
            <p className="text-muted-foreground text-sm">No Cash ISA</p>
          )}
        </div>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" onClick={() => open('balance')}>
            {mode === 'balance' ? 'Cancel' : isa ? 'Update balance' : 'Add Cash ISA'}
          </Button>
          {isa && (
            <>
              <Button size="sm" variant="outline" onClick={() => open('rate')}>
                {mode === 'rate' ? 'Cancel' : 'Change rate'}
              </Button>
              <Button
                size="sm"
                variant="ghost"
                disabled={remove.isPending}
                onClick={() => {
                  if (window.confirm(`Remove the Cash ISA for ${account.name}?`)) remove.mutate()
                }}
              >
                Remove
              </Button>
            </>
          )}
        </div>
      </div>

      {mode === 'balance' && (
        <form
          className="flex flex-wrap items-end gap-3"
          onSubmit={(e) => {
            e.preventDefault()
            save.mutate()
          }}
        >
          <LabelledInput label="Balance (£)" type="number" min="0" step="0.01" value={balance} onChange={setBalance} />
          <LabelledInput label="Balance as of" type="date" max={todayIso()} value={date} onChange={setDate} />
          <LabelledInput label="Interest rate (% a year)" type="number" min="0" max="100" step="0.001" value={rate} onChange={setRate} />
          <Button type="submit" disabled={!balance || !date || !rate || save.isPending}>
            {save.isPending ? 'Saving…' : 'Save'}
          </Button>
        </form>
      )}

      {mode === 'rate' && (
        <form
          className="flex flex-wrap items-end gap-3"
          onSubmit={(e) => {
            e.preventDefault()
            changeRate.mutate()
          }}
        >
          <LabelledInput label="New rate (% a year)" type="number" min="0" max="100" step="0.001" value={rate} onChange={setRate} />
          <LabelledInput label="Effective from" type="date" max={todayIso()} value={date} onChange={setDate} />
          <Button type="submit" disabled={!rate || !date || changeRate.isPending}>
            {changeRate.isPending ? 'Saving…' : 'Save rate'}
          </Button>
        </form>
      )}

      {isa && (
        <p className="text-muted-foreground text-xs">
          Balance {formatGbp(isa.balanceGbp)} entered for {formatDate(isa.balanceDate)}
          {Number(isa.interestCreditedGbp) > 0 && <>, {formatGbp(isa.interestCreditedGbp)} interest credited since</>}
          . Rates:{' '}
          {isa.rates.map((r) => `${r.ratePercent}% from ${formatDate(r.effectiveFrom)}`).join(', ')}.
        </p>
      )}

      {error && (
        <p className="text-destructive text-sm">
          {error instanceof Error ? error.message : 'Something went wrong.'}
        </p>
      )}
    </div>
  )
}

function LabelledInput({
  label,
  value,
  onChange,
  ...props
}: Omit<React.ComponentProps<typeof Input>, 'onChange' | 'value'> & {
  label: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <label className="space-y-1 text-sm">
      <span className="text-muted-foreground block text-xs">{label}</span>
      <Input {...props} value={value} onChange={(e) => onChange(e.target.value)} className="w-40" required />
    </label>
  )
}
