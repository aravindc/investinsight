import { areValuesHidden } from '@/lib/hide-values'

const gbpFormatter = new Intl.NumberFormat('en-GB', {
  style: 'currency',
  currency: 'GBP',
})

export const MASKED_VALUE = '*****'

export function formatGbp(value: string | number) {
  if (areValuesHidden()) return `£${MASKED_VALUE}`
  return gbpFormatter.format(Number(value))
}

/** Masks a non-currency figure (e.g. share counts) when values are hidden. */
export function maskValue(value: string | number) {
  return areValuesHidden() ? MASKED_VALUE : value
}

const compactGbpFormatter = new Intl.NumberFormat('en-GB', {
  style: 'currency',
  currency: 'GBP',
  notation: 'compact',
  maximumFractionDigits: 1,
})

/** Short GBP for chart axes (£950, £1.2K); masked when values are hidden. */
export function formatGbpCompact(value: string | number) {
  if (areValuesHidden()) return `£${MASKED_VALUE}`
  return compactGbpFormatter.format(Number(value))
}

const wholeGbpFormatter = new Intl.NumberFormat('en-GB', {
  style: 'currency',
  currency: 'GBP',
  maximumFractionDigits: 0,
})

/** GBP to the nearest pound (£1,235), for tight breakdown lines. */
export function formatGbpWhole(value: string | number) {
  if (areValuesHidden()) return `£${MASKED_VALUE}`
  return wholeGbpFormatter.format(Number(value))
}
