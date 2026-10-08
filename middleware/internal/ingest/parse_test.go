package ingest

import (
	"strings"
	"testing"
	"time"
)

// Trading 212's API export omits columns with no data in the period — here
// no sells (no Result), no stamp duty and no conversion fees.
func TestParseCSV_SparseExportHeader(t *testing.T) {
	csv := "Action,Time (UTC),ISIN,Ticker,Name,Notes,ID,No. of shares,Price / share,Currency (Price / share),Exchange rate,Total,Currency (Total),Withholding tax,Currency (Withholding tax)\n" +
		"Market buy,2026-10-01 09:30:00+00:00,GB00B10RZP78,ULVR,Unilever,,EOF1,2,45.10,GBP,1.00,90.20,GBP,,\n" +
		"Deposit,2026-10-02 08:00:00+00:00,,,,,DEP1,,,,,500.00,GBP,,\n"

	rows, errs := ParseCSV(strings.NewReader(csv), true)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	buy := rows[0]
	if buy.Action != "Market buy" || buy.ISIN != "GB00B10RZP78" || buy.Total.String() != "90.2" || buy.CurrencyTotal != "GBP" {
		t.Errorf("buy row parsed wrongly: %+v", buy)
	}
	if !buy.TransactionTime.Equal(time.Date(2026, 10, 1, 9, 30, 0, 0, time.UTC)) {
		t.Errorf("buy time = %v", buy.TransactionTime)
	}
	if !buy.Result.IsZero() || !buy.StampDuty.IsZero() || buy.CurrencyConversion != "" {
		t.Errorf("missing columns should read as empty/zero: %+v", buy)
	}
}

func TestParseCSV_MissingRequiredColumn(t *testing.T) {
	_, errs := ParseCSV(strings.NewReader("Action,ISIN\nDeposit,\n"), true)
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "Time") {
		t.Fatalf("want a missing Time column error, got %v", errs)
	}
}

func TestParseCSV_EmptyFile(t *testing.T) {
	for _, csv := range []string{"", "Action,Time (UTC),ID,Total,Currency (Total)\n"} {
		rows, errs := ParseCSV(strings.NewReader(csv), true)
		if len(rows) != 0 || len(errs) != 0 {
			t.Errorf("ParseCSV(%q) = %d rows, errs %v; want none", csv, len(rows), errs)
		}
	}
}
