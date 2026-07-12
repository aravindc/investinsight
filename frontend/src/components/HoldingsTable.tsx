import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
  type SortingState,
} from '@tanstack/react-table'
import { ArrowDown, ArrowUp, ArrowUpDown, Search, TrendingDown, TrendingUp } from 'lucide-react'
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
import { InstrumentLogo } from '@/components/InstrumentLogo'
import { useAccountScope } from '@/lib/account-scope'
import { fetchHoldingsTable, type HoldingRow } from '@/lib/api'
import { formatGbp } from '@/lib/format'

const PAGE_SIZES = [10, 25, 100]

function formatPercent(value: string | null): string {
  if (value === null) return 'N/A'
  return `${Number(value).toFixed(2)}%`
}

function GainCell({ amountGbp, percent }: { amountGbp: string | null; percent: string | null }) {
  if (amountGbp === null || percent === null) {
    return <span className="text-muted-foreground text-sm">N/A</span>
  }
  const positive = Number(amountGbp) >= 0
  const colorClass = positive ? 'text-green-600 dark:text-green-400' : 'text-destructive'
  return (
    <div className={colorClass}>
      <div className="font-medium">
        {positive ? '+' : ''}
        {formatGbp(amountGbp)}
      </div>
      <div className="flex items-center gap-1 text-xs">
        {positive ? <TrendingUp className="size-3" /> : <TrendingDown className="size-3" />}
        {formatPercent(percent)}
      </div>
    </div>
  )
}

function SignedPercentCell({ percent }: { percent: string | null }) {
  if (percent === null) {
    return <span className="text-muted-foreground text-sm">N/A</span>
  }
  const positive = Number(percent) >= 0
  const colorClass = positive ? 'text-green-600 dark:text-green-400' : 'text-destructive'
  return (
    <div className={`flex items-center gap-1 font-medium ${colorClass}`}>
      {positive ? <TrendingUp className="size-3" /> : <TrendingDown className="size-3" />}
      {formatPercent(percent)}
    </div>
  )
}

const columnHelper = createColumnHelper<HoldingRow>()

const columns = [
  columnHelper.accessor('name', {
    header: 'Holding',
    cell: (info) => {
      const row = info.row.original
      return (
        <div className="flex items-center gap-3">
          <InstrumentLogo ticker={row.ticker} />
          <div>
            <div className="font-medium">{row.name}</div>
            <div className="text-muted-foreground text-xs">{row.ticker}</div>
          </div>
        </div>
      )
    },
  }),
  columnHelper.accessor((row) => Number(row.shares), {
    id: 'shares',
    header: 'Shares',
    cell: (info) => Number(info.getValue()).toLocaleString('en-GB', { maximumFractionDigits: 4 }),
  }),
  columnHelper.accessor((row) => Number(row.costBasisGbp), {
    id: 'costBasis',
    header: 'Cost basis',
    cell: (info) => {
      const row = info.row.original
      return (
        <div>
          <div className="font-medium">{formatGbp(row.costBasisGbp)}</div>
          <div className="text-muted-foreground text-xs">{formatGbp(row.costBasisPerShareGbp)}</div>
        </div>
      )
    },
  }),
  columnHelper.accessor((row) => Number(row.currentValueGbp), {
    id: 'currentValue',
    header: 'Current value',
    cell: (info) => {
      const row = info.row.original
      if (!row.priced) return <span className="text-muted-foreground text-sm">N/A</span>
      return (
        <div>
          <div className="font-medium">{formatGbp(row.currentValueGbp)}</div>
          <div className="text-muted-foreground text-xs">{formatGbp(row.currentValuePerShareGbp)}</div>
        </div>
      )
    },
  }),
  columnHelper.accessor((row) => Number(row.dividendsReceivedGbp), {
    id: 'dividendsReceived',
    header: 'Dividends',
    cell: (info) => {
      const row = info.row.original
      return (
        <div>
          <div className="font-medium">{formatGbp(row.dividendsReceivedGbp)}</div>
          <div className="text-muted-foreground text-xs">{formatGbp(row.dividendsReceivedPerShareGbp)}</div>
        </div>
      )
    },
  }),
  columnHelper.accessor((row) => (row.dividendYieldPercent === null ? null : Number(row.dividendYieldPercent)), {
    id: 'dividendYield',
    header: 'Dividend yield',
    cell: (info) => (
      <span className="font-medium">{formatPercent(info.row.original.dividendYieldPercent)}</span>
    ),
  }),
  columnHelper.accessor(
    (row) => (row.dividendGrowthPercent === null ? null : Number(row.dividendGrowthPercent)),
    {
      id: 'dividendGrowth',
      header: 'Dividend growth (5Y)',
      cell: (info) => <SignedPercentCell percent={info.row.original.dividendGrowthPercent} />,
    },
  ),
  columnHelper.accessor((row) => (row.totalProfitGbp === null ? null : Number(row.totalProfitGbp)), {
    id: 'totalProfit',
    header: 'Total profit',
    cell: (info) => {
      const row = info.row.original
      return <GainCell amountGbp={row.totalProfitGbp} percent={row.totalProfitPercent} />
    },
  }),
  columnHelper.accessor((row) => (row.irrPercent === null ? null : Number(row.irrPercent)), {
    id: 'irr',
    header: 'IRR',
    cell: (info) => <SignedPercentCell percent={info.row.original.irrPercent} />,
  }),
  columnHelper.accessor(
    (row) => (row.portfolioSharePercent === null ? null : Number(row.portfolioSharePercent)),
    {
      id: 'portfolioShare',
      header: 'Share in portfolio',
      cell: (info) => (
        <span className="font-medium">{formatPercent(info.row.original.portfolioSharePercent)}</span>
      ),
    },
  ),
]

export function HoldingsTable() {
  const { scopeParam } = useAccountScope()
  const { data, isLoading, error } = useQuery({
    queryKey: ['holdings-table', scopeParam],
    queryFn: () => fetchHoldingsTable(scopeParam!),
    enabled: !!scopeParam,
  })

  const [search, setSearch] = useState('')
  const [sorting, setSorting] = useState<SortingState>([{ id: 'currentValue', desc: true }])
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: 10 })

  const filtered = useMemo(() => {
    const rows = data ?? []
    const query = search.trim().toLowerCase()
    if (!query) return rows
    return rows.filter(
      (r) => r.name.toLowerCase().includes(query) || r.ticker.toLowerCase().includes(query),
    )
  }, [data, search])

  const table = useReactTable({
    data: filtered,
    columns,
    state: { sorting, pagination },
    onSortingChange: setSorting,
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
  })

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-4">
        <CardTitle>Holdings</CardTitle>
        <div className="relative w-64">
          <Search className="text-muted-foreground absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
          <Input
            placeholder="Search…"
            className="pl-8"
            value={search}
            onChange={(e) => {
              setSearch(e.target.value)
              setPagination((p) => ({ ...p, pageIndex: 0 }))
            }}
          />
        </div>
      </CardHeader>
      <CardContent>
        {isLoading && <p className="text-muted-foreground text-sm">Loading…</p>}
        {error && <p className="text-destructive text-sm">Failed to load holdings.</p>}
        {!isLoading && !error && (
          <>
            <Table>
              <TableHeader>
                {table.getHeaderGroups().map((headerGroup) => (
                  <TableRow key={headerGroup.id}>
                    {headerGroup.headers.map((header) => (
                      <TableHead
                        key={header.id}
                        className="cursor-pointer select-none"
                        onClick={header.column.getToggleSortingHandler()}
                      >
                        <div className="flex items-center gap-1">
                          {flexRender(header.column.columnDef.header, header.getContext())}
                          {header.column.getIsSorted() === 'asc' && <ArrowUp className="size-3" />}
                          {header.column.getIsSorted() === 'desc' && <ArrowDown className="size-3" />}
                          {!header.column.getIsSorted() && (
                            <ArrowUpDown className="text-muted-foreground/50 size-3" />
                          )}
                        </div>
                      </TableHead>
                    ))}
                  </TableRow>
                ))}
              </TableHeader>
              <TableBody>
                {table.getRowModel().rows.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={columns.length} className="text-muted-foreground text-center">
                      No holdings match your search.
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

            <div className="mt-4 flex items-center justify-between">
              <div className="text-muted-foreground text-sm">
                {filtered.length === 0
                  ? '0 holdings'
                  : `Showing ${table.getRowModel().rows.length} of ${filtered.length} holdings`}
              </div>
              <div className="flex items-center gap-4">
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground text-sm">Rows per page</span>
                  <Select
                    value={String(pagination.pageSize)}
                    onValueChange={(v) => setPagination({ pageIndex: 0, pageSize: Number(v) })}
                  >
                    <SelectTrigger className="w-20">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {PAGE_SIZES.map((size) => (
                        <SelectItem key={size} value={String(size)}>
                          {size}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    className="border-input hover:bg-accent rounded-md border px-3 py-1.5 text-sm disabled:pointer-events-none disabled:opacity-50"
                    onClick={() => table.previousPage()}
                    disabled={!table.getCanPreviousPage()}
                  >
                    Previous
                  </button>
                  <span className="text-muted-foreground text-sm">
                    Page {table.getState().pagination.pageIndex + 1} of {Math.max(table.getPageCount(), 1)}
                  </span>
                  <button
                    type="button"
                    className="border-input hover:bg-accent rounded-md border px-3 py-1.5 text-sm disabled:pointer-events-none disabled:opacity-50"
                    onClick={() => table.nextPage()}
                    disabled={!table.getCanNextPage()}
                  >
                    Next
                  </button>
                </div>
              </div>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}
