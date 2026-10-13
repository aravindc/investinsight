# Story: Stunning visualizations for Analytics

Add analysis of the portfolio by **Sector**, **Sub-sector**, **Category** and
**Sub-category** to the Analytics page.

Status key: `[ ]` todo · `[~]` in progress · `[x]` done. Update this file at
the end of each session, and add a dated line to the log at the bottom.

## Context (as of 2026-10-10)

- Classification lives on `instruments`: `sector` → `sub_sector` (optional,
  NULL when unset; migration 00014) and `category` → `sub_category` (required,
  defaults to `Other`/`Other`; migration 00012).
- Only sector has an allocation endpoint: `GET /portfolio/sector-allocations`
  (`middleware/internal/portfolio/summary.go` `SectorAllocations`), returning
  invested, market value, gain and gain % per sector. It is used by the
  Holdings page donut (`frontend/src/components/SectorAllocationSection.tsx`).
- Analytics page (`frontend/src/pages/AnalyticsPage.tsx`) has four KPI cards
  plus `AssetCategoryFilter` (`lib/analytics-filter.tsx`), which narrows to
  selected sectors or instruments via `resolvedInstrumentIds`.
- Charts use Recharts 3. Sector colours and icons come from `lib/sector.ts`
  (`colorForSector`, `MAX_SECTOR_SLICES`).
- Every money figure must go through `formatGbp`/`maskValue` so the hide-values
  toggle masks it, including chart axes and tooltips.
- Load the `dataviz` skill before writing chart code.

## Phase 0: Foundations

- [x] **T1. Generic allocation endpoint.** Add `GET /portfolio/allocations?groupBy=sector|sub_sector|category|sub_category`
  with the same scope and optional instrument filter as `holdings-summary`. Per
  group, return the parent group (for sub-levels), instrument count, invested,
  market value, gain, gain %, share of portfolio %, and trailing-12-month
  dividends. Report a NULL sub-sector as "Unclassified". Add Go tests.
  Either keep `sector-allocations` as is, or move it onto the new endpoint.
- [x] **T2. Frontend data layer.** Add types and a `fetchAllocations(groupBy)`
  function in `lib/api.ts`, plus a `useAllocations(groupBy)` hook that respects
  the account scope and the Analytics filter.
- [x] **T3. Colour system for all four dimensions.** Keep stable colours for
  sectors and categories, and give each sub-sector or sub-category a shade of
  its parent's colour. Validate the palette in light and dark mode using the
  dataviz skill.
- [x] **T4. Analytics layout and navigation.** Below the KPI cards, add a
  section with a Sector · Sub-sector · Category · Sub-category switcher, plus a
  shared "measure" toggle (market value / invested / gain / dividends). Persist
  the chosen tab in the URL query string.

### Phase 0 outcome (2026-10-12)

- **Endpoint:** `GET /portfolio/allocations?scope=…&groupBy=sector|sub-sector|category|sub-category[&instrumentId=…]`
  (`middleware/internal/portfolio/allocations.go`). It returns
  `{groupBy, totals, groups[]}`. Each group has `name`, `parent` (null at the
  top level), `unclassified`, `sharePercent` (of market value) and the shared
  figures: instrument and unpriced counts, invested, market value, gain,
  gain %, trailing-12-month dividends and yield %. Gain and yield are null
  unless every holding in the group is priced. Groups are sorted by market
  value. Unit tests are in `allocations_test.go`. `sector-allocations` is
  unchanged and still feeds the Holdings page.
- **Frontend data:** `fetchAllocations` in `lib/api.ts`. `useAllocations(groupBy)`,
  `useGroupColor(groupBy)`, `allocationKey` and `DIMENSIONS` are in
  `lib/allocations.ts`.
- **Colours:** sectors use `colorForSector`. Categories take palette hues in
  their configured order, and the `Other` category is neutral grey.
  Sub-groups are shades of the parent's colour (100/72/50/34%), chosen by
  position in the parent's configured list. Unclassified is 20%.
- **UI:** `components/AllocationBreakdown.tsx` is on the Analytics page below
  the KPI cards. It has dimension tabs and a measure select (market value,
  invested, gain, dividends 12m), both kept in the URL (`?dimension=&measure=`).
  It shows a ranked list with share bars. Phases 1–4 add their charts beside
  or above this list, reusing the same hooks.
- **Known gap for T5:** the portfolio has 12 sectors but there are only 8
  palette hues, so `colorForSector` repeats colours. Fold the smallest
  sectors into "Other" in charts (as `SectorAllocationSection` does with
  `MAX_SECTOR_SLICES`), or label them directly.

## Phase 1: Sector analysis

- [x] **T5. Sector allocation chart.** Show a donut or treemap of the selected
  measure per sector, with a centre total and a legend showing % share.
- [x] **T6. Sector performance chart.** Show a diverging bar chart of gain %
  (and gain £) per sector, sorted, with green/red against a zero baseline.
- [x] **T7. Sector breakdown table.** Show sortable rows with share, invested,
  value, gain, yield and instrument count. Clicking a row drills into its
  sub-sectors (T8).

### Phase 1 outcome (2026-10-12)

- **Palette fix:** the `--chart-1..8` order in `index.css` failed the dataviz
  validator (orange next to magenta in light mode, magenta next to red in
  dark mode). It now uses the validated order (blue, orange, aqua, yellow,
  magenta, green, violet, red), with the same hues and violet lightened to
  `#6250d6`. This recolours sectors on the Holdings page, and the
  invested-value chart's second line is now orange. The dividends chart was
  switched to `chart-7` so it stays violet.
- **Single hue for groups:** there are more sectors (12) than palette hues
  (8), so the Phase 1 charts draw every group in `chart-1` and let labels
  carry identity, rather than reusing a hue. `useGroupColor` is still unused
  and is kept for the T8/T12 hierarchy views, where parents are few.
- **Components:** `AllocationAnalysis` (the control row, a breadcrumb when
  drilled in, and the layout) replaces `AllocationBreakdown`. It also adds:
  - `AllocationTreemap` (T5): sized by the measure, or by market value when
    the measure is Gain. Labels show only where they fit; tooltips work on
    hover and keyboard focus.
  - `AllocationPerformanceChart` (T6): return on cost, diverging from zero
    with signed tip labels. Groups with unpriced holdings are listed
    underneath.
  - `AllocationTable` (T7): sortable, starts sorted by the measure, has a
    totals row, and is the table version of both charts.
- **Drill-down:** clicking a sector or category (tile, row or chevron) sets
  `?dimension=sub-sector&parent=<sector>` (or the sub-category equivalent).
  The breadcrumb goes back. This covers the basic part of T8 and T12, but
  not the sunburst.
- `SortableHead` is now generic over its sort key.

## Phase 2: Sub-sector analysis

- [x] **T8. Sector → sub-sector hierarchy chart.** Show a sunburst (or nested
  donut) and a treemap with sub-sectors grouped inside their sector. Clicking
  zooms into one sector, with a breadcrumb to go back.
- [x] **T9. Sub-sector performance and table.** Reuse T6/T7 with a group
  column for the parent sector.
- [x] **T10. Unclassified callout.** Show how much value has no sub-sector,
  linking to the Admin classification card so it can be fixed.

### Phase 2 outcome (2026-10-12)

- **T8:** `AllocationHierarchy` shows sub-groups nested inside labelled
  parent blocks when a sub-level tab is open and nothing is drilled into.
  Clicking a parent header or any tile in it zooms in (sets `parent`), and
  hovering a tile dims the other parents. It uses a custom squarified
  layout (`lib/treemap-layout.ts`), because Recharts' Treemap can't add
  parent gaps or header strips. The layout was checked for exact areas,
  no overlaps and staying in bounds. The sunburst was dropped: the dataviz
  skill caps donuts at about 6 segments, and there are 36 sub-sectors.
- **T9:** the table gets a sortable Sector or Category column in the nested
  view. The return chart adds the parent to repeated names (e.g. "ETF
  (Equity)"). Above 14 rows it shows the best and worst 7, with a
  Show all button. An outlier more than 2.5× the next bar (e.g. +3,329%)
  is clipped at the axis end with a break mark, and its label keeps the
  true value. The label margin is sized to the longest label.
- **T10:** `UnclassifiedCallout` appears on the Sub-sector tab when any
  holding has no sub-sector, linking to `/admin?tab=categories`. Live data
  currently has none, so it hasn't been seen rendered.
- **T12:** covered by the same generic components on the Sub-category tab
  (nested view, drill-down and breadcrumb).
- Checked with Playwright screenshots (`npx -y playwright@latest screenshot
  --browser chromium --color-scheme dark …`). Values are hidden by
  default, so screenshots don't expose amounts.

## Phase 3: Category analysis

- [x] **T11. Category allocation and performance.** Reuse the T5–T7 components
  for categories (e.g. Equity vs Other). Flag value still in the `Other`
  catch-all.

### Phase 3 outcome (2026-10-12)

- `needsClassifying(group, dimension)` in `lib/allocations.ts` flags the
  protected `Other` sector and category, sub-categories under `Other`, and
  holdings with no sub-sector. The `Other` sector is treated like the
  category, since new holdings land in it too.
- `ClassificationCallout` (which replaces `UnclassifiedCallout`) shows on
  any tab with flagged holdings: count, share of market value, and a link
  to `/admin?tab=categories`. Treemap and nested-treemap tiles for flagged
  groups are drawn in neutral grey with dark labels.
- The treemap names groups under 3% share below the chart ("Smaller: …"),
  since a dominant group (Equity at 98%) leaves the rest as unlabelled
  slivers.
- Live data has nothing in `Other`, so this was checked with a Playwright
  script that renames one category in the intercepted response
  (`page.route`), in light and dark mode.

## Phase 4: Sub-category analysis

- [x] **T12. Category → sub-category hierarchy.** Reuse T8/T9 for categories
  (e.g. Equity → Stock vs ETF), including the drill-down and breadcrumb.

## Phase 5: Cross-cutting insights (stretch)

- [x] **T13. Cross-dimension heatmap.** Show a sector × category matrix
  (e.g. how much Technology is held through ETFs vs stocks), shaded by value or
  gain %. This needs a two-level `groupBy` on the endpoint.
- [x] **T14. Concentration insights.** Show the top holdings' share within each
  group, plus a simple diversification score, with warnings when one group
  exceeds a threshold.
- [x] **T15. Allocation over time.** Show a 100% stacked area chart of each
  group's share by month. This needs a new backend series grouped by dimension,
  so it's the largest item and should be done last.

### Phase 5 outcome (2026-10-12)

- **Backend:**
  - `walkDailyValues` was extracted from `MarketValueSeries` and sums daily
    values into buckets. A fingerprint of the series history was identical
    before and after.
  - `GET /portfolio/allocation-matrix?rows=&cols=` cross-tabulates any two
    different dimensions. It returns non-empty cells only, with rows and
    columns sorted by value.
  - `GET /portfolio/allocation-series?groupBy=` returns month-end values per
    group plus today, for holdings held today, classified as today.
  - Every allocation group and the totals report `largestHolding` (share of
    the group) and `effectiveHoldings` (1 / Σ weight²).
  - All of these honour the Analytics filter, and tests are in
    `allocation_insights_test.go`.
- **T13 – `AllocationHeatmap`:** the current dimension × the other family
  (sector-family views use sub-category columns, so you see how each sector
  is held; category-family views use sector columns).
  - Toggles between % of row and % of portfolio, and has row and column
    totals.
  - Uses one sequential hue, mixed from the card surface, shaded relative to
    the largest cell, with a scale legend.
  - The hovered or focused cell's details appear in a line below (no
    tooltip).
- **T14 – `ConcentrationCard`:**
  - Shows effective holdings, the largest holding (warning over 10%) and the
    largest group (warning over 30%, sector tabs only, since most of a
    portfolio in equities is normal).
  - Lists groups with 50% or more in one holding.
  - Warnings use an amber icon and a label.
- **T15 – `AllocationOverTime`:** a 100% stacked area chart with straight
  segments between month ends.
  - The 7 largest groups by current value take palette slots in that order;
    the rest fold into a grey "N other …" band.
  - The legend always shows, with the latest shares. The tooltip lists every
    band.
  - Colours are by current rank, not fixed per entity (there are more
    sectors than hues). A filter that changes the ranking can recolour
    bands. Revisit if that's confusing.
- **Layout:** treemap and return chart → full-width heatmap → Concentration
  next to the mix over time → table.
- **Known gap (for T16):** when drilled into one sector, the Concentration
  headline figures (effective holdings, largest holding) still describe the
  whole selection, not that sector.

## Phase 6: Polish

- [x] **T16. States and responsiveness.** Cover loading skeletons, empty and
  error states, mobile layout, keyboard and screen-reader access for
  drill-downs, and dark mode.
- [x] **T17. Verify end to end.** Check that totals match the KPI cards, that
  the Analytics filter and account scope apply, and that the hide-values
  toggle masks all axes and tooltips. Run Go tests, `tsc` and lint, and check
  in the browser.

### Phase 6 outcome (2026-10-12)

- **T16 – mobile:** at 390px the page overflowed by 676px, all from the nav
  bar. The nav now wraps, the links get their own scrolling row below `lg`
  (the current page's link is scrolled into view), the Analytics KPI cards
  are 2-up below `lg`, and page padding is 16px on phones. No horizontal
  overflow remains.
- **T16 – states:** the first load shows chart-sized pulse placeholders (later
  loads keep the previous charts, dimmed). A load error offers Try again.
- **T16 – keyboard and screen readers:**
  - Treemap tiles show a 2px ring on keyboard focus, and nested-treemap
    parent headers underline.
  - Drilling in moves focus to the breadcrumb; drilling out moves it to the
    Allocation heading.
- **T16 – concentration when drilled in:** the card now describes that sector
  ("Concentration in Financials"), using the parent's own figures from the
  parent dimension's allocations. Group shares are re-based on the parent.
- **T17 – API checks (19/19 pass):** for the household and two single
  accounts, every dimension's totals and the matrix cells match
  `holdings-summary`. The filter narrows allocations, matrix and series, and
  a filter matching nothing returns nothing.
  - Prices are live, so compare against a summary fetched just before and
    after each call. A single earlier summary drifts as prices refresh.
- **T17 – browser checks (15/15 pass):**
  - With values hidden, no amount appears in the page text, the chart SVG
    text, or the treemap, return-chart, heatmap or mix-over-time details.
  - The focus ring shows; Enter drills in and focus lands on the breadcrumb;
    going back focuses the heading.
  - After Show values (on Holdings), amounts appear. A reload hides them
    again, with no page errors.
- **T17 – static checks:** Go vet and tests, `tsc`, the production build and
  lint pass. All 6 lint warnings are in files that predate this story.
- The check scripts lived in the session scratchpad and weren't committed.
  They used the Playwright package installed there, with `page.route` for
  mocking and `waitForLoadState('networkidle')`. Values were checked by
  pattern only, never printed.

## Session log

- 2026-10-10: Task list created. Nothing started.
- 2026-10-12: Phase 0 (T1–T4) done on branch `feature/analytics-foundation`.
  Totals for all four dimensions were checked against `holdings-summary`
  on live data. Not yet viewed in a browser.
- 2026-10-12: Phase 1 (T5–T7) built on `feature/analytics-sector`. tsc and
  lint are clean. No headless browser was available, so the charts haven't
  been checked visually yet.
- 2026-10-12: Phase 2 (T8–T10) built on `feature/analytics-sub-sector`, and
  T12 is covered by the same components. Checked visually with Playwright
  in light and dark mode.
- 2026-10-12: Phase 3 (T11) built on `feature/analytics-category`. Phases 0–4
  are now done; Phase 5 (stretch) and Phase 6 (polish) remain.
- 2026-10-12: Phase 5 (T13–T15) built on `feature/analytics-insights`.
  Endpoints were checked against live data, and screenshots were taken in
  light and dark mode on the sector, category and drilled-in views.
- 2026-10-12: Phase 6 (T16–T17) on `feature/analytics-polish`. All 17 tasks
  are done.
- 2026-10-12: The four KPI cards (investment value, total profit, IRR,
  passive income) were removed from the Analytics page at the user's
  request, along with their Analytics-only variants. The Holdings page
  versions are unchanged. The backend still accepts `instrumentId` on
  `holdings-summary`, `irr` and `passive-income`, but nothing sends it now.
- 2026-10-12: At the user's request, the Analytics filters were removed
  entirely, so the page always covers the whole account or household. A
  draft two-filter version (sector or asset, and category or sub-category,
  with And/Or) was discarded before it was committed. Removed on the
  frontend: `AssetCategoryFilter`, `lib/analytics-filter.tsx`,
  `fetchAssetCategories`, and the `instrumentIds` argument on the API
  helpers. The backend still accepts `instrumentId` on the allocation,
  summary, IRR and passive-income endpoints, and still serves
  `/portfolio/asset-categories`, but nothing calls these now.
