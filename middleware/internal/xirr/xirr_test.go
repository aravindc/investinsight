package xirr

import (
	"math"
	"testing"
	"time"
)

func TestCompute_SimpleAnnualDouble(t *testing.T) {
	// Invest 100 on day 0, receive 200 exactly one year later: 100% return.
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	flows := []CashFlow{
		{Date: start, Amount: -100},
		{Date: start.AddDate(1, 0, 0), Amount: 200},
	}

	rate, err := Compute(flows)
	if err != nil {
		t.Fatalf("Compute returned error: %v", err)
	}
	// 2020 is a leap year, so day-count/365 gives slightly more than
	// exactly 1.0 years between the flows — expect ~100%, not exact.
	if math.Abs(rate-1.0) > 0.005 {
		t.Errorf("expected rate ~1.0 (100%%), got %v", rate)
	}
}

func TestCompute_KnownExcelExample(t *testing.T) {
	// Classic XIRR worked example: -10000 on 2008-01-01, then 2750, 4250,
	// 3250, 2750 on quarterly-ish dates; Excel's XIRR gives ~37.34%.
	flows := []CashFlow{
		{Date: date(2008, 1, 1), Amount: -10000},
		{Date: date(2008, 3, 1), Amount: 2750},
		{Date: date(2008, 10, 30), Amount: 4250},
		{Date: date(2009, 2, 15), Amount: 3250},
		{Date: date(2009, 4, 1), Amount: 2750},
	}

	rate, err := Compute(flows)
	if err != nil {
		t.Fatalf("Compute returned error: %v", err)
	}
	if math.Abs(rate-0.3734) > 0.001 {
		t.Errorf("expected rate ~0.3734, got %v", rate)
	}
}

func TestCompute_TooFewFlows(t *testing.T) {
	_, err := Compute([]CashFlow{{Date: date(2020, 1, 1), Amount: -100}})
	if err != ErrNoCashFlows {
		t.Errorf("expected ErrNoCashFlows, got %v", err)
	}
}

func TestCompute_NoSignChange(t *testing.T) {
	flows := []CashFlow{
		{Date: date(2020, 1, 1), Amount: 100},
		{Date: date(2021, 1, 1), Amount: 200},
	}
	_, err := Compute(flows)
	if err != ErrNoSignChange {
		t.Errorf("expected ErrNoSignChange, got %v", err)
	}
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
