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
	EntryStockSplit        EntryType = "STOCK_SPLIT"
	EntryAcquisition       EntryType = "ACQUISITION"
	EntryUnknown           EntryType = "UNKNOWN"
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
	case strings.Contains(action, "stock split"):
		return EntryStockSplit
	case strings.Contains(action, "acquisition"):
		return EntryAcquisition
	default:
		return EntryUnknown
	}
}
