package portfolio

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// GroupBy is the classification dimension allocations are grouped by.
type GroupBy string

const (
	GroupBySector      GroupBy = "sector"
	GroupBySubSector   GroupBy = "sub-sector"
	GroupByCategory    GroupBy = "category"
	GroupBySubCategory GroupBy = "sub-category"
)

// UnclassifiedSubSector names the group of holdings with no sub-sector.
const UnclassifiedSubSector = "Unclassified"

func ParseGroupBy(raw string) (GroupBy, error) {
	switch g := GroupBy(raw); g {
	case GroupBySector, GroupBySubSector, GroupByCategory, GroupBySubCategory:
		return g, nil
	}
	return "", fmt.Errorf("invalid groupBy %q: want sector, sub-sector, category or sub-category", raw)
}

// AllocationFigures are the values reported for one group and for the
// whole selection. MarketValueGBP counts priced holdings only; GainGBP,
// GainPercent and YieldPercent are nil unless every holding was priced,
// since a partial market value would understate them.
type AllocationFigures struct {
	InstrumentCount          int              `json:"instrumentCount"`
	UnpricedCount            int              `json:"unpricedCount"`
	InvestedGBP              decimal.Decimal  `json:"investedGbp"`
	MarketValueGBP           decimal.Decimal  `json:"marketValueGbp"`
	GainGBP                  *decimal.Decimal `json:"gainGbp"`
	GainPercent              *decimal.Decimal `json:"gainPercent"`
	TrailingYearDividendsGBP decimal.Decimal  `json:"trailingYearDividendsGbp"`
	YieldPercent             *decimal.Decimal `json:"yieldPercent"`
	// LargestHolding is the priced holding with the most market value, and
	// its share of this group's (priced) market value.
	LargestHolding *HoldingShare `json:"largestHolding"`
	// EffectiveHoldings is how many equally sized holdings would give the
	// same concentration (1 / Σ weight², by market value): 1 means one
	// holding is everything, n means n equal holdings. Nil without a priced
	// market value.
	EffectiveHoldings *decimal.Decimal `json:"effectiveHoldings"`
}

type HoldingShare struct {
	Name         string          `json:"name"`
	Ticker       string          `json:"ticker"`
	SharePercent decimal.Decimal `json:"sharePercent"`
}

// Allocation is one group of current holdings. Parent is the sector or
// category a sub-sector or sub-category belongs to, nil at the top level;
// sub-level names are only unique within their parent.
type Allocation struct {
	Name         string  `json:"name"`
	Parent       *string `json:"parent"`
	Unclassified bool    `json:"unclassified"`
	AllocationFigures
	// SharePercent is the group's share of total market value.
	SharePercent decimal.Decimal `json:"sharePercent"`
}

type Allocations struct {
	GroupBy GroupBy           `json:"groupBy"`
	Totals  AllocationFigures `json:"totals"`
	Groups  []Allocation      `json:"groups"`
}

// Allocations groups current holdings by groupBy, honouring the Analytics
// instrument filter. Groups are sorted by market value, then invested
// value, descending.
func (s *SummaryService) Allocations(ctx context.Context, scope Scope, filter Filter, groupBy GroupBy) (*Allocations, error) {
	holdings, err := s.pricedHoldings(ctx, scope, filter)
	if err != nil {
		return nil, err
	}
	// Per instrument, so only held (and filtered) instruments contribute.
	dividends, err := s.repo.InstrumentTrailingYearDividends(ctx, scope)
	if err != nil {
		return nil, err
	}
	return aggregateAllocations(groupBy, holdings, dividends), nil
}

// figureTotals accumulates unrounded sums for one group.
type figureTotals struct {
	instruments int
	unpriced    int
	invested    decimal.Decimal
	marketValue decimal.Decimal
	dividends   decimal.Decimal
	// Priced holdings' market values, for concentration.
	values  []decimal.Decimal
	largest *pricedHolding
}

func (t *figureTotals) add(h pricedHolding, dividends decimal.Decimal) {
	t.instruments++
	t.invested = t.invested.Add(h.InvestedGBP)
	t.dividends = t.dividends.Add(dividends)
	if h.Priced {
		t.marketValue = t.marketValue.Add(h.MarketValueGBP)
		t.values = append(t.values, h.MarketValueGBP)
		if t.largest == nil || h.MarketValueGBP.GreaterThan(t.largest.MarketValueGBP) {
			t.largest = &h
		}
	} else {
		t.unpriced++
	}
}

func (t *figureTotals) figures() AllocationFigures {
	hundred := decimal.NewFromInt(100)
	f := AllocationFigures{
		InstrumentCount:          t.instruments,
		UnpricedCount:            t.unpriced,
		InvestedGBP:              t.invested.Round(2),
		MarketValueGBP:           t.marketValue.Round(2),
		TrailingYearDividendsGBP: t.dividends.Round(2),
	}
	if t.marketValue.IsPositive() {
		sumSquares := decimal.Zero
		for _, v := range t.values {
			w := v.Div(t.marketValue)
			sumSquares = sumSquares.Add(w.Mul(w))
		}
		if sumSquares.IsPositive() {
			f.EffectiveHoldings = decimalPtr(decimal.NewFromInt(1).Div(sumSquares).Round(1))
		}
		f.LargestHolding = &HoldingShare{
			Name:         t.largest.Name,
			Ticker:       t.largest.Ticker,
			SharePercent: t.largest.MarketValueGBP.Div(t.marketValue).Mul(hundred).Round(2),
		}
	}
	if t.unpriced > 0 {
		return f
	}
	f.GainGBP = decimalPtr(t.marketValue.Sub(t.invested).Round(2))
	if t.invested.IsPositive() {
		f.GainPercent = decimalPtr(t.marketValue.Sub(t.invested).Div(t.invested).Mul(hundred).Round(2))
	}
	if t.marketValue.IsPositive() {
		f.YieldPercent = decimalPtr(t.dividends.Div(t.marketValue).Mul(hundred).Round(2))
	}
	return f
}

type groupKey struct {
	parent       string
	name         string
	hasParent    bool
	unclassified bool
}

func keyFor(groupBy GroupBy, h CurrentHolding) groupKey {
	switch groupBy {
	case GroupBySubSector:
		if h.SubSector == nil {
			return groupKey{parent: h.Sector, name: UnclassifiedSubSector, hasParent: true, unclassified: true}
		}
		return groupKey{parent: h.Sector, name: *h.SubSector, hasParent: true}
	case GroupByCategory:
		return groupKey{name: h.Category}
	case GroupBySubCategory:
		return groupKey{parent: h.Category, name: h.SubCategory, hasParent: true}
	default:
		return groupKey{name: h.Sector}
	}
}

func aggregateAllocations(groupBy GroupBy, holdings []pricedHolding, dividends map[uuid.UUID]decimal.Decimal) *Allocations {
	var total figureTotals
	byKey := make(map[groupKey]*figureTotals)
	var order []groupKey

	for _, h := range holdings {
		k := keyFor(groupBy, h.CurrentHolding)
		t, ok := byKey[k]
		if !ok {
			t = &figureTotals{}
			byKey[k] = t
			order = append(order, k)
		}
		t.add(h, dividends[h.InstrumentID])
		total.add(h, dividends[h.InstrumentID])
	}

	groups := make([]Allocation, 0, len(order))
	for _, k := range order {
		t := byKey[k]
		a := Allocation{
			Name:              k.name,
			Unclassified:      k.unclassified,
			AllocationFigures: t.figures(),
		}
		if k.hasParent {
			parent := k.parent
			a.Parent = &parent
		}
		if total.marketValue.IsPositive() {
			a.SharePercent = t.marketValue.Div(total.marketValue).Mul(decimal.NewFromInt(100)).Round(2)
		}
		groups = append(groups, a)
	}

	sort.SliceStable(groups, func(i, j int) bool {
		if !groups[i].MarketValueGBP.Equal(groups[j].MarketValueGBP) {
			return groups[i].MarketValueGBP.GreaterThan(groups[j].MarketValueGBP)
		}
		return groups[i].InvestedGBP.GreaterThan(groups[j].InvestedGBP)
	})

	return &Allocations{GroupBy: groupBy, Totals: total.figures(), Groups: groups}
}
