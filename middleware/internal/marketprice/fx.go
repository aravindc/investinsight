package marketprice

import (
	"context"
	"fmt"
	"time"
)

// fxSymbol returns the Yahoo Finance symbol for converting 1 unit of
// currency into GBP, or "" if currency is already GBP (no conversion
// needed).
func fxSymbol(currency string) string {
	switch currency {
	case "GBP":
		return ""
	case "USD":
		return "GBP=X" // Yahoo quotes this pair directly as USD/GBP
	default:
		return currency + "GBP=X" // e.g. EURGBP=X
	}
}

// fxRateToGBP returns the exchange rate to multiply a `currency`-denominated
// price by to get GBP, using the latest available rate on or before date.
func (s *Service) fxRateToGBP(ctx context.Context, currency string, date time.Time) (float64, error) {
	symbol := fxSymbol(currency)
	if symbol == "" {
		return 1, nil
	}

	closes, err := s.yahoo.FetchDailyCloses(ctx, symbol, date.AddDate(0, 0, -7), date.AddDate(0, 0, 1))
	if err != nil {
		return 0, fmt.Errorf("fetching FX rate for %s: %w", symbol, err)
	}
	if len(closes) == 0 {
		return 0, fmt.Errorf("no FX rate data for %s", symbol)
	}

	return closes[len(closes)-1].Close, nil
}

// FXRateSeries fetches the full daily EUR/GBP-style exchange rate history
// for currency across [from, to], keyed by date. Intended for reconstructing
// a historical value series without one live FX call per day. Returns an
// empty map (no error) if currency is already GBP.
func (s *Service) FXRateSeries(ctx context.Context, currency string, from, to time.Time) (map[time.Time]float64, error) {
	symbol := fxSymbol(currency)
	if symbol == "" {
		return map[time.Time]float64{}, nil
	}

	closes, err := s.yahoo.FetchDailyCloses(ctx, symbol, from.AddDate(0, 0, -7), to)
	if err != nil {
		return nil, fmt.Errorf("fetching FX rate history for %s: %w", symbol, err)
	}

	rates := make(map[time.Time]float64, len(closes))
	for _, c := range closes {
		rates[c.Date] = c.Close
	}
	return rates, nil
}

// FXRateOnOrBefore returns the exchange rate for date from a rate series
// built by FXRateSeries, falling back to the most recent earlier date
// (weekends/holidays have no FX close either). ok is false if no rate is
// available on or before date within a reasonable lookback window.
func FXRateOnOrBefore(rates map[time.Time]float64, date time.Time) (rate float64, ok bool) {
	for d := date; !d.Before(date.AddDate(0, 0, -14)); d = d.AddDate(0, 0, -1) {
		if r, found := rates[d]; found {
			return r, true
		}
	}
	return 0, false
}
