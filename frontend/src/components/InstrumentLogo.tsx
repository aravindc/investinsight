import { useState } from 'react'

// Logos come from nvstly/icons (community-maintained, ticker-keyed, no
// auth), served via jsdelivr's CDN mirror for better caching than
// raw.githubusercontent. That repo is NYSE/NASDAQ-centric, so LSE-only
// tickers routinely 404 — the initials badge is the expected fallback for
// those, not an error state.
function logoUrl(ticker: string, source: 'jsdelivr' | 'raw'): string {
  const base =
    source === 'jsdelivr'
      ? 'https://cdn.jsdelivr.net/gh/nvstly/icons@main/ticker_icons'
      : 'https://raw.githubusercontent.com/nvstly/icons/main/ticker_icons'
  return `${base}/${ticker}.png`
}

export function InstrumentLogo({ ticker }: { ticker: string }) {
  const [stage, setStage] = useState<'jsdelivr' | 'raw' | 'fallback'>('jsdelivr')
  const upper = ticker.toUpperCase()

  if (stage === 'fallback') {
    return (
      <span className="bg-primary/10 text-primary flex size-9 shrink-0 items-center justify-center rounded-md text-xs font-semibold">
        {upper.slice(0, 2)}
      </span>
    )
  }

  return (
    <img
      src={logoUrl(upper, stage)}
      alt={upper}
      className="size-9 shrink-0 rounded-md object-contain"
      onError={() => setStage(stage === 'jsdelivr' ? 'raw' : 'fallback')}
    />
  )
}
