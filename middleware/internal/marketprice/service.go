package marketprice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const liveWindowDays = 2 // "t-1 and t-0": always live-fetch the last 2 calendar days

type Service struct {
	repo   *Repository
	yahoo  *YahooClient
	logger *zap.Logger
}

func NewService(repo *Repository, yahoo *YahooClient, logger *zap.Logger) *Service {
	return &Service{repo: repo, yahoo: yahoo, logger: logger}
}

// BackfillResult summarizes the outcome of a backfill run.
type BackfillResult struct {
	Processed int
	Failed    []string // tickers that could not be fetched/stored
}

// BackfillAll fetches and stores full price history (from first BUY to
// today) for every instrument that doesn't have it yet, and tops up
// instruments that are missing only recent days.
func (s *Service) BackfillAll(ctx context.Context) (*BackfillResult, error) {
	instruments, err := s.repo.InstrumentsNeedingPrices(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing instruments: %w", err)
	}

	now := time.Now().UTC()
	result := &BackfillResult{}
	for _, item := range instruments {
		from := item.FirstBuyDate.UTC().Truncate(24 * time.Hour)

		latest, err := s.repo.LatestPriceDate(ctx, item.ID)
		if err != nil {
			s.logger.Error("failed to read latest price date", zap.String("instrument", item.Ticker), zap.Error(err))
			result.Failed = append(result.Failed, item.Ticker)
			continue
		}
		if !latest.IsZero() {
			from = latest.AddDate(0, 0, 1)
		}
		if !from.Before(now) {
			continue // already up to date
		}

		if err := s.fetchAndStore(ctx, item.Instrument, from, now); err != nil {
			s.logger.Error("failed to backfill instrument", zap.String("instrument", item.Ticker), zap.Error(err))
			result.Failed = append(result.Failed, item.Ticker)
			continue
		}
		result.Processed++
	}

	return result, nil
}

func (s *Service) fetchAndStore(ctx context.Context, inst Instrument, from, to time.Time) error {
	closes, _, err := s.fetchDailyClosesWithFallback(ctx, inst, from, to)
	if err != nil {
		return err
	}

	for i := range closes {
		closes[i].Currency = normalizeCurrency(closes[i].Currency)
	}

	return s.repo.UpsertPrices(ctx, inst.ID, closes)
}

// fetchDailyClosesWithFallback tries YahooSymbol's derived symbol first; if
// that fails and the instrument isn't already suffixed for the LSE
// (GBP/GBX instruments already get ".L"), it retries with ".L" appended.
// This covers USD-denominated instruments that are nonetheless LSE-listed
// (e.g. JPMorgan's UCITS ETFs like JEQP/JEGP/JEIP, quoted in USD but only
// resolvable on Yahoo as "<ticker>.L") without needing a manual per-ticker
// exchange mapping.
func (s *Service) fetchDailyClosesWithFallback(ctx context.Context, inst Instrument, from, to time.Time) ([]DailyClose, string, error) {
	symbol := YahooSymbol(inst.Ticker, inst.BaseCurrency)

	closes, err := s.yahoo.FetchDailyCloses(ctx, symbol, from, to)
	if err == nil && len(closes) > 0 {
		return closes, symbol, nil
	}

	if strings.HasSuffix(symbol, ".L") {
		if err != nil {
			return nil, symbol, fmt.Errorf("fetching %s: %w", symbol, err)
		}
		return closes, symbol, nil
	}

	lseSymbol := symbol + ".L"
	lseCloses, lseErr := s.yahoo.FetchDailyCloses(ctx, lseSymbol, from, to)
	if lseErr == nil && len(lseCloses) > 0 {
		return lseCloses, lseSymbol, nil
	}

	if err != nil {
		return nil, symbol, fmt.Errorf("fetching %s: %w", symbol, err)
	}
	return closes, symbol, nil
}

// ClosePriceGBP returns the closing price for instrument on date, converted
// to GBP (including FX conversion for non-GBP instruments — see
// ConvertToGBP). Dates older than the live window (t-2 and earlier) are
// served from the stored table; the most recent liveWindowDays (t-1, t-0)
// are always fetched live from Yahoo Finance so today's/yesterday's value
// is never stale. If no exact-date price exists (non-trading day), the
// most recent prior close is used.
func (s *Service) ClosePriceGBP(ctx context.Context, inst Instrument, date time.Time) (decimal.Decimal, error) {
	date = date.UTC().Truncate(24 * time.Hour)
	now := time.Now().UTC().Truncate(24 * time.Hour)
	liveCutoff := now.AddDate(0, 0, -(liveWindowDays - 1))

	if !date.Before(liveCutoff) {
		closes, symbol, err := s.fetchDailyClosesWithFallback(ctx, inst, date.AddDate(0, 0, -7), now.AddDate(0, 0, 1))
		if err != nil {
			return decimal.Zero, fmt.Errorf("fetching live price for %s: %w", symbol, err)
		}
		if len(closes) == 0 {
			return decimal.Zero, fmt.Errorf("no live price data for %s", symbol)
		}

		latest := closes[len(closes)-1]
		currency := normalizeCurrency(latest.Currency)
		return s.ConvertToGBP(ctx, decimal.NewFromFloat(latest.Close), currency, date)
	}

	stored, err := s.repo.PriceOnOrBefore(ctx, inst.ID, date)
	if err != nil {
		return decimal.Zero, err
	}
	if stored == nil {
		return decimal.Zero, fmt.Errorf("no stored price for instrument %s on or before %s", inst.ID, date.Format("2006-01-02"))
	}
	return s.ConvertToGBP(ctx, stored.Close, stored.Currency, date)
}

// normalizeCurrency preserves the pence-vs-pounds distinction Yahoo
// reports, rather than collapsing it: LSE-listed equities are quoted in
// pence as "GBp" (lowercase p — normalized here to "GBX", matching the
// broker CSV convention for the same thing), while some LSE-listed
// instruments (notably accumulating ETFs like VUAG/VUSA/VWRP) are quoted
// by Yahoo in whole pounds as "GBP" (capital P) and must NOT be treated as
// pence. Case-sensitive exact match on the original Yahoo string is
// required — do not uppercase before this check.
func normalizeCurrency(currency string) string {
	switch currency {
	case "GBp", "GBX":
		return "GBX"
	case "GBP":
		return "GBP"
	default:
		return strings.ToUpper(currency)
	}
}

// ConvertToGBP converts a price in the given currency into GBP as of date.
// GBX prices are pence and are divided by 100; GBP prices are already
// whole pounds and pass through unchanged. Any other currency (USD, EUR,
// ...) is converted using the latest FX rate on or before date, fetched
// live from Yahoo Finance.
func (s *Service) ConvertToGBP(ctx context.Context, price decimal.Decimal, currency string, date time.Time) (decimal.Decimal, error) {
	switch currency {
	case "GBX":
		return price.Div(decimal.NewFromInt(100)), nil
	case "GBP":
		return price, nil
	}

	rate, err := s.fxRateToGBP(ctx, currency, date)
	if err != nil {
		return decimal.Zero, err
	}
	return price.Mul(decimal.NewFromFloat(rate)), nil
}

// PricesByDate exposes the repository's historical price lookup for
// reconstructing a value series over time.
func (s *Service) PricesByDate(ctx context.Context, instrumentID uuid.UUID, from, to time.Time) (map[time.Time]StoredPrice, error) {
	return s.repo.PricesByDate(ctx, instrumentID, from, to)
}

// EarliestPriceDate exposes the repository's earliest-price-date lookup.
func (s *Service) EarliestPriceDate(ctx context.Context, instrumentIDs []uuid.UUID) (time.Time, error) {
	return s.repo.EarliestPriceDate(ctx, instrumentIDs)
}
