package marketprice

import "testing"

func TestYahooSymbol(t *testing.T) {
	tests := []struct {
		name, ticker, currency, isin, want string
	}{
		{"LSE pence", "ULVR", "GBX", "GB00B10RZP78", "ULVR.L"},
		{"LSE pounds", "VUSA", "GBP", "IE00B3XXRP09", "VUSA.L"},
		{"US listed", "AAPL", "USD", "US0378331005", "AAPL"},
		{"French issuer on Euronext Paris", "HO", "EUR", "FR0000121329", "HO.PA"},
		{"French issuer, lowercase input", " am ", "eur", "fr0014004l86", "AM.PA"},
		{"German issuer on XETRA", "BMW", "EUR", "DE0005190003", "BMW.DE"},
		{"Dutch issuer stays on XETRA", "AIR", "EUR", "NL0000235190", "AIR.DE"},
		{"EUR with missing ISIN", "XYZ", "EUR", "", "XYZ.DE"},
		{"ISIN override: Samsung GDR on LSE", "SMSN", "USD", "US7960508882", "SMSN.L"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := YahooSymbol(tt.ticker, tt.currency, tt.isin); got != tt.want {
				t.Errorf("YahooSymbol(%q, %q, %q) = %q, want %q", tt.ticker, tt.currency, tt.isin, got, tt.want)
			}
		})
	}
}
