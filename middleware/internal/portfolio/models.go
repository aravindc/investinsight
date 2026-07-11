package portfolio

import (
	"time"

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
}

// DividendsSummary reports total dividend income received to date.
type DividendsSummary struct {
	TotalDividendsGBP decimal.Decimal `json:"totalDividendsGbp"`
}

// SectorAllocation is one sector's share of current holdings, by invested
// (cost basis) value.
type SectorAllocation struct {
	Sector      string          `json:"sector"`
	InvestedGBP decimal.Decimal `json:"investedGbp"`
}
