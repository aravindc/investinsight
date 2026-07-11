package marketprice

import "strings"

// YahooSymbol derives the Yahoo Finance chart symbol for an instrument from
// its broker ticker and base currency. LSE-listed instruments are quoted in
// GBX/GBP by this broker and need a ".L" suffix; EUR instruments in this
// portfolio trade on Deutsche Börse/XETRA and need a ".DE" suffix;
// everything else (assumed USD/US-listed) uses the ticker as-is. This is a
// heuristic, not an explicit per-instrument mapping, so it can still be
// wrong for tickers on other exchanges.
func YahooSymbol(ticker, baseCurrency string) string {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	currency := strings.ToUpper(strings.TrimSpace(baseCurrency))

	switch currency {
	case "GBP", "GBX":
		return ticker + ".L"
	case "EUR":
		return ticker + ".DE"
	default:
		return ticker
	}
}
