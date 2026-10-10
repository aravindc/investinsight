import { useSyncExternalStore } from 'react'

// A module-level store rather than context, so plain formatters (table
// column definitions, chart tick formatters) can read it outside React.
// Values start hidden on every load; showing them lasts until a reload.
let hidden = true
const listeners = new Set<() => void>()

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function areValuesHidden() {
  return hidden
}

export function setValuesHidden(next: boolean) {
  hidden = next
  listeners.forEach((l) => l())
}

export function useValuesHidden() {
  return useSyncExternalStore(subscribe, areValuesHidden)
}
