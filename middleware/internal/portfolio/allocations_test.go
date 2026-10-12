package portfolio

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func strPtr(s string) *string { return &s }

func holding(sector string, subSector *string, category, subCategory, invested, market string) pricedHolding {
	h := pricedHolding{CurrentHolding: CurrentHolding{
		InstrumentID: uuid.New(),
		Sector:       sector,
		SubSector:    subSector,
		Category:     category,
		SubCategory:  subCategory,
		InvestedGBP:  dec(invested),
	}}
	if market != "" {
		h.MarketValueGBP = dec(market)
		h.Priced = true
	}
	return h
}

func TestParseGroupBy(t *testing.T) {
	for _, ok := range []string{"sector", "sub-sector", "category", "sub-category"} {
		if _, err := ParseGroupBy(ok); err != nil {
			t.Errorf("ParseGroupBy(%q) rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "Sector", "sub_sector", "ticker"} {
		if _, err := ParseGroupBy(bad); err == nil {
			t.Errorf("ParseGroupBy(%q) accepted", bad)
		}
	}
}

func TestAggregateAllocationsBySector(t *testing.T) {
	tech1 := holding("Technology", strPtr("Software"), "Equity", "Stock", "100", "150")
	tech2 := holding("Technology", nil, "Equity", "ETF", "200", "250")
	fin := holding("Financials", strPtr("Banks"), "Equity", "Stock", "100", "100")
	dividends := map[uuid.UUID]decimal.Decimal{tech1.InstrumentID: dec("8"), fin.InstrumentID: dec("5")}

	got := aggregateAllocations(GroupBySector, []pricedHolding{fin, tech1, tech2}, dividends)

	if len(got.Groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(got.Groups))
	}
	tech := got.Groups[0]
	if tech.Name != "Technology" || tech.Parent != nil {
		t.Fatalf("first group = %q (parent %v), want Technology at top level", tech.Name, tech.Parent)
	}
	if tech.InstrumentCount != 2 || !tech.InvestedGBP.Equal(dec("300")) || !tech.MarketValueGBP.Equal(dec("400")) {
		t.Errorf("Technology figures = %+v", tech.AllocationFigures)
	}
	if !tech.GainGBP.Equal(dec("100")) || !tech.GainPercent.Equal(dec("33.33")) {
		t.Errorf("Technology gain = %v (%v%%), want 100 (33.33%%)", tech.GainGBP, tech.GainPercent)
	}
	if !tech.SharePercent.Equal(dec("80")) || !tech.YieldPercent.Equal(dec("2")) {
		t.Errorf("Technology share = %v%%, yield = %v%%; want 80%%, 2%%", tech.SharePercent, tech.YieldPercent)
	}

	if !got.Totals.MarketValueGBP.Equal(dec("500")) || !got.Totals.TrailingYearDividendsGBP.Equal(dec("13")) {
		t.Errorf("totals = %+v", got.Totals)
	}
}

func TestAggregateAllocationsSubLevels(t *testing.T) {
	holdings := []pricedHolding{
		holding("Technology", strPtr("Software"), "Equity", "Stock", "100", "100"),
		holding("Technology", nil, "Equity", "ETF", "100", "100"),
		holding("Financials", nil, "Other", "Other", "100", "100"),
	}

	subSectors := aggregateAllocations(GroupBySubSector, holdings, nil)
	if len(subSectors.Groups) != 3 {
		t.Fatalf("got %d sub-sector groups, want 3 (unclassified kept per sector)", len(subSectors.Groups))
	}
	unclassified := 0
	for _, g := range subSectors.Groups {
		if g.Parent == nil {
			t.Errorf("sub-sector %q has no parent", g.Name)
		}
		if g.Unclassified {
			unclassified++
			if g.Name != UnclassifiedSubSector {
				t.Errorf("unclassified group named %q", g.Name)
			}
		}
	}
	if unclassified != 2 {
		t.Errorf("got %d unclassified groups, want 2", unclassified)
	}

	subCategories := aggregateAllocations(GroupBySubCategory, holdings, nil)
	if len(subCategories.Groups) != 3 {
		t.Fatalf("got %d sub-category groups, want 3", len(subCategories.Groups))
	}
	for _, g := range subCategories.Groups {
		if g.Parent == nil || g.Unclassified {
			t.Errorf("sub-category %q: parent %v, unclassified %v", g.Name, g.Parent, g.Unclassified)
		}
	}
}

func TestAggregateAllocationsUnpriced(t *testing.T) {
	priced := holding("Energy", nil, "Equity", "Stock", "100", "120")
	unpriced := holding("Energy", nil, "Equity", "Stock", "50", "")

	got := aggregateAllocations(GroupByCategory, []pricedHolding{priced, unpriced}, nil)

	g := got.Groups[0]
	if g.UnpricedCount != 1 || !g.MarketValueGBP.Equal(dec("120")) {
		t.Errorf("figures = %+v, want 1 unpriced and market value 120", g.AllocationFigures)
	}
	if g.GainGBP != nil || g.GainPercent != nil || g.YieldPercent != nil {
		t.Errorf("gain/yield reported for a partly priced group: %+v", g.AllocationFigures)
	}
	if got.Totals.GainGBP != nil {
		t.Errorf("total gain reported with an unpriced holding")
	}
}

func TestAggregateAllocationsEmpty(t *testing.T) {
	got := aggregateAllocations(GroupBySector, nil, nil)
	if got.Groups == nil || len(got.Groups) != 0 {
		t.Errorf("groups = %v, want an empty non-nil slice", got.Groups)
	}
}
