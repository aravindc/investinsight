package cashisa

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestValue_AccruesDailyCreditsMonthly(t *testing.T) {
	rates := []Rate{{EffectiveFrom: day("2026-01-01"), AnnualPercent: dec("3.65")}}
	// £10,000 at 3.65% is exactly £1.00 a day on a 365-day basis.

	// Mid-month: 9 completed days (1st–9th) accrued, nothing credited.
	v := Value(dec("10000"), day("2026-01-01"), rates, day("2026-01-10"))
	if !v.Value.Equal(dec("10000")) || !v.AccruedThisMonth.Equal(dec("9")) {
		t.Errorf("10 Jan: value %s accrued %s; want 10000 / 9", v.Value, v.AccruedThisMonth)
	}
	if !v.NextCreditDate.Equal(day("2026-01-31")) || !v.CurrentRate.Equal(dec("3.65")) {
		t.Errorf("10 Jan: next credit %s rate %s", v.NextCreditDate, v.CurrentRate)
	}

	// On 31 Jan, that day hasn't completed: 30 days accrued, not yet credited.
	v = Value(dec("10000"), day("2026-01-01"), rates, day("2026-01-31"))
	if !v.Value.Equal(dec("10000")) || !v.AccruedThisMonth.Equal(dec("30")) {
		t.Errorf("31 Jan: value %s accrued %s; want 10000 / 30", v.Value, v.AccruedThisMonth)
	}

	// 1 Feb: January's 31 days (£31) credited to the balance.
	v = Value(dec("10000"), day("2026-01-01"), rates, day("2026-02-01"))
	if !v.Value.Equal(dec("10031")) || !v.AccruedThisMonth.IsZero() || !v.InterestCredited.Equal(dec("31")) {
		t.Errorf("1 Feb: value %s accrued %s credited %s; want 10031 / 0 / 31", v.Value, v.AccruedThisMonth, v.InterestCredited)
	}

	// 1 Mar: February accrues on the larger balance (monthly compounding):
	// 28 × 10031 × 0.0365/365 = 28.0868 → 28.09.
	v = Value(dec("10000"), day("2026-01-01"), rates, day("2026-03-01"))
	if !v.Value.Equal(dec("10059.09")) {
		t.Errorf("1 Mar: value %s; want 10059.09", v.Value)
	}
}

func TestValue_RateChangeAppliesFromEffectiveDate(t *testing.T) {
	rates := []Rate{
		{EffectiveFrom: day("2026-01-01"), AnnualPercent: dec("3.65")}, // £1.00/day on £10k
		{EffectiveFrom: day("2026-01-11"), AnnualPercent: dec("7.30")}, // £2.00/day
	}
	v := Value(dec("10000"), day("2026-01-01"), rates, day("2026-01-21"))
	// 10 days × £1 + 10 days × £2
	if !v.AccruedThisMonth.Equal(dec("30")) || !v.CurrentRate.Equal(dec("7.3")) {
		t.Errorf("accrued %s rate %s; want 30 / 7.3", v.AccruedThisMonth, v.CurrentRate)
	}
}

func TestValue_StartsMidMonthAndIgnoresEarlierRates(t *testing.T) {
	rates := []Rate{
		{EffectiveFrom: day("2025-06-01"), AnnualPercent: dec("1")},
		{EffectiveFrom: day("2026-01-01"), AnnualPercent: dec("3.65")},
	}
	// Balance entered on 20 Jan: 20th–31st = 12 days credited on 31 Jan.
	v := Value(dec("10000"), day("2026-01-20"), rates, day("2026-02-01"))
	if !v.Value.Equal(dec("10012")) {
		t.Errorf("value %s; want 10012", v.Value)
	}
}

func TestValue_SameDayAndNoRate(t *testing.T) {
	v := Value(dec("500"), day("2026-01-05"), nil, day("2026-01-05"))
	if !v.Value.Equal(dec("500")) || !v.AccruedThisMonth.IsZero() || !v.CurrentRate.IsZero() {
		t.Errorf("same day / no rate: %+v", v)
	}
}

func TestValue_LeapYearFebruaryMonthEnd(t *testing.T) {
	rates := []Rate{{EffectiveFrom: day("2028-02-01"), AnnualPercent: dec("3.65")}}
	// 2028 is a leap year: February has 29 days, credited on the 29th.
	v := Value(dec("10000"), day("2028-02-01"), rates, day("2028-03-01"))
	if !v.Value.Equal(dec("10029")) {
		t.Errorf("value %s; want 10029", v.Value)
	}
}
