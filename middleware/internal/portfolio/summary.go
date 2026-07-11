package portfolio

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"investinsight/middleware/internal/marketprice"
	"investinsight/middleware/internal/xirr"
)

// PriceLookup resolves current and historical instrument prices in GBP,
// implemented by *marketprice.Service.
type PriceLookup interface {
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

// HoldingsSummary computes invested (cost basis) value and current market
// value for all instruments still held. If a price lookup fails for an
// instrument (e.g. unmapped symbol, Yahoo error), that instrument is
// skipped from the market value total rather than failing the whole
// request, and the failure is logged.
func (s *SummaryService) HoldingsSummary(ctx context.Context) (*HoldingsSummary, error) {
	holdings, err := s.repo.CurrentHoldings(ctx)
	if err != nil {
		return nil, err
	}

	today := time.Now().UTC()
	invested := decimal.Zero
	marketValue := decimal.Zero

	for _, h := range holdings {
		invested = invested.Add(h.InvestedGBP)

		price, err := s.prices.ClosePriceGBP(ctx, marketprice.Instrument{
			ID:           h.InstrumentID,
			Ticker:       h.Ticker,
			BaseCurrency: h.BaseCurrency,
		}, today)
		if err != nil {
			s.logger.Warn("failed to price holding, excluding from market value",
				zap.String("ticker", h.Ticker), zap.Error(err))
			continue
		}

		marketValue = marketValue.Add(h.NetShares.Mul(price))
	}

	return &HoldingsSummary{
		InvestedValueGBP: invested,
		MarketValueGBP:   marketValue.Round(2),
	}, nil
}

// PortfolioIRR computes the money-weighted rate of return (XIRR) on
// invested holdings: every BUY/SELL/DIVIDEND/INTEREST transaction as a
// dated cash flow, plus today's current market value as a final terminal
// inflow (as if everything were liquidated today). Deposits/withdrawals
// are excluded — see Repository.InvestmentCashFlows.
func (s *SummaryService) PortfolioIRR(ctx context.Context) (*PortfolioIRR, error) {
	cashFlows, err := s.repo.InvestmentCashFlows(ctx)
	if err != nil {
		return nil, err
	}

	holdings, err := s.HoldingsSummary(ctx)
	if err != nil {
		return nil, err
	}

	flows := make([]xirr.CashFlow, 0, len(cashFlows)+1)
	for _, f := range cashFlows {
		amount, _ := f.AmountGBP.Float64()
		flows = append(flows, xirr.CashFlow{Date: f.Date, Amount: amount})
	}
	if holdings.MarketValueGBP.IsPositive() {
		terminalValue, _ := holdings.MarketValueGBP.Float64()
		flows = append(flows, xirr.CashFlow{Date: time.Now().UTC(), Amount: terminalValue})
	}

	rate, err := xirr.Compute(flows)
	if err != nil {
		return nil, err
	}

	return &PortfolioIRR{IRR: rate}, nil
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
func (s *SummaryService) MarketValueSeries(ctx context.Context) ([]ValuePoint, error) {
	deltas, err := s.repo.ShareDeltasForCurrentHoldings(ctx)
	if err != nil {
		return nil, err
	}
	if len(deltas) == 0 {
		return []ValuePoint{}, nil
	}

	byInstrument := make(map[uuid.UUID][]ShareDelta)
	var instrumentIDs []uuid.UUID
	for _, d := range deltas {
		if _, ok := byInstrument[d.InstrumentID]; !ok {
			instrumentIDs = append(instrumentIDs, d.InstrumentID)
		}
		byInstrument[d.InstrumentID] = append(byInstrument[d.InstrumentID], d)
	}

	earliestPriceDate, err := s.prices.EarliestPriceDate(ctx, instrumentIDs)
	if err != nil {
		return nil, err
	}
	if earliestPriceDate.IsZero() {
		return []ValuePoint{}, nil // no held instrument has any price data
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
			return nil, err
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
	// a live call per instrument-day.
	fxSeriesByCurrency := make(map[string]map[time.Time]float64)
	for _, currency := range currencyByInstrument {
		if currency == "" || currency == "GBP" {
			continue
		}
		if _, ok := fxSeriesByCurrency[currency]; ok {
			continue
		}
		rates, err := s.prices.FXRateSeries(ctx, currency, coverageStart, now)
		if err != nil {
			return nil, err
		}
		fxSeriesByCurrency[currency] = rates
	}

	series := make([]ValuePoint, 0)
	for day := coverageStart; !day.After(now); day = day.AddDate(0, 0, 1) {
		total := decimal.Zero

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
				total = total.Add(st.runningShare.Mul(st.lastPrice))
				continue
			}

			gbpPrice, ok := priceToGBP(p, day, fxSeriesByCurrency)
			if !ok {
				continue // no FX rate available for this day
			}
			st.lastPrice = gbpPrice
			st.havePrice = true

			total = total.Add(st.runningShare.Mul(st.lastPrice))
		}

		series = append(series, ValuePoint{Date: day.Format("2006-01-02"), ValueGBP: total.Round(2)})
	}

	return series, nil
}

// priceToGBP converts a stored price to GBP for a given day: pence-based
// division for GBP, or the day's FX rate for any other currency.
func priceToGBP(p marketprice.StoredPrice, day time.Time, fxSeriesByCurrency map[string]map[time.Time]float64) (decimal.Decimal, bool) {
	if p.Currency == "GBP" {
		return p.Close.Div(decimal.NewFromInt(100)), true
	}

	rate, ok := marketprice.FXRateOnOrBefore(fxSeriesByCurrency[p.Currency], day)
	if !ok {
		return decimal.Zero, false
	}
	return p.Close.Mul(decimal.NewFromFloat(rate)), true
}
