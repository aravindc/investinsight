import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, Trash2, X } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { SortableHead } from '@/components/SortableHead'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  createSector,
  createSubSector,
  deleteSector,
  deleteSubSector,
  fetchSectors,
  type Sector,
} from '@/lib/api'
import { useNameCountSort } from '@/lib/use-name-count-sort'
import { colorForSector, iconForSector } from '@/lib/sector'

export function SectorAdminCard() {
  const queryClient = useQueryClient()
  const { data: sectors = [], isLoading, error } = useQuery({
    queryKey: ['sectors'],
    queryFn: fetchSectors,
  })
  const [name, setName] = useState('')

  const create = useMutation({
    mutationFn: () => createSector(name),
    onSuccess: () => {
      setName('')
      queryClient.invalidateQueries({ queryKey: ['sectors'] })
    },
  })
  const remove = useMutation({
    mutationFn: (sectorName: string) => deleteSector(sectorName),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['sectors'] }),
  })
  const mutationError = create.error ?? remove.error
  const { sorted, sort, toggle } = useNameCountSort(sectors)

  return (
    <Card>
      <CardHeader>
        <CardTitle>Sectors</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-muted-foreground text-sm">
          Sectors are independent of categories and drive the sector charts. Each sector can be
          split into optional sub-sectors (e.g. Financials → Banks, Insurance). Unused entries
          can be deleted.
        </p>

        <form
          className="flex flex-wrap gap-3"
          onSubmit={(e) => {
            e.preventDefault()
            create.mutate()
          }}
        >
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Aerospace & Defence"
            maxLength={50}
            className="max-w-sm"
            aria-label="New sector name"
          />
          <Button type="submit" disabled={!name.trim() || create.isPending}>
            {create.isPending ? 'Creating…' : 'Create sector'}
          </Button>
        </form>

        {mutationError && (
          <p className="text-destructive text-sm">
            {mutationError instanceof Error ? mutationError.message : 'Something went wrong.'}
          </p>
        )}
        {create.isSuccess && (
          <p className="text-sm">
            <span className="font-semibold">{create.data.name}</span> created. Assign it to
            instruments in the table below.
          </p>
        )}

        {isLoading && <p className="text-muted-foreground text-sm">Loading…</p>}
        {!!error && <p className="text-destructive text-sm">Failed to load sectors.</p>}
        {sectors.length > 0 && (
          <Table>
            <TableHeader>
              <TableRow>
                <SortableHead label="Sector" sortKey="name" sort={sort} onToggle={toggle} />
                <SortableHead
                  label="Instruments"
                  sortKey="count"
                  sort={sort}
                  onToggle={toggle}
                  className="text-right"
                />
                <TableHead className="w-12" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {sorted.map((s) => {
                const Icon = iconForSector(s.name)
                const deletable = s.instrumentCount === 0 && s.name !== 'Other'
                return (
                  <TableRow key={s.name}>
                    <TableCell className="space-y-2 whitespace-normal">
                      <span className="flex items-center gap-2 font-medium">
                        <Icon className="size-4" style={{ color: colorForSector(s.name) }} />
                        {s.name}
                      </span>
                      <SubSectorChips sector={s} />
                    </TableCell>
                    <TableCell className="text-right align-top tabular-nums">{s.instrumentCount}</TableCell>
                    <TableCell className="text-right align-top">
                      {deletable && (
                        <Button
                          size="icon-sm"
                          variant="ghost"
                          aria-label={`Delete ${s.name}`}
                          disabled={remove.isPending}
                          onClick={() => {
                            if (window.confirm(`Delete the "${s.name}" sector?`)) remove.mutate(s.name)
                          }}
                        >
                          <Trash2 className="size-4" />
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}

function SubSectorChips({ sector }: { sector: Sector }) {
  const queryClient = useQueryClient()
  const [name, setName] = useState('')
  const refresh = () => queryClient.invalidateQueries({ queryKey: ['sectors'] })

  const add = useMutation({
    mutationFn: () => createSubSector(sector.name, name),
    onSuccess: () => {
      setName('')
      refresh()
    },
  })
  const remove = useMutation({
    mutationFn: (sub: string) => deleteSubSector(sector.name, sub),
    onSuccess: refresh,
  })
  const error = add.error ?? remove.error

  return (
    <div className="space-y-1">
      <div className="flex flex-wrap items-center gap-1.5">
        {sector.subSectors.map((sub) => (
          <Badge key={sub.name} variant="secondary" className="gap-1 font-normal">
            {sub.name}
            <span className="text-muted-foreground">· {sub.instrumentCount}</span>
            {sub.instrumentCount === 0 && (
              <button
                type="button"
                aria-label={`Delete sub-sector ${sub.name}`}
                className="hover:text-destructive"
                disabled={remove.isPending}
                onClick={() => remove.mutate(sub.name)}
              >
                <X className="size-3" />
              </button>
            )}
          </Badge>
        ))}
        <form
          className="flex items-center gap-1"
          onSubmit={(e) => {
            e.preventDefault()
            add.mutate()
          }}
        >
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Add sub-sector"
            maxLength={50}
            className="h-6 w-32 text-xs"
            aria-label={`New sub-sector of ${sector.name}`}
          />
          <Button
            type="submit"
            size="icon-xs"
            variant="outline"
            aria-label="Add sub-sector"
            disabled={!name.trim() || add.isPending}
          >
            <Plus className="size-3" />
          </Button>
        </form>
      </div>
      {error && (
        <p className="text-destructive text-xs">
          {error instanceof Error ? error.message : 'Something went wrong.'}
        </p>
      )}
    </div>
  )
}
