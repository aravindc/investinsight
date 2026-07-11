const gbpFormatter = new Intl.NumberFormat('en-GB', {
  style: 'currency',
  currency: 'GBP',
})

export function formatGbp(value: string | number) {
  return gbpFormatter.format(Number(value))
}
