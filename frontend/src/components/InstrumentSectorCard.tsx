import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { fetchInstruments, updateInstrumentSector, type Instrument } from '@/lib/api'
import { SECTOR_ORDER } from '@/lib/sector'

function SectorCell({ instrument }: { instrument: Instrument }) {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: (sectorName: string) => updateInstrumentSector(instrument.id, sectorName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['instruments'] })
      queryClient.invalidateQueries({ queryKey: ['sector-allocations'] })
      queryClient.invalidateQueries({ queryKey: ['asset-categories'] })
      queryClient.invalidateQueries({ queryKey: ['holdings-table'] })
    },
  })

  return (
    <div className="space-y-1">
      <Select
        value={instrument.sector}
        onValueChange={(value) => mutation.mutate(value)}
        disabled={mutation.isPending}
      >
        <SelectTrigger className="w-56">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {SECTOR_ORDER.map((s) => (
            <SelectItem key={s} value={s}>
              {s}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {mutation.isError && (
        <p className="text-destructive text-xs">
          {mutation.error instanceof Error ? mutation.error.message : 'Failed to update sector.'}
        </p>
      )}
    </div>
  )
}

export function InstrumentSectorCard() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['instruments'],
    queryFn: fetchInstruments,
  })

  const [search, setSearch] = useState('')

  const filtered = useMemo(() => {
    const instruments = data ?? []
    const query = search.trim().toLowerCase()
    if (!query) return instruments
    return instruments.filter(
      (i) => i.ticker.toLowerCase().includes(query) || i.name.toLowerCase().includes(query),
    )
  }, [data, search])

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-4">
        <CardTitle>Instrument sectors</CardTitle>
        <Input
          placeholder="Search ticker or name…"
          className="max-w-xs"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </CardHeader>
      <CardContent>
        {isLoading && <p className="text-muted-foreground text-sm">Loading…</p>}
        {error && <p className="text-destructive text-sm">Failed to load instruments.</p>}
        {!isLoading && !error && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Ticker</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Sector</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.length === 0 && (
                <TableRow>
                  <TableCell colSpan={3} className="text-muted-foreground text-center">
                    No instruments match your search.
                  </TableCell>
                </TableRow>
              )}
              {filtered.map((instrument) => (
                <TableRow key={instrument.id}>
                  <TableCell className="font-medium">{instrument.ticker || '—'}</TableCell>
                  <TableCell>{instrument.name || '—'}</TableCell>
                  <TableCell>
                    <SectorCell instrument={instrument} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}
