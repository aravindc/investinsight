package trading212

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"investinsight/middleware/internal/portfolio"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestBuildReconciliation(t *testing.T) {
	t212 := &AccountSummary{ID: 42, Currency: "gbp", TotalValue: d("10250")}
	t212.Investments.CurrentValue = d("10000")
	t212.Investments.TotalCost = d("8000")
	t212.Investments.UnrealizedProfitLoss = d("2000")
	t212.Cash.AvailableToTrade = d("250")

	app := &portfolio.HoldingsSummary{
		MarketValueGBP:   d("10050"), // +0.5%: within tolerance
		InvestedValueGBP: d("7800"),  // −2.5%: mismatch
		UnpricedHoldings: 1,
	}

	r := buildReconciliation(uuid.New(), t212, app, time.Now())
	if r.Currency != "GBP" || r.UnpricedHoldings != 1 || !r.Trading212Info.CashAvailable.Equal(d("250")) {
		t.Fatalf("header fields wrong: %+v", r)
	}

	want := map[string]struct {
		diff, pct string
		matches   bool
	}{
		"marketValue":  {"50", "0.5", true},
		"costBasis":    {"-200", "-2.5", false},
		"unrealizedPL": {"250", "12.5", false}, // app 2250 vs 2000
	}
	for _, l := range r.Lines {
		w := want[l.Key]
		if !l.Difference.Equal(d(w.diff)) || l.DifferencePercent == nil || !l.DifferencePercent.Equal(d(w.pct)) || l.Matches != w.matches {
			t.Errorf("%s: diff=%s pct=%v matches=%v; want %s / %s%% / %v", l.Key, l.Difference, l.DifferencePercent, l.Matches, w.diff, w.pct, w.matches)
		}
	}
}

func TestCompareLine_ZeroTrading212Figure(t *testing.T) {
	if l := compareLine("x", "x", decimal.Zero, decimal.Zero); !l.Matches || l.DifferencePercent != nil {
		t.Errorf("0 vs 0: %+v", l)
	}
	if l := compareLine("x", "x", decimal.Zero, d("5")); l.Matches {
		t.Errorf("0 vs 5 should not match: %+v", l)
	}
}

func TestClient_AccountSummaryDecodesAmounts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id": 42, "currency": "GBP", "totalValue": 1234.56,
			"cash":        map[string]any{"availableToTrade": 10.5, "inPies": 0, "reservedForOrders": 1.25},
			"investments": map[string]any{"currentValue": 1224.06, "totalCost": 1000, "realizedProfitLoss": -3.2, "unrealizedProfitLoss": 224.06},
		})
	}))
	defer srv.Close()
	client, _ := NewClient(srv.URL)

	s, err := client.AccountSummary(context.Background(), Credentials{"k", "s"})
	if err != nil {
		t.Fatal(err)
	}
	if !s.Investments.CurrentValue.Equal(d("1224.06")) || !s.Cash.ReservedForOrders.Equal(d("1.25")) || !s.Investments.RealizedProfitLoss.Equal(d("-3.2")) {
		t.Fatalf("decoded wrongly: %+v", s)
	}
}
