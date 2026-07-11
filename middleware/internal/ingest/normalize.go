package ingest

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type RawSourceTransaction struct {
	ID                    uuid.UUID
	Action                string
	TransactionTime       time.Time
	ISIN                  sql.NullString
	Ticker                sql.NullString
	Name                  sql.NullString
	NoOfShares            decimal.Decimal
	PricePerShare         decimal.Decimal
	CurrencyPricePerShare string
	ExchangeRate          decimal.Decimal
	Total                 decimal.Decimal
	CurrencyTotal         string
	WithholdingTax        decimal.Decimal
	StampDutyReserveTax   decimal.Decimal
	CurrencyConversionFee decimal.Decimal
}

type NormalizedTransaction struct {
	SourceID           uuid.UUID
	InstrumentID       *uuid.UUID
	EntryType          EntryType
	Timestamp          time.Time
	Shares             decimal.Decimal
	PricePerShareLocal decimal.Decimal
	FXRateToGBP        decimal.Decimal
	NetAmountGBP       decimal.Decimal
	FeesGBP            decimal.Decimal
	TaxesGBP           decimal.Decimal
	TotalImpactGBP     decimal.Decimal
}

func NormalizeTransaction(raw RawSourceTransaction, instrumentID *uuid.UUID) (*NormalizedTransaction, error) {
	entryType := MapAction(raw.Action)
	if entryType == EntryUnknown {
		return nil, fmt.Errorf("unsupported broker action: %s", raw.Action)
	}

	feesGBP := raw.CurrencyConversionFee
	taxesGBP := raw.WithholdingTax.Add(raw.StampDutyReserveTax)

	fxRate := raw.ExchangeRate
	if fxRate.IsZero() {
		fxRate = decimal.NewFromInt(1)
	}

	totalImpactGBP := raw.Total
	netAmountGBP := totalImpactGBP.Sub(feesGBP).Sub(taxesGBP)

	if entryType == EntryBuy {
		totalImpactGBP = totalImpactGBP.Abs().Neg()
	}

	return &NormalizedTransaction{
		SourceID:           raw.ID,
		InstrumentID:       instrumentID,
		EntryType:          entryType,
		Timestamp:          raw.TransactionTime,
		Shares:             raw.NoOfShares,
		PricePerShareLocal: raw.PricePerShare,
		FXRateToGBP:        fxRate,
		NetAmountGBP:       netAmountGBP.Abs(),
		FeesGBP:            feesGBP,
		TaxesGBP:           taxesGBP,
		TotalImpactGBP:     totalImpactGBP,
	}, nil
}

func isAssetEntry(entryType EntryType) bool {
	switch entryType {
	case EntryBuy, EntrySell, EntryDividend, EntryStockDistribution, EntryStockSplit, EntryAcquisition:
		return true
	default:
		return false
	}
}

func normalizeCurrency(rawCurrency string) string {
	currency := strings.ToUpper(rawCurrency)
	if currency == "GBX" {
		return "GBP"
	}
	return currency
}
