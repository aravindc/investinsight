package marketprice

import "strings"

// YahooSymbol derives the Yahoo Finance chart symbol for an instrument from
// its broker ticker, base currency and ISIN. LSE-listed instruments are
// quoted in GBX/GBP by this broker and need a ".L" suffix. EUR instruments
// are resolved by ISIN country: French issuers (e.g. Thales "HO", Dassault
// Aviation "AM") trade on Euronext Paris and need ".PA"; the rest in this
// portfolio trade on Deutsche Börse/XETRA and need ".DE". Everything else
// (assumed USD/US-listed) uses the ticker as-is. This is a heuristic, not an
// explicit per-instrument mapping, so it can still be wrong for tickers on
// other exchanges — those are pinned in yahooSymbolOverridesByISIN.
func YahooSymbol(ticker, baseCurrency, isin string) string {
	if symbol, ok := yahooSymbolOverridesByISIN[strings.ToUpper(strings.TrimSpace(isin))]; ok {
		return symbol
	}

	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	currency := strings.ToUpper(strings.TrimSpace(baseCurrency))

	switch currency {
	case "GBP", "GBX":
		return ticker + ".L"
	case "EUR":
		return ticker + eurExchangeSuffix(isin)
	default:
		return ticker
	}
}

// yahooSymbolOverridesByISIN pins the Yahoo symbol for instruments the
// currency/ISIN heuristic gets wrong, keyed by ISIN.
var yahooSymbolOverridesByISIN = map[string]string{
	// Samsung Electronics GDR: USD-quoted but LSE-listed; plain "SMSN" has
	// no data on Yahoo.
	"US7960508882": "SMSN.L",
}

// eurExchangeSuffixByISINCountry maps an ISIN's country prefix to the Yahoo
// suffix of the exchange this portfolio's EUR instruments from that country
// are priced on. Countries not listed fall back to XETRA (".DE") — including
// NL, since Airbus (NL ISIN) resolves fine as "AIR.DE".
var eurExchangeSuffixByISINCountry = map[string]string{
	"FR": ".PA", // Euronext Paris
}

func eurExchangeSuffix(isin string) string {
	isin = strings.ToUpper(strings.TrimSpace(isin))
	if len(isin) >= 2 {
		if suffix, ok := eurExchangeSuffixByISINCountry[isin[:2]]; ok {
			return suffix
		}
	}
	return ".DE"
}
