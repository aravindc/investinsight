package portfolio

import (
	"testing"

	"github.com/shopspring/decimal"
)

func named(h pricedHolding, name string) pricedHolding {
	h.Name = name
	h.Ticker = name
	return h
}

func TestConcentration(t *testing.T) {
	big := named(holding("Technology", nil, "Equity", "Stock", "100", "300"), "BIG")
	small := named(holding("Technology", nil, "Equity", "Stock", "100", "100"), "SMALL")
	unpriced := named(holding("Technology", nil, "Equity", "Stock", "100", ""), "NOPRICE")

	got := aggregateAllocations(GroupBySector, []pricedHolding{small, big, unpriced}, nil).Groups[0]

	if got.LargestHolding == nil || got.LargestHolding.Ticker != "BIG" || !got.LargestHolding.SharePercent.Equal(dec("75")) {
		t.Errorf("largest holding = %+v, want BIG at 75%%", got.LargestHolding)
	}
	// Weights 0.75 and 0.25: 1 / (0.5625 + 0.0625) = 1.6.
	if got.EffectiveHoldings == nil || !got.EffectiveHoldings.Equal(dec("1.6")) {
		t.Errorf("effective holdings = %v, want 1.6", got.EffectiveHoldings)
	}

	equal := aggregateAllocations(GroupBySector, []pricedHolding{
		holding("Energy", nil, "Equity", "Stock", "1", "50"),
		holding("Energy", nil, "Equity", "Stock", "1", "50"),
		holding("Energy", nil, "Equity", "Stock", "1", "50"),
		holding("Energy", nil, "Equity", "Stock", "1", "50"),
	}, nil).Totals
	if !equal.EffectiveHoldings.Equal(dec("4")) {
		t.Errorf("four equal holdings: effective = %v, want 4", equal.EffectiveHoldings)
	}

	none := aggregateAllocations(GroupBySector, []pricedHolding{unpriced}, nil).Groups[0]
	if none.LargestHolding != nil || none.EffectiveHoldings != nil {
		t.Errorf("unpriced-only group reported concentration: %+v", none.AllocationFigures)
	}
}

func TestAggregateMatrix(t *testing.T) {
	holdings := []pricedHolding{
		holding("Technology", nil, "Equity", "Stock", "100", "300"),
		holding("Technology", nil, "Equity", "ETF", "100", "100"),
		holding("Financials", nil, "Equity", "Stock", "100", "50"),
		holding("Financials", nil, "Fixed Income", "ETF", "100", "50"),
	}

	m := aggregateMatrix(GroupBySector, GroupByCategory, holdings)

	if len(m.Rows) != 2 || m.Rows[0].Name != "Technology" || m.Rows[1].Name != "Financials" {
		t.Fatalf("rows = %+v, want Technology then Financials", m.Rows)
	}
	if len(m.Cols) != 2 || m.Cols[0].Name != "Equity" {
		t.Fatalf("cols = %+v, want Equity first", m.Cols)
	}
	if len(m.Cells) != 3 {
		t.Fatalf("got %d cells, want 3 (Technology has no Fixed Income)", len(m.Cells))
	}
	first := m.Cells[0]
	if first.Row != 0 || first.Col != 0 || first.InstrumentCount != 2 || !first.MarketValueGBP.Equal(dec("400")) {
		t.Errorf("Technology × Equity = %+v", first)
	}
	if !first.SharePercent.Equal(dec("80")) {
		t.Errorf("Technology × Equity share = %v, want 80", first.SharePercent)
	}
	sum := decimal.Zero
	for _, c := range m.Cells {
		sum = sum.Add(c.MarketValueGBP)
	}
	if !sum.Equal(m.Totals.MarketValueGBP) {
		t.Errorf("cells sum to %v, totals say %v", sum, m.Totals.MarketValueGBP)
	}
}

func TestBuildAllocationSeriesOrdersByLatestValue(t *testing.T) {
	keys := []groupKey{{name: "Small"}, {name: "Big"}}
	points := []AllocationSeriesPoint{
		{Date: "2026-08-31", ValuesGBP: []decimal.Decimal{dec("50"), dec("10")}},
		{Date: "2026-10-12", ValuesGBP: []decimal.Decimal{dec("20"), dec("90")}},
	}

	got := buildAllocationSeries(GroupBySector, keys, points)

	if got.Groups[0].Name != "Big" || got.Groups[1].Name != "Small" {
		t.Fatalf("groups = %+v, want Big then Small", got.Groups)
	}
	if !got.Points[0].ValuesGBP[0].Equal(dec("10")) || !got.Points[0].ValuesGBP[1].Equal(dec("50")) {
		t.Errorf("first point values = %v, want reordered to [10 50]", got.Points[0].ValuesGBP)
	}
}
