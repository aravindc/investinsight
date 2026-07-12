import { useState } from 'react'

// Logo sources, tried in order: our own backend (logo.dev images fetched
// and cached daily via POST /api/logos/backfill — see middleware/internal/logo),
// then nvstly/icons (community-maintained, ticker-keyed, no auth) via
// jsdelivr's CDN mirror, then raw.githubusercontent. nvstly/icons is
// NYSE/NASDAQ-centric, so LSE-only tickers routinely 404 there — the
// initials badge is the expected final fallback for those, not an error
// state.
type Stage = 'backend' | 'jsdelivr' | 'raw' | 'fallback'

function logoUrl(instrumentId: string, ticker: string, stage: Stage): string {
  if (stage === 'backend') return `/api/logos/${instrumentId}`
  const base =
    stage === 'jsdelivr'
      ? 'https://cdn.jsdelivr.net/gh/nvstly/icons@main/ticker_icons'
      : 'https://raw.githubusercontent.com/nvstly/icons/main/ticker_icons'
  return `${base}/${ticker}.png`
}

function nextStage(stage: Stage): Stage {
  if (stage === 'backend') return 'jsdelivr'
  if (stage === 'jsdelivr') return 'raw'
  return 'fallback'
}

export function InstrumentLogo({ instrumentId, ticker }: { instrumentId: string; ticker: string }) {
  const [stage, setStage] = useState<Stage>('backend')
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
      src={logoUrl(instrumentId, upper, stage)}
      alt={upper}
      className="size-9 shrink-0 rounded-md object-contain"
      onError={() => setStage(nextStage(stage))}
    />
  )
}
