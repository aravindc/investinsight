export type Rect = { x: number; y: number; width: number; height: number }

type Sized<T> = { value: number; item: T }
type Placed<T> = Rect & { item: T }

// The worse aspect ratio in a row of areas laid along a side of this length.
function worstRatio(areas: number[], side: number): number {
  const sum = areas.reduce((a, b) => a + b, 0)
  const max = Math.max(...areas)
  const min = Math.min(...areas)
  return Math.max((side * side * max) / (sum * sum), (sum * sum) / (side * side * min))
}

/**
 * Squarified treemap layout (Bruls, Huizing & van Wijk): splits rect into
 * one rectangle per item, with area proportional to value and aspect ratios
 * kept close to 1. Items with no positive value are left out.
 */
export function squarify<T>(items: Sized<T>[], rect: Rect): Placed<T>[] {
  const positive = items.filter((i) => i.value > 0).sort((a, b) => b.value - a.value)
  const total = positive.reduce((sum, i) => sum + i.value, 0)
  if (total <= 0 || rect.width <= 0 || rect.height <= 0) return []

  const scale = (rect.width * rect.height) / total
  const nodes = positive.map((i) => ({ area: i.value * scale, item: i.item }))
  const placed: Placed<T>[] = []
  let { x, y, width, height } = rect

  const layoutRow = (row: typeof nodes) => {
    const sum = row.reduce((s, n) => s + n.area, 0)
    if (width >= height) {
      // A column along the left edge.
      const colWidth = sum / height
      let cy = y
      for (const n of row) {
        const h = n.area / colWidth
        placed.push({ x, y: cy, width: colWidth, height: h, item: n.item })
        cy += h
      }
      x += colWidth
      width -= colWidth
    } else {
      // A row along the top edge.
      const rowHeight = sum / width
      let cx = x
      for (const n of row) {
        const w = n.area / rowHeight
        placed.push({ x: cx, y, width: w, height: rowHeight, item: n.item })
        cx += w
      }
      y += rowHeight
      height -= rowHeight
    }
  }

  let row: typeof nodes = []
  for (const node of nodes) {
    const side = Math.min(width, height)
    const candidate = [...row, node]
    if (row.length === 0 || worstRatio(candidate.map((n) => n.area), side) <= worstRatio(row.map((n) => n.area), side)) {
      row = candidate
    } else {
      layoutRow(row)
      row = [node]
    }
  }
  if (row.length > 0) layoutRow(row)
  return placed
}

export function inset(rect: Rect, by: number): Rect {
  return {
    x: rect.x + by,
    y: rect.y + by,
    width: Math.max(rect.width - by * 2, 0),
    height: Math.max(rect.height - by * 2, 0),
  }
}
