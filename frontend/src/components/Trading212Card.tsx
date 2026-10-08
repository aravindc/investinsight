import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  connectTrading212,
  disconnectTrading212,
  fetchAccounts,
  fetchHouseholds,
  fetchTrading212Connections,
  syncTrading212,
  type Account,
  type Trading212Connection,
} from '@/lib/api'

const CONNECTIONS_KEY = ['trading212-connections']

function formatDateTime(iso: string | null) {
  return iso ? new Date(iso).toLocaleString() : '—'
}

export function Trading212Card() {
  const queryClient = useQueryClient()
  const { data: accounts = [] } = useQuery({ queryKey: ['accounts'], queryFn: fetchAccounts })
  const { data: households = [] } = useQuery({ queryKey: ['households'], queryFn: fetchHouseholds })
  const { data: connections = [] } = useQuery({
    queryKey: CONNECTIONS_KEY,
    queryFn: fetchTrading212Connections,
    // Poll while any sync runs, so progress and the result show up.
    refetchInterval: (query) =>
      query.state.data?.some((c) => c.syncStatus === 'running') ? 5000 : false,
  })

  // When a sync finishes, refresh everything derived from transactions.
  const wasRunning = useRef(new Set<string>())
  useEffect(() => {
    let finished = false
    for (const c of connections) {
      if (c.syncStatus === 'running') wasRunning.current.add(c.accountId)
      else if (wasRunning.current.delete(c.accountId)) finished = true
    }
    if (finished) {
      queryClient.invalidateQueries({
        predicate: (q) => q.queryKey[0] !== CONNECTIONS_KEY[0],
      })
    }
  }, [connections, queryClient])

  const householdName = (id: string) => households.find((h) => h.id === id)?.name ?? ''
  const byAccount = new Map(connections.map((c) => [c.accountId, c]))

  return (
    <Card>
      <CardHeader>
        <CardTitle>Trading 212 sync</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-muted-foreground text-sm">
          Connect each account to its own Trading 212 API key to import transactions directly.
          Create a key in the Trading 212 app under Settings → API, with the account data,
          history and export permissions. Keys are stored encrypted and never shown again.
        </p>
        {accounts.length === 0 && (
          <p className="text-muted-foreground text-sm">Create an account first.</p>
        )}
        <div className="divide-y">
          {accounts.map((account) => (
            <AccountRow
              key={account.id}
              account={account}
              householdName={householdName(account.householdId)}
              connection={byAccount.get(account.id)}
            />
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function AccountRow({
  account,
  householdName,
  connection,
}: {
  account: Account
  householdName: string
  connection: Trading212Connection | undefined
}) {
  const queryClient = useQueryClient()
  const [editing, setEditing] = useState(false)
  const [apiKey, setApiKey] = useState('')
  const [apiSecret, setApiSecret] = useState('')

  const refresh = () => queryClient.invalidateQueries({ queryKey: CONNECTIONS_KEY })

  const connect = useMutation({
    mutationFn: () => connectTrading212(account.id, apiKey.trim(), apiSecret.trim()),
    onSuccess: () => {
      setEditing(false)
      setApiKey('')
      setApiSecret('')
      refresh()
    },
  })
  const disconnect = useMutation({
    mutationFn: () => disconnectTrading212(account.id),
    onSuccess: refresh,
  })
  const sync = useMutation({
    mutationFn: () => syncTrading212(account.id),
    onSettled: refresh,
  })

  const running = connection?.syncStatus === 'running'
  const error = connect.error ?? disconnect.error ?? sync.error

  return (
    <div className="space-y-3 py-4 first:pt-0 last:pb-0">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
        <div className="min-w-0 flex-1">
          <p className="font-medium">{account.name}</p>
          {householdName && <p className="text-muted-foreground text-xs">{householdName}</p>}
        </div>

        {connection ? (
          <Badge variant="secondary">
            Connected ••••{connection.keyHint}
            {connection.t212AccountId !== null && ` · #${connection.t212AccountId}`}
          </Badge>
        ) : (
          <Badge variant="outline">Not connected</Badge>
        )}

        <div className="flex gap-2">
          {connection && (
            <Button size="sm" onClick={() => sync.mutate()} disabled={running || sync.isPending}>
              <RefreshCw className={running ? 'size-4 animate-spin' : 'size-4'} />
              {running ? 'Syncing…' : 'Sync now'}
            </Button>
          )}
          <Button size="sm" variant="outline" onClick={() => setEditing((v) => !v)} disabled={running}>
            {editing ? 'Cancel' : connection ? 'Replace key' : 'Connect'}
          </Button>
          {connection && (
            <Button
              size="sm"
              variant="ghost"
              disabled={running || disconnect.isPending}
              onClick={() => {
                if (window.confirm(`Remove the Trading 212 key for ${account.name}? Imported transactions are kept.`)) {
                  disconnect.mutate()
                }
              }}
            >
              Remove
            </Button>
          )}
        </div>
      </div>

      {editing && (
        <form
          className="flex flex-wrap gap-3"
          autoComplete="off"
          onSubmit={(e) => {
            e.preventDefault()
            connect.mutate()
          }}
        >
          <Input
            type="password"
            autoComplete="off"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder="API key"
            aria-label="Trading 212 API key"
            className="max-w-xs"
          />
          <Input
            type="password"
            autoComplete="off"
            value={apiSecret}
            onChange={(e) => setApiSecret(e.target.value)}
            placeholder="API secret"
            aria-label="Trading 212 API secret"
            className="max-w-xs"
          />
          <Button type="submit" disabled={!apiKey.trim() || !apiSecret.trim() || connect.isPending}>
            {connect.isPending ? 'Verifying…' : 'Save'}
          </Button>
        </form>
      )}

      {connection && (
        <div className="text-muted-foreground space-y-0.5 text-sm">
          {connection.syncMessage && (
            <p className={connection.syncStatus === 'failed' ? 'text-destructive' : undefined}>
              {connection.syncStatus === 'failed' && 'Last sync failed: '}
              {connection.syncMessage}
            </p>
          )}
          <p>
            Last sync: {formatDateTime(connection.syncFinishedAt ?? connection.syncStartedAt)}
            {connection.syncedThrough && <> · synced through {formatDateTime(connection.syncedThrough)}</>}
          </p>
          {running && (
            <p>
              Trading 212 limits how fast reports can be requested, so a sync takes at least a
              minute per year of history.
            </p>
          )}
        </div>
      )}

      {error && (
        <p className="text-destructive text-sm">
          {error instanceof Error ? error.message : 'Something went wrong.'}
        </p>
      )}
    </div>
  )
}
