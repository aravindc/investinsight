package instrument

import (
	"strings"
	"testing"
)

func TestParseClassificationCSV(t *testing.T) {
	csv := "\uFEFFticker, Name ,Category,Sub Category,Sector,Notes\n" +
		"AAPL,Apple,Equity,Stock,Technology,extra column ignored\n" +
		"\n" +
		"BA,Boeing,  Equity ,Stock,  Aerospace   &  Defence\n" +
		",,Equity,Stock,Technology\n" +
		"MSFT,Microsoft,,Stock,Technology\n" +
		"AAPL,apple,Equity,ETF,Technology\n" +
		"OIL,Oil co,Commodity,Oil/Gas,Energy\n"

	rows, issues, err := ParseClassificationCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2: %+v", len(rows), rows)
	}
	if rows[1].Ticker != "BA" || rows[1].Category != "Equity" || rows[1].Sector != "Aerospace & Defence" || rows[1].Line != 4 {
		t.Errorf("BA row normalized wrongly: %+v", rows[1])
	}

	want := map[int]string{5: "ticker and name are both empty", 6: "category", 7: "duplicate of line 2", 8: "sub-category"}
	if len(issues) != len(want) {
		t.Fatalf("got issues %+v", issues)
	}
	for _, is := range issues {
		if !strings.Contains(is.Reason, want[is.Line]) {
			t.Errorf("line %d: reason %q, want it to mention %q", is.Line, is.Reason, want[is.Line])
		}
	}
}

func TestParseClassificationCSV_BadHeader(t *testing.T) {
	_, _, err := ParseClassificationCSV(strings.NewReader("Ticker,Name,Sector\nAAPL,Apple,Technology\n"))
	if err == nil || !strings.Contains(err.Error(), "category") || !strings.Contains(err.Error(), "sub-category") {
		t.Fatalf("err = %v; want missing category and sub-category", err)
	}
	if _, _, err := ParseClassificationCSV(strings.NewReader("")); err == nil {
		t.Fatal("empty file accepted")
	}
}

func TestParseClassificationCSV_SubSector(t *testing.T) {
	rows, issues, err := ParseClassificationCSV(strings.NewReader(
		"Ticker,Name,Category,Sub-Category,Sector,Sub Sector\n" +
			"BARC,Barclays,Equity,Stock,Financials,  Banks \n" +
			"AV,Aviva,Equity,Stock,Financials,\n" +
			"X,X,Equity,Stock,Financials,Bad/Name\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || !rows[0].HasSubSector || rows[0].SubSector != "Banks" || !rows[1].HasSubSector || rows[1].SubSector != "" {
		t.Fatalf("rows = %+v", rows)
	}
	if len(issues) != 1 || !strings.Contains(issues[0].Reason, "sub-sector") {
		t.Fatalf("issues = %+v", issues)
	}

	// Without the column, sub-sectors are left alone.
	rows, _, _ = ParseClassificationCSV(strings.NewReader("Ticker,Name,Category,Sub-Category,Sector\nBARC,Barclays,Equity,Stock,Financials\n"))
	if len(rows) != 1 || rows[0].HasSubSector {
		t.Fatalf("rows = %+v", rows)
	}
}
