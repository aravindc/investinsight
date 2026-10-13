import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ChevronLeft, ChevronRight, HelpCircle } from 'lucide-react'
import { DividendIncomeByMonth } from '@/components/DividendIncomeByMonth'
import { DividendStatusMarker } from '@/components/DividendStatusMarker'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { useAccountScope } from '@/lib/account-scope'
import { fetchDividendCalendar, type DividendEvent } from '@/lib/api'
import {
  monthDate,
  monthKey,
  monthsBetween,
  placedOnExDate,
  STATUS_META,
  STATUS_ORDER,
  sumAmounts,
} from '@/lib/dividend-calendar'
import { formatGbp } from '@/lib/format'
import { cn } from '@/lib/utils'

const WEEKDAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
// Events listed inside a day cell before collapsing to "+N more".
const MAX_IN_CELL = 2

const monthTitle = new Intl.DateTimeFormat('en-GB', { month: 'long', year: 'numeric', timeZone: 'UTC' })
const dayLabel = new Intl.DateTimeFormat('en-GB', { weekday: 'short', day: 'numeric', month: 'short', timeZone: 'UTC' })

function isoDay(year: number, month: number, day: number) {
  return `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`
}

/** The month's days, Monday-first, padded with nulls to whole weeks. */
function monthGrid(key: string): (string | null)[] {
  const [y, m] = key.split('-').map(Number)
  const first = new Date(Date.UTC(y, m - 1, 1))
  const days = new Date(Date.UTC(y, m, 0)).getUTCDate()
  const lead = (first.getUTCDay() + 6) % 7
  const trail = (7 - ((lead + days) % 7)) % 7
  return [
    ...Array<null>(lead).fill(null),
    ...Array.from({ length: days }, (_, i) => isoDay(y, m, i + 1)),
    ...Array<null>(trail).fill(null),
  ]
}

function Amount({ event }: { event: DividendEvent }) {
  if (event.amountGbp === null) {
    return <span className="text-muted-foreground text-xs font-normal">Amount not yet known</span>
  }
  return <span className="tabular-nums">{formatGbp(event.amountGbp)}</span>
}

export function DividendCalendarCard() {
  const { scopeParam } = useAccountScope()
  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ['dividend-calendar', scopeParam],
    queryFn: () => fetchDividendCalendar(scopeParam!),
    enabled: !!scopeParam,
  })
  const [month, setMonth] = useState<string | null>(null)
  const [selectedDay, setSelectedDay] = useState<string | null>(null)

  const months = useMemo(() => (data ? monthsBetween(data.from, data.to) : []), [data])
  const current = month ?? (data ? monthKey(data.today) : null)
  const index = current ? months.indexOf(current) : -1
  const go = (key: string) => {
    setMonth(key)
    setSelectedDay(null)
  }

  const byDay = useMemo(() => {
    const map = new Map<string, DividendEvent[]>()
    for (const e of data?.events ?? []) map.set(e.date, [...(map.get(e.date) ?? []), e])
    return map
  }, [data])

  const monthEvents = current ? (data?.events ?? []).filter((e) => monthKey(e.date) === current) : []
  const listed = selectedDay ? monthEvents.filter((e) => e.date === selectedDay) : monthEvents

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-1.5">
          Dividend calendar
          <Tooltip>
            <TooltipTrigger asChild>
              <HelpCircle className="text-muted-foreground size-3.5" />
            </TooltipTrigger>
            <TooltipContent className="max-w-xs">
              Paid dividends come from your transactions. Upcoming ones repeat each holding's
              payments from the last 12 months, scaled to the shares you hold now. Where Yahoo
              Finance lists an announced dividend, its dates replace the estimate's.
            </TooltipContent>
          </Tooltip>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-8">
        {isLoading && (
          <div className="bg-muted h-[420px] animate-pulse rounded-lg" aria-busy="true">
            <span className="sr-only">Loading dividend calendar…</span>
          </div>
        )}
        {!!error && !data && (
          <div className="flex items-center gap-3 text-sm" role="alert">
            <span className="text-destructive">Failed to load the dividend calendar.</span>
            <Button variant="outline" size="sm" disabled={isFetching} onClick={() => refetch()}>
              Try again
            </Button>
          </div>
        )}
        {data && current && (
          <>
            <DividendIncomeByMonth calendar={data} />

            <section aria-label="Month">
              <div className="mb-3 flex items-center justify-between gap-3">
                <div>
                  <h3 className="text-base font-semibold">{monthTitle.format(monthDate(current))}</h3>
                  <p className="text-muted-foreground text-xs">
                    {monthEvents.length} {monthEvents.length === 1 ? 'payment' : 'payments'} ·{' '}
                    {formatGbp(sumAmounts(monthEvents))}
                  </p>
                </div>
                <div className="flex items-center gap-1">
                  <Button
                    variant="outline"
                    size="icon-sm"
                    aria-label="Previous month"
                    disabled={index <= 0}
                    onClick={() => go(months[index - 1])}
                  >
                    <ChevronLeft />
                  </Button>
                  <Button variant="outline" size="sm" onClick={() => go(monthKey(data.today))}>
                    Today
                  </Button>
                  <Button
                    variant="outline"
                    size="icon-sm"
                    aria-label="Next month"
                    disabled={index < 0 || index >= months.length - 1}
                    onClick={() => go(months[index + 1])}
                  >
                    <ChevronRight />
                  </Button>
                </div>
              </div>

              <div className="grid grid-cols-7 gap-px overflow-hidden rounded-lg border bg-[var(--color-border)]">
                {WEEKDAYS.map((d) => (
                  <div key={d} className="bg-muted text-muted-foreground px-1.5 py-1 text-center text-xs font-medium">
                    {d}
                  </div>
                ))}
                {monthGrid(current).map((date, i) => {
                  if (!date) return <div key={`pad-${i}`} className="bg-card min-h-20" />
                  const events = byDay.get(date) ?? []
                  const isToday = date === data.today
                  const selected = selectedDay === date
                  return (
                    <button
                      key={date}
                      type="button"
                      disabled={events.length === 0}
                      aria-pressed={events.length > 0 ? selected : undefined}
                      aria-label={`${dayLabel.format(new Date(date))}: ${events.length} ${
                        events.length === 1 ? 'payment' : 'payments'
                      }`}
                      onClick={() => setSelectedDay(selected ? null : date)}
                      className={cn(
                        'bg-card flex min-h-20 flex-col items-stretch gap-0.5 p-1.5 text-left align-top outline-none',
                        'focus-visible:ring-ring focus-visible:ring-2 focus-visible:ring-inset',
                        events.length > 0 && 'hover:bg-accent cursor-pointer',
                        selected && 'bg-accent',
                      )}
                    >
                      <span
                        className={cn(
                          'mb-0.5 flex size-5 items-center justify-center rounded-full text-xs tabular-nums',
                          isToday ? 'bg-primary text-primary-foreground font-semibold' : 'text-muted-foreground',
                        )}
                      >
                        {Number(date.slice(8))}
                      </span>
                      {/* Ticker and amount on wider screens; markers only on phones. */}
                      <span className="flex flex-wrap gap-1 sm:hidden">
                        {events.map((e, j) => (
                          <DividendStatusMarker key={j} status={e.status} />
                        ))}
                      </span>
                      <span className="hidden flex-col gap-0.5 sm:flex">
                        {events.slice(0, MAX_IN_CELL).map((e, j) => (
                          <span key={j} className="flex min-w-0 items-center gap-1 text-[11px] leading-tight">
                            <DividendStatusMarker status={e.status} />
                            <span className="truncate font-medium">{e.ticker}</span>
                          </span>
                        ))}
                        {events.length > MAX_IN_CELL && (
                          <span className="text-muted-foreground text-[11px]">
                            +{events.length - MAX_IN_CELL} more
                          </span>
                        )}
                      </span>
                    </button>
                  )
                })}
              </div>
              <ul className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs" aria-label="Legend">
                {STATUS_ORDER.map((s) => (
                  <li key={s} className="flex items-center gap-1.5">
                    <DividendStatusMarker status={s} />
                    {STATUS_META[s].label}
                  </li>
                ))}
              </ul>
            </section>

            <section aria-label="Payments" aria-live="polite">
              <div className="mb-2 flex items-center justify-between">
                <h3 className="text-sm font-medium">
                  {selectedDay ? dayLabel.format(new Date(selectedDay)) : `All of ${monthTitle.format(monthDate(current))}`}
                </h3>
                {selectedDay && (
                  <Button variant="ghost" size="sm" onClick={() => setSelectedDay(null)}>
                    Show whole month
                  </Button>
                )}
              </div>
              {listed.length === 0 ? (
                <p className="text-muted-foreground text-sm">No dividends this month.</p>
              ) : (
                <ul className="divide-y text-sm">
                  {listed.map((e, i) => (
                    <li key={`${e.date}-${e.instrumentId}-${i}`} className="flex items-center gap-3 py-2">
                      {/* On phones the date moves under the holding to leave room. */}
                      <span className="text-muted-foreground hidden w-24 shrink-0 tabular-nums sm:block">
                        {dayLabel.format(new Date(e.date))}
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className="text-muted-foreground block text-xs tabular-nums sm:hidden">
                          {dayLabel.format(new Date(e.date))}
                        </span>
                        <span className="block truncate sm:inline">
                          <span className="font-medium">{e.ticker}</span>
                          <span className="text-muted-foreground"> · {e.name}</span>
                        </span>
                        {e.exDate && (
                          <span className="text-muted-foreground block text-xs">
                            {placedOnExDate(e)
                              ? 'Payment date not announced; shown on the ex-dividend date'
                              : `Ex-dividend ${dayLabel.format(new Date(e.exDate))}`}
                          </span>
                        )}
                      </span>
                      <span className="flex shrink-0 items-center gap-1.5 text-xs">
                        <DividendStatusMarker status={e.status} />
                        <span className="text-muted-foreground hidden sm:inline">{STATUS_META[e.status].label}</span>
                      </span>
                      <span className="max-w-24 shrink-0 text-right font-medium sm:w-32 sm:max-w-none">
                        <Amount event={e} />
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </section>
          </>
        )}
      </CardContent>
    </Card>
  )
}
