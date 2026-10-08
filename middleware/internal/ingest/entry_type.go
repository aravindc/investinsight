package ingest

import "strings"

type EntryType string

const (
	EntryBuy               EntryType = "BUY"
	EntrySell              EntryType = "SELL"
	EntryDividend          EntryType = "DIVIDEND"
	EntryDeposit           EntryType = "DEPOSIT"
	EntryWithdrawal        EntryType = "WITHDRAWAL"
	EntryInterest          EntryType = "INTEREST"
	EntryStockDistribution EntryType = "STOCK_DISTRIBUTION"
	// EntryStockSplitClose/EntryStockSplitOpen are the two ledger rows a
	// broker emits per stock split event: "close" removes the pre-split
	// share count, "open" adds the post-split share count. They must stay
	// distinct (not collapsed into one STOCK_SPLIT type) because the
	// direction differs — treating both as additive double-counts shares
	// after a reverse split.
	EntryStockSplitClose EntryType = "STOCK_SPLIT_CLOSE"
	EntryStockSplitOpen  EntryType = "STOCK_SPLIT_OPEN"
	EntryAcquisition     EntryType = "ACQUISITION"
	// EntryFee is a broker charge levied against a holding (e.g. an ADR
	// custody fee) — a cash outflow with no change in share count.
	EntryFee     EntryType = "FEE"
	EntryUnknown EntryType = "UNKNOWN"
)

// MapAction normalizes broker-specific action text into system entry types.
func MapAction(brokerAction string) EntryType {
	action := strings.ToLower(strings.TrimSpace(brokerAction))
	switch {
	case strings.Contains(action, "buy"):
		return EntryBuy
	case strings.Contains(action, "sell"):
		return EntrySell
	case strings.Contains(action, "dividend") || strings.Contains(action, "div"):
		return EntryDividend
	case strings.Contains(action, "deposit") || strings.Contains(action, "top-up"):
		return EntryDeposit
	case strings.Contains(action, "withdrawal"):
		return EntryWithdrawal
	case strings.Contains(action, "interest") || strings.Contains(action, "result adjustment"):
		return EntryInterest
	case strings.Contains(action, "stock distribution"):
		return EntryStockDistribution
	case strings.Contains(action, "stock split close"):
		return EntryStockSplitClose
	case strings.Contains(action, "stock split open"):
		return EntryStockSplitOpen
	case strings.Contains(action, "acquisition"):
		return EntryAcquisition
	case strings.Contains(action, "fee"):
		return EntryFee
	default:
		return EntryUnknown
	}
}
