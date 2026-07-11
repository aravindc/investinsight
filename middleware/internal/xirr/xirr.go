// Package xirr computes the money-weighted rate of return (XIRR) for a
// series of irregularly-dated cash flows — the same calculation as Excel's
// XIRR function. There's no closed-form solution, so it's solved
// numerically: Newton-Raphson for fast convergence in the common case,
// falling back to bisection (slower but guaranteed to converge within a
// bracketed range) if Newton's method fails to converge or leaves the
// bracket.
package xirr

import (
	"errors"
	"math"
	"time"
)

// CashFlow is one dated cash flow: negative for money out (e.g. a
// purchase), positive for money in (e.g. a sale, dividend, or the final
// terminal value of holdings still held).
type CashFlow struct {
	Date   time.Time
	Amount float64
}

var (
	// ErrNoCashFlows is returned when there are fewer than two cash flows —
	// a rate of return isn't defined for less than that.
	ErrNoCashFlows = errors.New("xirr: need at least two cash flows")
	// ErrNoSignChange is returned when every cash flow has the same sign —
	// no discount rate can make them net to zero, so no rate exists.
	ErrNoSignChange = errors.New("xirr: cash flows must include both positive and negative amounts")
	// ErrDidNotConverge is returned when neither Newton-Raphson nor the
	// bisection fallback found a root within the iteration budget.
	ErrDidNotConverge = errors.New("xirr: solver did not converge")
)

const (
	maxNewtonIterations    = 50
	maxBisectionIterations = 200
	tolerance              = 1e-7
	daysPerYear            = 365.0
)

// Compute returns the annualized money-weighted rate of return (e.g. 0.08
// for 8%) for the given cash flows. Flows do not need to be pre-sorted.
func Compute(flows []CashFlow) (float64, error) {
	if len(flows) < 2 {
		return 0, ErrNoCashFlows
	}

	hasPositive, hasNegative := false, false
	for _, f := range flows {
		if f.Amount > 0 {
			hasPositive = true
		} else if f.Amount < 0 {
			hasNegative = true
		}
	}
	if !hasPositive || !hasNegative {
		return 0, ErrNoSignChange
	}

	t0 := flows[0].Date
	for _, f := range flows {
		if f.Date.Before(t0) {
			t0 = f.Date
		}
	}

	years := make([]float64, len(flows))
	amounts := make([]float64, len(flows))
	for i, f := range flows {
		years[i] = f.Date.Sub(t0).Hours() / 24 / daysPerYear
		amounts[i] = f.Amount
	}

	npv := func(rate float64) float64 {
		total := 0.0
		for i := range amounts {
			total += amounts[i] / math.Pow(1+rate, years[i])
		}
		return total
	}

	npvDerivative := func(rate float64) float64 {
		total := 0.0
		for i := range amounts {
			if years[i] == 0 {
				continue
			}
			total -= years[i] * amounts[i] / math.Pow(1+rate, years[i]+1)
		}
		return total
	}

	if rate, ok := newtonRaphson(npv, npvDerivative, 0.1); ok {
		return rate, nil
	}

	if rate, ok := bisection(npv, -0.999999, 100); ok {
		return rate, nil
	}

	return 0, ErrDidNotConverge
}

func newtonRaphson(npv, npvDerivative func(float64) float64, guess float64) (float64, bool) {
	rate := guess
	for i := 0; i < maxNewtonIterations; i++ {
		value := npv(rate)
		if math.Abs(value) < tolerance {
			return rate, true
		}
		deriv := npvDerivative(rate)
		if deriv == 0 || math.IsNaN(deriv) || math.IsInf(deriv, 0) {
			return 0, false
		}
		next := rate - value/deriv
		if math.IsNaN(next) || math.IsInf(next, 0) || next <= -1 {
			return 0, false
		}
		if math.Abs(next-rate) < tolerance {
			return next, true
		}
		rate = next
	}
	return 0, false
}

// bisection requires npv(low) and npv(high) to have opposite signs
// (a bracketed root). low/high are rates, e.g. -0.999999 to 100 (-99.9999%
// to +10000%), a wide enough range to bracket any realistic XIRR.
func bisection(npv func(float64) float64, low, high float64) (float64, bool) {
	npvLow, npvHigh := npv(low), npv(high)
	if math.IsNaN(npvLow) || math.IsNaN(npvHigh) || npvLow*npvHigh > 0 {
		return 0, false
	}

	for i := 0; i < maxBisectionIterations; i++ {
		mid := (low + high) / 2
		npvMid := npv(mid)
		if math.Abs(npvMid) < tolerance || (high-low)/2 < tolerance {
			return mid, true
		}
		if npvLow*npvMid < 0 {
			high = mid
		} else {
			low = mid
			npvLow = npvMid
		}
	}
	return 0, false
}
