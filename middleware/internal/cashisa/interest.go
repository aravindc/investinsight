package cashisa

import (
	"time"

	"github.com/shopspring/decimal"
)

// daysPerYear is the day-count basis UK banks use for daily interest,
// including in leap years.
const daysPerYear = 365

var hundred = decimal.NewFromInt(100)

// Rate is an annual interest rate applying from EffectiveFrom (a date at
// midnight UTC) until the next rate takes over.
type Rate struct {
	EffectiveFrom time.Time       `json:"effectiveFrom"`
	AnnualPercent decimal.Decimal `json:"ratePercent"`
}

// Valuation is a Cash ISA's position on a given day.
type Valuation struct {
	// Value is the balance including interest credited at each completed
	// month end.
	Value decimal.Decimal
	// AccruedThisMonth is interest earned in the current month so far,
	// credited to Value at the month's end.
	AccruedThisMonth decimal.Decimal
	// InterestCredited is the total credited since the balance date.
	InterestCredited decimal.Decimal
	// CurrentRate is the rate applying today (zero if none).
	CurrentRate decimal.Decimal
	// NextCreditDate is the last day of the current month.
	NextCreditDate time.Time
}

// Value replays interest from balanceDate up to (not including) asOf:
// each completed day accrues balance × rate ÷ 365, and on the last day of
// each month the month's accrual, rounded to the penny, is credited to the
// balance — so interest compounds monthly. Today's interest accrues once
// the day is over. Dates are calendar days at midnight UTC.
func Value(balance decimal.Decimal, balanceDate time.Time, rates []Rate, asOf time.Time) Valuation {
	balanceDate, asOf = civil(balanceDate), civil(asOf)

	accrued := decimal.Zero
	credited := decimal.Zero
	for d := balanceDate; d.Before(asOf); d = d.AddDate(0, 0, 1) {
		rate := rateOn(rates, d)
		accrued = accrued.Add(balance.Mul(rate).Div(hundred).Div(decimal.NewFromInt(daysPerYear)))

		if isMonthEnd(d) {
			interest := accrued.Round(2)
			balance = balance.Add(interest)
			credited = credited.Add(interest)
			accrued = decimal.Zero
		}
	}

	return Valuation{
		Value:            balance.Round(2),
		AccruedThisMonth: accrued.Round(2),
		InterestCredited: credited,
		CurrentRate:      rateOn(rates, asOf),
		NextCreditDate:   monthEnd(asOf),
	}
}

// rateOn returns the rate in force on day d: the latest rate whose
// EffectiveFrom is on or before d. rates must be sorted by EffectiveFrom.
func rateOn(rates []Rate, d time.Time) decimal.Decimal {
	rate := decimal.Zero
	for _, r := range rates {
		if civil(r.EffectiveFrom).After(d) {
			break
		}
		rate = r.AnnualPercent
	}
	return rate
}

func civil(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func isMonthEnd(d time.Time) bool {
	return d.AddDate(0, 0, 1).Day() == 1
}

func monthEnd(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month()+1, 0, 0, 0, 0, 0, time.UTC)
}
