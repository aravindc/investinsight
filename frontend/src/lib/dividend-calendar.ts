import type { DividendEvent, DividendStatus } from '@/lib/api'

// One hue, stepping lighter as a dividend becomes less certain: paid, then
// announced, then estimated. Shape (see DividendStatusMarker) carries the
// status too, so it never rests on colour alone.
export const STATUS_META: Record<DividendStatus, { label: string; fill: string }> = {
  paid: { label: 'Paid', fill: 'var(--color-chart-1)' },
  announced: {
    label: 'Announced',
    fill: 'color-mix(in oklab, var(--color-chart-1) 65%, var(--color-card))',
  },
  estimated: {
    label: 'Estimated',
    fill: 'color-mix(in oklab, var(--color-chart-1) 40%, var(--color-card))',
  },
}

export const STATUS_ORDER: DividendStatus[] = ['paid', 'announced', 'estimated']

/** "2026-11" from "2026-11-13". */
export function monthKey(date: string): string {
  return date.slice(0, 7)
}

/** Every month from from's to to's, inclusive, as "YYYY-MM". */
export function monthsBetween(from: string, to: string): string[] {
  const months: string[] = []
  let [y, m] = from.slice(0, 7).split('-').map(Number)
  const [ty, tm] = to.slice(0, 7).split('-').map(Number)
  while (y < ty || (y === ty && m <= tm)) {
    months.push(`${y}-${String(m).padStart(2, '0')}`)
    m += 1
    if (m > 12) {
      m = 1
      y += 1
    }
  }
  return months
}

export function monthDate(key: string): Date {
  const [y, m] = key.split('-').map(Number)
  return new Date(Date.UTC(y, m - 1, 1))
}

export function sumAmounts(events: DividendEvent[]): number {
  return events.reduce((sum, e) => sum + Number(e.amountGbp ?? 0), 0)
}

/**
 * An announcement with no payment date or estimate is placed on its
 * ex-dividend date; the UI says so.
 */
export function placedOnExDate(e: DividendEvent): boolean {
  return e.status === 'announced' && e.exDate === e.date && e.amountGbp === null
}
