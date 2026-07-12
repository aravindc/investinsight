import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  useReactTable,
  type ColumnFiltersState,
} from '@tanstack/react-table'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
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
import { useAccountScope } from '@/lib/account-scope'
import { fetchTransactions, type PortfolioTransaction } from '@/lib/api'

const ALL = '__all__'

const columnHelper = createColumnHelper<PortfolioTransaction>()

const columns = [
  columnHelper.accessor('timestamp', {
    header: 'Date',
    cell: (info) => new Date(info.getValue()).toLocaleDateString('en-GB'),
  }),
  columnHelper.accessor('entryType', {
    header: 'Type',
    cell: (info) => <Badge variant="secondary">{info.getValue()}</Badge>,
    filterFn: 'equals',
  }),
  columnHelper.accessor('instrumentName', {
    header: 'Instrument',
    cell: (info) => info.getValue() ?? '—',
    filterFn: 'equals',
  }),
  columnHelper.accessor('shares', {
    header: 'Shares',
  }),
  columnHelper.accessor('totalImpactGbp', {
    header: 'Impact (GBP)',
  }),
]

export function TransactionsTable() {
  const { scopeParam } = useAccountScope()
  const { data, isLoading, error } = useQuery({
    queryKey: ['transactions', scopeParam],
    queryFn: () => fetchTransactions(scopeParam!),
    enabled: !!scopeParam,
  })

  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])

  const entryTypes = useMemo(
    () => Array.from(new Set((data ?? []).map((t) => t.entryType))).sort(),
    [data],
  )
  const instrumentNames = useMemo(
    () =>
      Array.from(
        new Set((data ?? []).map((t) => t.instrumentName).filter((name) => !!name)),
      ).sort(),
    [data],
  )

  const table = useReactTable({
    data: data ?? [],
    columns,
    state: { columnFilters },
    onColumnFiltersChange: setColumnFilters,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>Recent transactions</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading && <p className="text-muted-foreground text-sm">Loading…</p>}
        {error && (
          <p className="text-destructive text-sm">Failed to load transactions.</p>
        )}
        {data && (
          <>
            <div className="mb-4 flex gap-3">
              <Select
                value={(table.getColumn('entryType')?.getFilterValue() as string) ?? ALL}
                onValueChange={(value) =>
                  table.getColumn('entryType')?.setFilterValue(value === ALL ? undefined : value)
                }
              >
                <SelectTrigger className="w-40">
                  <SelectValue placeholder="Type" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ALL}>All types</SelectItem>
                  {entryTypes.map((type) => (
                    <SelectItem key={type} value={type}>
                      {type}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <Select
                value={(table.getColumn('instrumentName')?.getFilterValue() as string) ?? ALL}
                onValueChange={(value) =>
                  table
                    .getColumn('instrumentName')
                    ?.setFilterValue(value === ALL ? undefined : value)
                }
              >
                <SelectTrigger className="w-56">
                  <SelectValue placeholder="Instrument" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ALL}>All instruments</SelectItem>
                  {instrumentNames.map((name) => (
                    <SelectItem key={name} value={name as string}>
                      {name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <Table>
              <TableHeader>
                {table.getHeaderGroups().map((headerGroup) => (
                  <TableRow key={headerGroup.id}>
                    {headerGroup.headers.map((header) => (
                      <TableHead key={header.id}>
                        {flexRender(header.column.columnDef.header, header.getContext())}
                      </TableHead>
                    ))}
                  </TableRow>
                ))}
              </TableHeader>
              <TableBody>
                {table.getRowModel().rows.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={columns.length} className="text-muted-foreground text-center">
                      No transactions match the selected filters.
                    </TableCell>
                  </TableRow>
                )}
                {table.getRowModel().rows.map((row) => (
                  <TableRow key={row.id}>
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </>
        )}
      </CardContent>
    </Card>
  )
}
