package portfolio

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Transaction struct {
	ID             string    `json:"id"`
	EntryType      string    `json:"entryType"`
	Timestamp      time.Time `json:"timestamp"`
	InstrumentName *string   `json:"instrumentName"`
	Shares         string    `json:"shares"`
	NetAmountGBP   string    `json:"netAmountGbp"`
	TotalImpactGBP string    `json:"totalImpactGbp"`
}

// ValuePoint is one day's cumulative value in a time series.
type ValuePoint struct {
	Date     string          `json:"date"`
	ValueGBP decimal.Decimal `json:"valueGbp"`
}

// HoldingsSummary reports the value of instruments currently held (net
// shares > 0). InvestedValueGBP is the net cost basis (BUY minus SELL cash
// flows) for those holdings. MarketValueGBP requires live prices and is
// zero until a price feed is wired in.
type HoldingsSummary struct {
	InvestedValueGBP decimal.Decimal `json:"investedValueGbp"`
	MarketValueGBP   decimal.Decimal `json:"marketValueGbp"`
	// UnpricedHoldings counts holdings left out of MarketValueGBP because
	// no price could be fetched.
	UnpricedHoldings int `json:"unpricedHoldings"`
}

// DividendsSummary reports total dividend income received to date.
type DividendsSummary struct {
	TotalDividendsGBP decimal.Decimal `json:"totalDividendsGbp"`
}

// SectorAllocation is one sector's share of current holdings: invested
// (cost basis) value, current market value, the unrealized gain/loss
// between them, and how many distinct instruments it contains. GainGBP and
// GainPercent are zero if MarketValueGBP couldn't be priced for every
// instrument in the sector (see SummaryService.SectorAllocations).
type SectorAllocation struct {
	Sector          string          `json:"sector"`
	InstrumentCount int             `json:"instrumentCount"`
	InvestedGBP     decimal.Decimal `json:"investedGbp"`
	MarketValueGBP  decimal.Decimal `json:"marketValueGbp"`
	GainGBP         decimal.Decimal `json:"gainGbp"`
	GainPercent     decimal.Decimal `json:"gainPercent"`
}

// MonthlyDividend is total dividend income received in one calendar month.
type MonthlyDividend struct {
	Year     int             `json:"year"`
	Month    int             `json:"month"` // 1-12
	TotalGBP decimal.Decimal `json:"totalGbp"`
}

// CashFlow is one dated external cash flow into or out of the portfolio's
// invested holdings (not the ISA cash balance) — negative for money out
// (a purchase), positive for money in (a sale, dividend, or interest).
type CashFlow struct {
	Date      time.Time
	AmountGBP decimal.Decimal
}

// PortfolioIRR reports the money-weighted rate of return (XIRR) on
// invested holdings: BUY/SELL/DIVIDEND/INTEREST cash flows plus current
// market value as a final terminal inflow. CurrentHoldingsIRR is the same
// calculation restricted to cash flows for instruments still held today
// (excluding fully-exited positions) — nil if not enough activity exists
// for that narrower calculation.
type PortfolioIRR struct {
	IRR                float64  `json:"irr"`
	CurrentHoldingsIRR *float64 `json:"currentHoldingsIrr"`
}

// PassiveIncome reports a trailing-12-month income yield: dividend and
// interest income received in the last 365 days, as a percentage of
// current market value. ChangePercent compares TrailingYearGBP against the
// preceding 365-day period (365-730 days ago) and is nil if that prior
// period had no income to divide by.
type PassiveIncome struct {
	YieldPercent    decimal.Decimal  `json:"yieldPercent"`
	TrailingYearGBP decimal.Decimal  `json:"trailingYearGbp"`
	ChangePercent   *decimal.Decimal `json:"changePercent"`
}

// AssetOption is one currently-held instrument, for populating the
// Analytics page's category/asset multi-select.
type AssetOption struct {
	InstrumentID uuid.UUID `json:"instrumentId"`
	Ticker       string    `json:"ticker"`
	Name         string    `json:"name"`
}

// AssetCategory groups currently-held instruments by sector.
type AssetCategory struct {
	Sector      string        `json:"sector"`
	Instruments []AssetOption `json:"instruments"`
}

// HoldingRow is one currently-held instrument's full set of per-holding
// figures, for the Holdings table. PerShare figures and PortfolioSharePercent
// are nil/zero-valued in the ways noted per-field below when the underlying
// data isn't available (unpriced instrument, no dividend history, etc.) —
// callers must render those as "N/A" rather than 0.
type HoldingRow struct {
	InstrumentID uuid.UUID `json:"instrumentId"`
	Ticker       string    `json:"ticker"`
	Name         string    `json:"name"`
	Sector       string    `json:"sector"`

	Shares decimal.Decimal `json:"shares"`

	CostBasisGBP         decimal.Decimal `json:"costBasisGbp"`
	CostBasisPerShareGBP decimal.Decimal `json:"costBasisPerShareGbp"`

	// CurrentValueGBP and CurrentValuePerShareGBP are zero when Priced is
	// false (the instrument couldn't be priced — see pricedHolding).
	CurrentValueGBP         decimal.Decimal `json:"currentValueGbp"`
	CurrentValuePerShareGBP decimal.Decimal `json:"currentValuePerShareGbp"`
	Priced                  bool            `json:"priced"`

	DividendsReceivedGBP         decimal.Decimal `json:"dividendsReceivedGbp"`
	DividendsReceivedPerShareGBP decimal.Decimal `json:"dividendsReceivedPerShareGbp"`

	// DividendYieldPercent is trailing-12-month dividend income divided by
	// current value; nil if unpriced or zero trailing dividend income.
	DividendYieldPercent *decimal.Decimal `json:"dividendYieldPercent"`

	// DividendGrowthPercent compares total dividend income in the earliest
	// vs. latest full calendar year within the last 5 years of history
	// available for this instrument; nil if fewer than 2 distinct years of
	// dividend history exist.
	DividendGrowthPercent *decimal.Decimal `json:"dividendGrowthPercent"`

	// TotalProfitGBP/TotalProfitPercent are nil if unpriced.
	TotalProfitGBP     *decimal.Decimal `json:"totalProfitGbp"`
	TotalProfitPercent *decimal.Decimal `json:"totalProfitPercent"`

	// IRRPercent is the money-weighted return on this instrument's own cash
	// flows plus current value as a terminal inflow; nil if unpriced or the
	// solver has no defined rate (too little history, no sign change, or
	// non-convergence).
	IRRPercent *decimal.Decimal `json:"irrPercent"`

	// PortfolioSharePercent is this holding's current value as a percent of
	// the sum of every priced holding's current value; nil if unpriced.
	PortfolioSharePercent *decimal.Decimal `json:"portfolioSharePercent"`
}
