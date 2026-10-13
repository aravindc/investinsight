package portfolio

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"investinsight/middleware/internal/marketprice"
	"investinsight/middleware/internal/xirr"
)

// pricingConcurrency bounds how many instruments are priced at once in
// pricedHoldings. Each price lookup for "today" is a live Yahoo Finance HTTP
// call (see marketprice.Service.ClosePriceGBP), so pricing holdings one at a
// time serializes dozens of network round-trips; a worker pool keeps this
// bounded rather than firing unlimited concurrent requests at Yahoo.
const pricingConcurrency = 16

// PriceLookup resolves current and historical instrument prices in GBP,
// and announced dividend dates, implemented by *marketprice.Service.
type PriceLookup interface {
	DividendDates(ctx context.Context, inst marketprice.Instrument) (*marketprice.DividendDates, error)
	ClosePriceGBP(ctx context.Context, inst marketprice.Instrument, date time.Time) (decimal.Decimal, error)
	PricesByDate(ctx context.Context, instrumentID uuid.UUID, from, to time.Time) (map[time.Time]marketprice.StoredPrice, error)
	EarliestPriceDate(ctx context.Context, instrumentIDs []uuid.UUID) (time.Time, error)
	FXRateSeries(ctx context.Context, currency string, from, to time.Time) (map[time.Time]float64, error)
}

// SummaryService combines the ledger repository with live/stored market
// prices to compute holdings market value.
type SummaryService struct {
	repo   *Repository
	prices PriceLookup
	logger *zap.Logger
}

func NewSummaryService(repo *Repository, prices PriceLookup, logger *zap.Logger) *SummaryService {
	return &SummaryService{repo: repo, prices: prices, logger: logger}
}

// pricedHolding is one current holding with its live market value, if a
// price could be resolved for it.
type pricedHolding struct {
	CurrentHolding
	MarketValueGBP decimal.Decimal
	Priced         bool
}

// pricedHoldings prices every current holding once, for reuse across
// HoldingsSummary and SectorAllocations. If a price lookup fails for an
// instrument (e.g. unmapped symbol, Yahoo error), it's returned with
// Priced=false rather than failing the whole request, and the failure is
// logged.
func (s *SummaryService) pricedHoldings(ctx context.Context, scope Scope, filter Filter) ([]pricedHolding, error) {
	holdings, err := s.repo.CurrentHoldings(ctx, scope, filter)
	if err != nil {
		return nil, err
	}

	today := time.Now().UTC()
	priced := make([]pricedHolding, len(holdings))

	sem := make(chan struct{}, pricingConcurrency)
	var wg sync.WaitGroup
	for i, h := range holdings {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, h CurrentHolding) {
			defer wg.Done()
			defer func() { <-sem }()

			price, err := s.prices.ClosePriceGBP(ctx, marketprice.Instrument{
				ID:           h.InstrumentID,
				Ticker:       h.Ticker,
				BaseCurrency: h.BaseCurrency,
				ISIN:         h.ISIN,
			}, today)
			if err != nil {
				s.logger.Warn("failed to price holding, excluding from market value",
					zap.String("ticker", h.Ticker), zap.Error(err))
				priced[i] = pricedHolding{CurrentHolding: h}
				return
			}

			priced[i] = pricedHolding{
				CurrentHolding: h,
				MarketValueGBP: h.NetShares.Mul(price).Round(2),
				Priced:         true,
			}
		}(i, h)
	}
	wg.Wait()

	return priced, nil
}

// HoldingsSummary computes invested (cost basis) value and current market
// value for all instruments still held. Holdings that couldn't be priced
// are excluded from the market value total but still counted in invested
// value.
func (s *SummaryService) HoldingsSummary(ctx context.Context, scope Scope, filter Filter) (*HoldingsSummary, error) {
	holdings, err := s.pricedHoldings(ctx, scope, filter)
	if err != nil {
		return nil, err
	}

	invested := decimal.Zero
	marketValue := decimal.Zero
	unpriced := 0
	for _, h := range holdings {
		invested = invested.Add(h.InvestedGBP)
		if h.Priced {
			marketValue = marketValue.Add(h.MarketValueGBP)
		} else {
			unpriced++
		}
	}

	return &HoldingsSummary{
		InvestedValueGBP: invested.Round(2),
		MarketValueGBP:   marketValue.Round(2),
		UnpricedHoldings: unpriced,
	}, nil
}

// SectorAllocations groups current holdings by sector: invested (cost
// basis) value, current market value, and the unrealized gain/loss
// between them. GainGBP/GainPercent are zero for a sector if any of its
// instruments couldn't be priced (a partial market value would understate
// the sector's true value and produce a misleading loss). Sorted
// descending by invested value.
func (s *SummaryService) SectorAllocations(ctx context.Context, scope Scope) ([]SectorAllocation, error) {
	holdings, err := s.pricedHoldings(ctx, scope, NoFilter)
	if err != nil {
		return nil, err
	}

	type sectorTotals struct {
		instruments int
		invested    decimal.Decimal
		marketValue decimal.Decimal
		fullyPriced bool
	}
	byS := make(map[string]*sectorTotals)
	var order []string

	for _, h := range holdings {
		name := h.Sector
		t, ok := byS[name]
		if !ok {
			t = &sectorTotals{fullyPriced: true}
			byS[name] = t
			order = append(order, name)
		}
		t.instruments++
		t.invested = t.invested.Add(h.InvestedGBP)
		if h.Priced {
			t.marketValue = t.marketValue.Add(h.MarketValueGBP)
		} else {
			t.fullyPriced = false
		}
	}

	allocations := make([]SectorAllocation, 0, len(byS))
	for _, name := range order {
		t := byS[name]
		a := SectorAllocation{
			Sector:          name,
			InstrumentCount: t.instruments,
			InvestedGBP:     t.invested.Round(2),
		}
		if t.fullyPriced {
			a.MarketValueGBP = t.marketValue.Round(2)
			a.GainGBP = t.marketValue.Sub(t.invested).Round(2)
			if t.invested.IsPositive() {
				a.GainPercent = a.GainGBP.Div(t.invested).Mul(decimal.NewFromInt(100)).Round(2)
			}
		}
		allocations = append(allocations, a)
	}

	sort.Slice(allocations, func(i, j int) bool {
		return allocations[i].InvestedGBP.GreaterThan(allocations[j].InvestedGBP)
	})

	return allocations, nil
}

// AssetCategories groups every currently-held instrument by sector, for
// populating the Analytics page's category/asset multi-select. Unlike
// SectorAllocations, this needs no pricing — it's metadata only — so it
// reads CurrentHoldings directly instead of going through pricedHoldings.
func (s *SummaryService) AssetCategories(ctx context.Context, scope Scope) ([]AssetCategory, error) {
	holdings, err := s.repo.CurrentHoldings(ctx, scope, NoFilter)
	if err != nil {
		return nil, err
	}

	byS := make(map[string]*AssetCategory)
	var order []string
	for _, h := range holdings {
		name := h.Sector
		c, ok := byS[name]
		if !ok {
			c = &AssetCategory{Sector: name}
			byS[name] = c
			order = append(order, name)
		}
		c.Instruments = append(c.Instruments, AssetOption{
			InstrumentID: h.InstrumentID,
			Ticker:       h.Ticker,
			Name:         h.Name,
		})
	}

	result := make([]AssetCategory, 0, len(byS))
	for _, name := range order {
		result = append(result, *byS[name])
	}
	return result, nil
}

// HoldingsTable computes one HoldingRow per currently-held instrument, with
// cost basis, current value, dividends, dividend yield, 5-year dividend
// growth, total profit, per-instrument IRR, and share of portfolio.
func (s *SummaryService) HoldingsTable(ctx context.Context, scope Scope) ([]HoldingRow, error) {
	holdings, err := s.pricedHoldings(ctx, scope, NoFilter)
	if err != nil {
		return nil, err
	}

	dividends, err := s.repo.InstrumentDividends(ctx, scope)
	if err != nil {
		return nil, err
	}
	trailingDividends, err := s.repo.InstrumentTrailingYearDividends(ctx, scope)
	if err != nil {
		return nil, err
	}
	yearlyDividends, err := s.repo.InstrumentYearlyDividends(ctx, scope)
	if err != nil {
		return nil, err
	}
	cashFlows, err := s.repo.InstrumentCashFlows(ctx, scope)
	if err != nil {
		return nil, err
	}

	dividendGrowth := computeDividendGrowth(yearlyDividends)
	flowsByInstrument := make(map[uuid.UUID][]CashFlow)
	for _, f := range cashFlows {
		flowsByInstrument[f.InstrumentID] = append(flowsByInstrument[f.InstrumentID], CashFlow{Date: f.Date, AmountGBP: f.AmountGBP})
	}

	totalMarketValue := decimal.Zero
	for _, h := range holdings {
		if h.Priced {
			totalMarketValue = totalMarketValue.Add(h.MarketValueGBP)
		}
	}

	rows := make([]HoldingRow, 0, len(holdings))
	for _, h := range holdings {
		row := HoldingRow{
			InstrumentID:         h.InstrumentID,
			Ticker:               h.Ticker,
			Name:                 h.Name,
			Sector:               h.Sector,
			Shares:               h.NetShares,
			CostBasisGBP:         h.InvestedGBP.Round(2),
			Priced:               h.Priced,
			DividendsReceivedGBP: dividends[h.InstrumentID].Round(2),
		}
		if h.NetShares.IsPositive() {
			row.CostBasisPerShareGBP = h.InvestedGBP.Div(h.NetShares).Round(4)
			row.DividendsReceivedPerShareGBP = dividends[h.InstrumentID].Div(h.NetShares).Round(4)
		}
		if growth, ok := dividendGrowth[h.InstrumentID]; ok {
			row.DividendGrowthPercent = &growth
		}

		if h.Priced {
			row.CurrentValueGBP = h.MarketValueGBP
			if h.NetShares.IsPositive() {
				row.CurrentValuePerShareGBP = h.MarketValueGBP.Div(h.NetShares).Round(4)
			}
			if trailing, ok := trailingDividends[h.InstrumentID]; ok && h.MarketValueGBP.IsPositive() {
				yield := trailing.Div(h.MarketValueGBP).Mul(decimal.NewFromInt(100)).Round(2)
				row.DividendYieldPercent = &yield
			}

			profit := h.MarketValueGBP.Sub(h.InvestedGBP).Round(2)
			row.TotalProfitGBP = &profit
			if h.InvestedGBP.IsPositive() {
				profitPercent := profit.Div(h.InvestedGBP).Mul(decimal.NewFromInt(100)).Round(2)
				row.TotalProfitPercent = &profitPercent
			}

			if totalMarketValue.IsPositive() {
				share := h.MarketValueGBP.Div(totalMarketValue).Mul(decimal.NewFromInt(100)).Round(2)
				row.PortfolioSharePercent = &share
			}

			if rate, err := computeIRR(flowsByInstrument[h.InstrumentID], h.MarketValueGBP); err == nil {
				row.IRRPercent = decimalPtr(decimal.NewFromFloat(rate).Mul(decimal.NewFromInt(100)).Round(2))
			}
		}

		rows = append(rows, row)
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].CurrentValueGBP.GreaterThan(rows[j].CurrentValueGBP)
	})

	return rows, nil
}

// computeDividendGrowth compares total dividend income between the earliest
// and latest calendar years (capped at the last 5 years) within each
// instrument's own dividend history. Instruments with fewer than 2 distinct
// years of dividend history are omitted from the result.
func computeDividendGrowth(yearly []InstrumentYearlyDividend) map[uuid.UUID]decimal.Decimal {
	byInstrument := make(map[uuid.UUID][]InstrumentYearlyDividend)
	for _, y := range yearly {
		byInstrument[y.InstrumentID] = append(byInstrument[y.InstrumentID], y)
	}

	result := make(map[uuid.UUID]decimal.Decimal)
	for id, years := range byInstrument {
		if len(years) < 2 {
			continue
		}
		sort.Slice(years, func(i, j int) bool { return years[i].Year < years[j].Year })

		latest := years[len(years)-1]
		windowStart := latest.Year - 4 // up to 5 years inclusive
		earliest := years[0]
		for _, y := range years {
			if y.Year >= windowStart {
				earliest = y
				break
			}
		}
		if earliest.Year == latest.Year || !earliest.TotalGBP.IsPositive() {
			continue
		}

		growth := latest.TotalGBP.Sub(earliest.TotalGBP).Div(earliest.TotalGBP).Mul(decimal.NewFromInt(100)).Round(2)
		result[id] = growth
	}
	return result
}

func decimalPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}

// PassiveIncome reports a trailing-12-month income yield: dividend and
// interest income received in the last 365 days, as a percentage of
// current market value. Zero if there's no current market value to divide
// by.
func (s *SummaryService) PassiveIncome(ctx context.Context, scope Scope, filter Filter) (*PassiveIncome, error) {
	trailingYear, priorYear, err := s.repo.TrailingYearIncome(ctx, scope, filter)
	if err != nil {
		return nil, err
	}

	holdings, err := s.HoldingsSummary(ctx, scope, filter)
	if err != nil {
		return nil, err
	}

	result := &PassiveIncome{TrailingYearGBP: trailingYear.Round(2)}
	if holdings.MarketValueGBP.IsPositive() {
		result.YieldPercent = trailingYear.
			Div(holdings.MarketValueGBP).
			Mul(decimal.NewFromInt(100)).
			Round(2)
	}
	if priorYear.IsPositive() {
		change := trailingYear.
			Sub(priorYear).
			Div(priorYear).
			Mul(decimal.NewFromInt(100)).
			Round(2)
		result.ChangePercent = &change
	}
	return result, nil
}

// PortfolioIRR computes the money-weighted rate of return (XIRR) on
// invested holdings — every BUY/SELL/DIVIDEND/INTEREST transaction as a
// dated cash flow, plus today's current market value as a final terminal
// inflow (as if everything were liquidated today); deposits/withdrawals
// are excluded, see Repository.InvestmentCashFlows — alongside the same
// calculation restricted to instruments still held today (excluding
// fully-exited positions and INTEREST, which has no instrument to
// restrict by; see Repository.CurrentHoldingsCashFlows).
func (s *SummaryService) PortfolioIRR(ctx context.Context, scope Scope, filter Filter) (*PortfolioIRR, error) {
	holdings, err := s.HoldingsSummary(ctx, scope, filter)
	if err != nil {
		return nil, err
	}

	allFlows, err := s.repo.InvestmentCashFlows(ctx, scope, filter)
	if err != nil {
		return nil, err
	}
	rate, err := computeIRR(allFlows, holdings.MarketValueGBP)
	if err != nil {
		return nil, err
	}

	result := &PortfolioIRR{IRR: rate}

	currentFlows, err := s.repo.CurrentHoldingsCashFlows(ctx, scope, filter)
	if err != nil {
		return nil, err
	}
	if currentRate, err := computeIRR(currentFlows, holdings.MarketValueGBP); err == nil {
		result.CurrentHoldingsIRR = &currentRate
	}

	return result, nil
}

// computeIRR converts dated GBP cash flows plus a terminal market value
// into an XIRR. Returns xirr's sentinel errors unchanged when no rate is
// defined (too few flows, no sign change, or the solver didn't converge).
func computeIRR(cashFlows []CashFlow, terminalValueGBP decimal.Decimal) (float64, error) {
	flows := make([]xirr.CashFlow, 0, len(cashFlows)+1)
	for _, f := range cashFlows {
		amount, _ := f.AmountGBP.Float64()
		flows = append(flows, xirr.CashFlow{Date: f.Date, Amount: amount})
	}
	if terminalValueGBP.IsPositive() {
		terminalValue, _ := terminalValueGBP.Float64()
		flows = append(flows, xirr.CashFlow{Date: time.Now().UTC(), Amount: terminalValue})
	}
	return xirr.Compute(flows)
}

// MarketValueSeries reconstructs historical portfolio value: for each day,
// (shares held that day × that instrument's closing price that day), summed
// across every currently-held instrument that has a price for that day.
// This is a best-effort per-day total, not a fully-covered one: the series
// starts as early as any held instrument has both shares and a price (in
// practice, close to InvestedValueSeries' start), and a given day's total
// can under-count if a held instrument's price history hadn't started yet
// by that day — that instrument is simply omitted from that day's sum, the
// same way a permanently unpriceable instrument (delisted/unmapped symbol)
// is omitted from every day's sum in HoldingsSummary.
func (s *SummaryService) MarketValueSeries(ctx context.Context, scope Scope) ([]ValuePoint, error) {
	series := make([]ValuePoint, 0)
	err := s.walkDailyValues(ctx, scope,
		func(uuid.UUID) (string, bool) { return "", true },
		func(day time.Time, totals map[string]decimal.Decimal) {
			series = append(series, ValuePoint{Date: day.Format("2006-01-02"), ValueGBP: totals[""].Round(2)})
		})
	if err != nil {
		return nil, err
	}
	return series, nil
}

// walkDailyValues reconstructs each day's value of currently-held
// instruments (see MarketValueSeries for the coverage caveats), summed
// into buckets: keyOf names an instrument's bucket, or excludes it by
// returning false. visit is called once per day, in date order, with that
// day's unrounded bucket totals; buckets with nothing held or priced that
// day are absent. The map is reused between calls.
func (s *SummaryService) walkDailyValues(
	ctx context.Context,
	scope Scope,
	keyOf func(uuid.UUID) (string, bool),
	visit func(day time.Time, totals map[string]decimal.Decimal),
) error {
	deltas, err := s.repo.ShareDeltasForCurrentHoldings(ctx, scope)
	if err != nil {
		return err
	}
	if len(deltas) == 0 {
		return nil
	}

	byInstrument := make(map[uuid.UUID][]ShareDelta)
	var instrumentIDs []uuid.UUID
	for _, d := range deltas {
		if _, ok := byInstrument[d.InstrumentID]; !ok {
			instrumentIDs = append(instrumentIDs, d.InstrumentID)
		}
		byInstrument[d.InstrumentID] = append(byInstrument[d.InstrumentID], d)
	}

	keys := make(map[uuid.UUID]string, len(instrumentIDs))
	included := instrumentIDs[:0:0]
	for _, id := range instrumentIDs {
		if key, ok := keyOf(id); ok {
			keys[id] = key
			included = append(included, id)
		}
	}
	instrumentIDs = included
	if len(instrumentIDs) == 0 {
		return nil
	}

	earliestPriceDate, err := s.prices.EarliestPriceDate(ctx, instrumentIDs)
	if err != nil {
		return err
	}
	if earliestPriceDate.IsZero() {
		return nil // no held instrument has any price data
	}
	// Normalize to the time.UTC location: dates scanned from Postgres come
	// back in a fixed +0000 zone that is NOT time.UTC, so time.Time values
	// derived from it (via AddDate, etc.) would otherwise never compare
	// equal — by == or as map keys — to dates normalized with .UTC()
	// elsewhere (e.g. in PricesByDate), even though .Equal() and String()
	// both report them as identical.
	coverageStart := earliestPriceDate.UTC()

	now := time.Now().UTC().Truncate(24 * time.Hour)

	// Per instrument: deltas sorted ascending (to walk a running-share
	// pointer forward day-by-day) and stored prices for the full range.
	type instrumentState struct {
		deltas       []ShareDelta // sorted ascending by date
		nextDeltaIdx int
		runningShare decimal.Decimal
		lastPrice    decimal.Decimal
		havePrice    bool
		prices       map[time.Time]marketprice.StoredPrice
	}
	states := make(map[uuid.UUID]*instrumentState, len(instrumentIDs))
	currencyByInstrument := make(map[uuid.UUID]string, len(instrumentIDs))

	for _, id := range instrumentIDs {
		instDeltas := byInstrument[id]
		sort.Slice(instDeltas, func(i, j int) bool { return instDeltas[i].Date.Before(instDeltas[j].Date) })

		prices, err := s.prices.PricesByDate(ctx, id, coverageStart, now)
		if err != nil {
			return err
		}

		states[id] = &instrumentState{deltas: instDeltas, prices: prices}
		if len(instDeltas) > 0 {
			// Currency is stored per price row (normalized), not per
			// instrument, so peek at the first available price.
			for _, p := range prices {
				currencyByInstrument[id] = p.Currency
				break
			}
		}
	}

	// Fetch FX rate history once per distinct non-GBP currency, instead of
	// a live call per instrument-day. GBX (pence) and GBP (pounds) both
	// need no FX lookup — see priceToGBP.
	fxSeriesByCurrency := make(map[string]map[time.Time]float64)
	for _, currency := range currencyByInstrument {
		if currency == "" || currency == "GBP" || currency == "GBX" {
			continue
		}
		if _, ok := fxSeriesByCurrency[currency]; ok {
			continue
		}
		rates, err := s.prices.FXRateSeries(ctx, currency, coverageStart, now)
		if err != nil {
			return err
		}
		fxSeriesByCurrency[currency] = rates
	}

	totals := make(map[string]decimal.Decimal)
	for day := coverageStart; !day.After(now); day = day.AddDate(0, 0, 1) {
		clear(totals)

		for _, id := range instrumentIDs {
			st := states[id]

			// Advance the running share total past any deltas up to today.
			for st.nextDeltaIdx < len(st.deltas) && !st.deltas[st.nextDeltaIdx].Date.After(day) {
				st.runningShare = st.runningShare.Add(st.deltas[st.nextDeltaIdx].SharesDelta)
				st.nextDeltaIdx++
			}
			if st.runningShare.IsZero() {
				continue
			}

			p, ok := st.prices[day]
			if !ok {
				if !st.havePrice {
					continue // instrument not yet priced on/before this day
				}
				totals[keys[id]] = totals[keys[id]].Add(st.runningShare.Mul(st.lastPrice))
				continue
			}

			gbpPrice, ok := priceToGBP(p, day, fxSeriesByCurrency)
			if !ok {
				continue // no FX rate available for this day
			}
			st.lastPrice = gbpPrice
			st.havePrice = true

			totals[keys[id]] = totals[keys[id]].Add(st.runningShare.Mul(st.lastPrice))
		}

		visit(day, totals)
	}

	return nil
}

// priceToGBP converts a stored price to GBP for a given day: pence-based
// division for GBX, passthrough for GBP (already whole pounds), or the
// day's FX rate for any other currency.
func priceToGBP(p marketprice.StoredPrice, day time.Time, fxSeriesByCurrency map[string]map[time.Time]float64) (decimal.Decimal, bool) {
	switch p.Currency {
	case "GBX":
		return p.Close.Div(decimal.NewFromInt(100)), true
	case "GBP":
		return p.Close, true
	}

	rate, ok := marketprice.FXRateOnOrBefore(fxSeriesByCurrency[p.Currency], day)
	if !ok {
		return decimal.Zero, false
	}
	return p.Close.Mul(decimal.NewFromFloat(rate)), true
}
