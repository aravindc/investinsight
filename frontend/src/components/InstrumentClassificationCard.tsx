import { useMemo, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Download, TriangleAlert, Upload } from 'lucide-react'
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  fetchCategories,
  fetchInstruments,
  fetchSectors,
  importClassifications,
  updateInstrumentClassification,
  type Category,
  type ClassificationImportResult,
  type Instrument,
  type InstrumentClassification,
  type Sector,
} from '@/lib/api'

const CLASSIFICATION_KEYS = ['instruments', 'categories', 'sectors', 'sector-allocations', 'asset-categories', 'holdings-table']

const IMPORT_WARNING =
  'Importing overwrites the Category, Sub-Category, Sector and Sub-Sector of every instrument whose ' +
  'ticker appears in the file. Categories, sub-categories, sectors and sub-sectors that don\'t exist ' +
  'yet are created.'

// Radix Select can't use "" as an item value; this stands for "no sub-sector".
const NO_SUB_SECTOR = '__none__'

function csvCell(value: string) {
  return /[",\n\r]/.test(value) ? `"${value.replace(/"/g, '""')}"` : value
}

function exportCsv(instruments: Instrument[]) {
  const lines = [
    'Ticker,Name,Category,Sub-Category,Sector,Sub-Sector',
    ...[...instruments]
      .sort((a, b) => a.ticker.localeCompare(b.ticker) || a.name.localeCompare(b.name))
      .map((i) =>
        [i.ticker, i.name, i.category, i.subCategory, i.sector, i.subSector].map(csvCell).join(','),
      ),
  ]
  const url = URL.createObjectURL(new Blob([lines.join('\n') + '\n'], { type: 'text/csv' }))
  const a = document.createElement('a')
  a.href = url
  a.download = 'instrument-classifications.csv'
  a.click()
  URL.revokeObjectURL(url)
}

function InstrumentRow({
  instrument,
  categories,
  sectors,
}: {
  instrument: Instrument
  categories: Category[]
  sectors: Sector[]
}) {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: (next: InstrumentClassification) => updateInstrumentClassification(instrument.id, next),
    onSuccess: () => {
      for (const key of CLASSIFICATION_KEYS) queryClient.invalidateQueries({ queryKey: [key] })
    },
  })

  const current: InstrumentClassification = {
    category: instrument.category,
    subCategory: instrument.subCategory,
    sector: instrument.sector,
    subSector: instrument.subSector,
  }
  const subCategories = categories.find((c) => c.name === instrument.category)?.subCategories ?? []
  const subSectors = sectors.find((s) => s.name === instrument.sector)?.subSectors ?? []

  return (
    <TableRow>
      <TableCell className="font-medium">{instrument.ticker || '—'}</TableCell>
      <TableCell>
        {instrument.name || '—'}
        {mutation.isError && (
          <p className="text-destructive text-xs">
            {mutation.error instanceof Error ? mutation.error.message : 'Failed to update.'}
          </p>
        )}
      </TableCell>
      <TableCell>
        <Select
          value={instrument.category}
          disabled={mutation.isPending}
          onValueChange={(category) => {
            // Switching category picks that category's first sub-category.
            const first = categories.find((c) => c.name === category)?.subCategories[0]
            if (first) mutation.mutate({ ...current, category, subCategory: first.name })
          }}
        >
          <SelectTrigger className="w-40" aria-label={`Category of ${instrument.ticker}`}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {categories.map((c) => (
              <SelectItem key={c.name} value={c.name} disabled={c.subCategories.length === 0}>
                {c.name}
                {c.subCategories.length === 0 && ' (no sub-categories)'}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </TableCell>
      <TableCell>
        <Select
          value={instrument.subCategory}
          disabled={mutation.isPending}
          onValueChange={(subCategory) => mutation.mutate({ ...current, subCategory })}
        >
          <SelectTrigger className="w-40" aria-label={`Sub-category of ${instrument.ticker}`}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {subCategories.map((s) => (
              <SelectItem key={s.name} value={s.name}>
                {s.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </TableCell>
      <TableCell>
        <Select
          value={instrument.sector}
          disabled={mutation.isPending}
          // A sub-sector belongs to one sector, so changing sector clears it.
          onValueChange={(sector) => mutation.mutate({ ...current, sector, subSector: '' })}
        >
          <SelectTrigger className="w-52" aria-label={`Sector of ${instrument.ticker}`}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {sectors.map((s) => (
              <SelectItem key={s.name} value={s.name}>
                {s.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </TableCell>
      <TableCell>
        <Select
          value={instrument.subSector || NO_SUB_SECTOR}
          disabled={mutation.isPending || subSectors.length === 0}
          onValueChange={(value) =>
            mutation.mutate({ ...current, subSector: value === NO_SUB_SECTOR ? '' : value })
          }
        >
          <SelectTrigger className="w-44" aria-label={`Sub-sector of ${instrument.ticker}`}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={NO_SUB_SECTOR}>—</SelectItem>
            {subSectors.map((s) => (
              <SelectItem key={s.name} value={s.name}>
                {s.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </TableCell>
    </TableRow>
  )
}

export function InstrumentClassificationCard() {
  const queryClient = useQueryClient()
  const { data, isLoading, error } = useQuery({ queryKey: ['instruments'], queryFn: fetchInstruments })
  const fileInput = useRef<HTMLInputElement>(null)
  const importMutation = useMutation({
    mutationFn: importClassifications,
    onSuccess: () => {
      for (const key of CLASSIFICATION_KEYS) queryClient.invalidateQueries({ queryKey: [key] })
    },
    onSettled: () => {
      if (fileInput.current) fileInput.current.value = ''
    },
  })
  const { data: categories = [] } = useQuery({ queryKey: ['categories'], queryFn: fetchCategories })
  const { data: sectors = [] } = useQuery({ queryKey: ['sectors'], queryFn: fetchSectors })

  const [search, setSearch] = useState('')
  const filtered = useMemo(() => {
    const instruments = data ?? []
    const query = search.trim().toLowerCase()
    if (!query) return instruments
    return instruments.filter((i) =>
      [i.ticker, i.name, i.category, i.subCategory, i.sector, i.subSector].some((v) =>
        v.toLowerCase().includes(query),
      ),
    )
  }, [data, search])

  return (
    <Card>
      <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-4">
        <CardTitle>Instrument classification</CardTitle>
        <div className="flex flex-wrap items-center gap-2">
          <Input
            placeholder="Search ticker, name or classification…"
            className="w-72"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <Button variant="outline" onClick={() => exportCsv(data ?? [])} disabled={!data?.length}>
            <Download className="size-4" />
            Export CSV
          </Button>
          <Button
            variant="outline"
            onClick={() => fileInput.current?.click()}
            disabled={importMutation.isPending}
          >
            <Upload className="size-4" />
            {importMutation.isPending ? 'Importing…' : 'Import CSV'}
          </Button>
          <input
            ref={fileInput}
            type="file"
            accept=".csv,text/csv"
            className="hidden"
            onChange={(e) => {
              const file = e.target.files?.[0]
              if (!file) return
              if (window.confirm(`${IMPORT_WARNING}\n\nImport ${file.name}?`)) {
                importMutation.mutate(file)
              } else {
                e.target.value = ''
              }
            }}
          />
        </div>
      </CardHeader>
      <CardContent className="space-y-4 overflow-x-auto">
        <div className="flex gap-2 rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-sm">
          <TriangleAlert className="mt-0.5 size-4 shrink-0 text-amber-600 dark:text-amber-400" />
          <p>
            {IMPORT_WARNING} The CSV needs the columns{' '}
            <code className="text-xs">Ticker,Name,Category,Sub-Category,Sector</code>, plus an
            optional <code className="text-xs">Sub-Sector</code> (without it, sub-sectors are left
            as they are; a blank cell clears one). Instrument
            names aren't changed: Name only picks between instruments that share a ticker (e.g. BA).
            Export first to get a ready-made file to edit.
          </p>
        </div>

        {importMutation.isError && (
          <p className="text-destructive text-sm">
            {importMutation.error instanceof Error ? importMutation.error.message : 'Import failed.'}
          </p>
        )}
        {importMutation.data && <ImportSummary result={importMutation.data} />}

        {isLoading && <p className="text-muted-foreground text-sm">Loading…</p>}
        {!!error && <p className="text-destructive text-sm">Failed to load instruments.</p>}
        {!isLoading && !error && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Ticker</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Category</TableHead>
                <TableHead>Sub-Category</TableHead>
                <TableHead>Sector</TableHead>
                <TableHead>Sub-Sector</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="text-muted-foreground text-center">
                    No instruments match your search.
                  </TableCell>
                </TableRow>
              )}
              {filtered.map((instrument) => (
                <InstrumentRow
                  key={instrument.id}
                  instrument={instrument}
                  categories={categories}
                  sectors={sectors}
                />
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}

function ImportSummary({ result }: { result: ClassificationImportResult }) {
  const created = [
    ...result.createdCategories.map((n) => `category ${n}`),
    ...result.createdSubCategories.map((n) => `sub-category ${n}`),
    ...result.createdSectors.map((n) => `sector ${n}`),
    ...result.createdSubSectors.map((n) => `sub-sector ${n}`),
  ]
  return (
    <div className="space-y-1 rounded-lg border p-3 text-sm">
      <p>
        <span className="font-semibold">{result.updated}</span> instrument
        {result.updated === 1 ? '' : 's'} updated, {result.unchanged} already matched
        {result.skipped.length > 0 && (
          <>
            , <span className="text-destructive font-semibold">{result.skipped.length}</span> row
            {result.skipped.length === 1 ? '' : 's'} skipped
          </>
        )}
        .
      </p>
      {created.length > 0 && <p className="text-muted-foreground">Created {created.join(', ')}.</p>}
      {result.skipped.length > 0 && (
        <ul className="text-destructive list-disc pl-5">
          {result.skipped.slice(0, 10).map((s) => (
            <li key={`${s.line}-${s.ticker}`}>
              Line {s.line}
              {s.ticker && ` (${s.ticker})`}: {s.reason}
            </li>
          ))}
          {result.skipped.length > 10 && <li>…and {result.skipped.length - 10} more.</li>}
        </ul>
      )}
    </div>
  )
}
