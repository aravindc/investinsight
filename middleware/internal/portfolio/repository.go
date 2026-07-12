package portfolio

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListTransactions(ctx context.Context, limit int, scope Scope) ([]Transaction, error) {
	scopeFilter, scopeArg := buildScopeFilter(scope, "pt", 2)
	query := fmt.Sprintf(`
		SELECT
			pt.id, pt.entry_type, pt.timestamp, i.name,
			pt.shares, pt.net_amount_gbp, pt.total_impact_gbp
		FROM portfolio_transactions pt
		LEFT JOIN instruments i ON i.id = pt.instrument_id
		WHERE %s
		ORDER BY pt.timestamp DESC
		LIMIT $1;`, scopeFilter)

	rows, err := r.db.QueryContext(ctx, query, limit, scopeArg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := []Transaction{}
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(
			&t.ID, &t.EntryType, &t.Timestamp, &t.InstrumentName,
			&t.Shares, &t.NetAmountGBP, &t.TotalImpactGBP,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}

	return transactions, rows.Err()
}

// CurrentHolding is one instrument still held, with its net share count and
// net invested (cost basis) amount.
type CurrentHolding struct {
	InstrumentID uuid.UUID
	Ticker       string
	BaseCurrency string
	NetShares    decimal.Decimal
	InvestedGBP  decimal.Decimal
}

// CurrentHoldings lists every instrument with net shares
// (BUY/SELL/STOCK_SPLIT/STOCK_DISTRIBUTION/ACQUISITION) greater than zero,
// i.e. still held today.
func (r *Repository) CurrentHoldings(ctx context.Context, scope Scope) ([]CurrentHolding, error) {
	scopeFilter, scopeArg := buildScopeFilter(scope, "pt", 1)
	query := fmt.Sprintf(`
		SELECT
			i.id, i.ticker, i.base_currency,
			SUM(
				CASE pt.entry_type
					WHEN 'BUY' THEN pt.shares
					WHEN 'SELL' THEN -pt.shares
					WHEN 'STOCK_SPLIT' THEN pt.shares
					WHEN 'STOCK_DISTRIBUTION' THEN pt.shares
					WHEN 'ACQUISITION' THEN pt.shares
					ELSE 0
				END
			) AS net_shares,
			SUM(
				CASE pt.entry_type
					WHEN 'BUY' THEN pt.net_amount_gbp
					WHEN 'SELL' THEN -pt.net_amount_gbp
					ELSE 0
				END
			) AS invested_gbp
		FROM portfolio_transactions pt
		JOIN instruments i ON i.id = pt.instrument_id
		WHERE %s
		GROUP BY i.id, i.ticker, i.base_currency
		HAVING SUM(
			CASE pt.entry_type
				WHEN 'BUY' THEN pt.shares
				WHEN 'SELL' THEN -pt.shares
				WHEN 'STOCK_SPLIT' THEN pt.shares
				WHEN 'STOCK_DISTRIBUTION' THEN pt.shares
				WHEN 'ACQUISITION' THEN pt.shares
				ELSE 0
			END
		) > 0;`, scopeFilter)

	rows, err := r.db.QueryContext(ctx, query, scopeArg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holdings []CurrentHolding
	for rows.Next() {
		var h CurrentHolding
		if err := rows.Scan(&h.InstrumentID, &h.Ticker, &h.BaseCurrency, &h.NetShares, &h.InvestedGBP); err != nil {
			return nil, err
		}
		holdings = append(holdings, h)
	}
	return holdings, rows.Err()
}

// ShareDelta is one day's net change in shares held for an instrument.
type ShareDelta struct {
	InstrumentID uuid.UUID
	Date         time.Time
	SharesDelta  decimal.Decimal
}

// ShareDeltasForCurrentHoldings returns every day with a share-count change
// (BUY/SELL/STOCK_SPLIT/STOCK_DISTRIBUTION/ACQUISITION) for instruments
// that are still held today, ordered by instrument then date. Callers
// compute running per-instrument totals via a cumulative sum over this
// list, then join against daily prices to reconstruct historical value.
func (r *Repository) ShareDeltasForCurrentHoldings(ctx context.Context, scope Scope) ([]ShareDelta, error) {
	outerFilter, outerArg := buildScopeFilter(scope, "pt", 1)
	innerFilter, _ := buildScopeFilter(scope, "", 1)
	query := fmt.Sprintf(`
		SELECT pt.instrument_id, DATE(pt.timestamp) AS day,
			SUM(
				CASE pt.entry_type
					WHEN 'BUY' THEN pt.shares
					WHEN 'SELL' THEN -pt.shares
					WHEN 'STOCK_SPLIT' THEN pt.shares
					WHEN 'STOCK_DISTRIBUTION' THEN pt.shares
					WHEN 'ACQUISITION' THEN pt.shares
					ELSE 0
				END
			) AS shares_delta
		FROM portfolio_transactions pt
		WHERE %s
		AND pt.instrument_id IN (
			SELECT instrument_id
			FROM portfolio_transactions
			WHERE %s
			GROUP BY instrument_id
			HAVING SUM(
				CASE entry_type
					WHEN 'BUY' THEN shares
					WHEN 'SELL' THEN -shares
					WHEN 'STOCK_SPLIT' THEN shares
					WHEN 'STOCK_DISTRIBUTION' THEN shares
					WHEN 'ACQUISITION' THEN shares
					ELSE 0
				END
			) > 0
		)
		GROUP BY pt.instrument_id, DATE(pt.timestamp)
		ORDER BY pt.instrument_id, day;`, outerFilter, innerFilter)

	rows, err := r.db.QueryContext(ctx, query, outerArg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deltas []ShareDelta
	for rows.Next() {
		var d ShareDelta
		if err := rows.Scan(&d.InstrumentID, &d.Date, &d.SharesDelta); err != nil {
			return nil, err
		}
		deltas = append(deltas, d)
	}
	return deltas, rows.Err()
}

// InvestedValueSeries returns the cumulative net capital invested (BUY minus
// SELL cash flow, across all instruments ever traded) as of each day that
// had BUY/SELL activity. The value at any point is the true running total
// to date, not reset per period — callers filter/zoom into a date range.
func (r *Repository) InvestedValueSeries(ctx context.Context, scope Scope) ([]ValuePoint, error) {
	scopeFilter, scopeArg := buildScopeFilter(scope, "", 1)
	query := fmt.Sprintf(`
		SELECT
			day,
			SUM(net_change) OVER (ORDER BY day) AS cumulative_invested
		FROM (
			SELECT
				DATE(timestamp) AS day,
				SUM(
					CASE entry_type
						WHEN 'BUY' THEN net_amount_gbp
						WHEN 'SELL' THEN -net_amount_gbp
						ELSE 0
					END
				) AS net_change
			FROM portfolio_transactions
			WHERE entry_type IN ('BUY', 'SELL') AND %s
			GROUP BY DATE(timestamp)
		) daily
		ORDER BY day;`, scopeFilter)

	rows, err := r.db.QueryContext(ctx, query, scopeArg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	series := []ValuePoint{}
	for rows.Next() {
		var day time.Time
		var value decimal.Decimal
		if err := rows.Scan(&day, &value); err != nil {
			return nil, err
		}
		series = append(series, ValuePoint{Date: day.Format("2006-01-02"), ValueGBP: value})
	}

	return series, rows.Err()
}

// SectorAllocations groups current holdings' invested (cost basis) value by
// sector, using the static ticker-to-sector mapping. Returned in descending
// order by invested value.
// DividendsSummary sums net dividend income received across all holdings.
func (r *Repository) DividendsSummary(ctx context.Context, scope Scope) (*DividendsSummary, error) {
	scopeFilter, scopeArg := buildScopeFilter(scope, "", 1)
	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(net_amount_gbp), 0)
		FROM portfolio_transactions
		WHERE entry_type = 'DIVIDEND' AND %s;`, scopeFilter)

	var total decimal.Decimal
	if err := r.db.QueryRowContext(ctx, query, scopeArg).Scan(&total); err != nil {
		return nil, err
	}

	return &DividendsSummary{TotalDividendsGBP: total}, nil
}

// TrailingYearIncome sums dividend and interest income received in the
// last 365 days (the numerator for the passive-income yield) and in the
// preceding 365 days (365-730 days ago), for a year-over-year comparison.
func (r *Repository) TrailingYearIncome(ctx context.Context, scope Scope) (trailing, prior decimal.Decimal, err error) {
	scopeFilter, scopeArg := buildScopeFilter(scope, "", 1)
	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(net_amount_gbp) FILTER (
				WHERE timestamp >= NOW() - INTERVAL '365 days'
			), 0) AS trailing,
			COALESCE(SUM(net_amount_gbp) FILTER (
				WHERE timestamp >= NOW() - INTERVAL '730 days'
				AND timestamp < NOW() - INTERVAL '365 days'
			), 0) AS prior
		FROM portfolio_transactions
		WHERE entry_type IN ('DIVIDEND', 'INTEREST') AND %s;`, scopeFilter)

	if err := r.db.QueryRowContext(ctx, query, scopeArg).Scan(&trailing, &prior); err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	return trailing, prior, nil
}

// MonthlyDividends returns total dividend income received in each calendar
// month that had any, across the full ledger history, ordered
// chronologically. Used to derive both the cumulative dividends-received
// series and the year-over-year dividend growth comparison.
func (r *Repository) MonthlyDividends(ctx context.Context, scope Scope) ([]MonthlyDividend, error) {
	scopeFilter, scopeArg := buildScopeFilter(scope, "", 1)
	query := fmt.Sprintf(`
		SELECT
			EXTRACT(YEAR FROM timestamp)::int AS year,
			EXTRACT(MONTH FROM timestamp)::int AS month,
			SUM(net_amount_gbp) AS total
		FROM portfolio_transactions
		WHERE entry_type = 'DIVIDEND' AND %s
		GROUP BY year, month
		ORDER BY year, month;`, scopeFilter)

	rows, err := r.db.QueryContext(ctx, query, scopeArg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	months := []MonthlyDividend{}
	for rows.Next() {
		var m MonthlyDividend
		if err := rows.Scan(&m.Year, &m.Month, &m.TotalGBP); err != nil {
			return nil, err
		}
		months = append(months, m)
	}
	return months, rows.Err()
}

// InvestmentCashFlows returns every BUY/SELL/DIVIDEND/INTEREST transaction
// as a dated external cash flow (total_impact_gbp, already sign-correct:
// negative for BUY, positive for the others), ordered chronologically.
// Deposits and withdrawals are excluded — those move money into/out of the
// ISA cash balance, not into/out of invested holdings. Used as the input
// series for the portfolio's money-weighted rate of return (XIRR).
func (r *Repository) InvestmentCashFlows(ctx context.Context, scope Scope) ([]CashFlow, error) {
	scopeFilter, scopeArg := buildScopeFilter(scope, "", 1)
	query := fmt.Sprintf(`
		SELECT timestamp, total_impact_gbp
		FROM portfolio_transactions
		WHERE entry_type IN ('BUY', 'SELL', 'DIVIDEND', 'INTEREST') AND %s
		ORDER BY timestamp;`, scopeFilter)

	rows, err := r.db.QueryContext(ctx, query, scopeArg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	flows := []CashFlow{}
	for rows.Next() {
		var f CashFlow
		if err := rows.Scan(&f.Date, &f.AmountGBP); err != nil {
			return nil, err
		}
		flows = append(flows, f)
	}
	return flows, rows.Err()
}

// CurrentHoldingsCashFlows returns BUY/SELL/DIVIDEND cash flows (excluding
// INTEREST, which has no instrument_id and so can't be attributed to a
// position) for instruments that are still held today, i.e. narrower than
// InvestmentCashFlows: fully-exited positions are excluded entirely.
func (r *Repository) CurrentHoldingsCashFlows(ctx context.Context, scope Scope) ([]CashFlow, error) {
	outerFilter, outerArg := buildScopeFilter(scope, "pt", 1)
	innerFilter, _ := buildScopeFilter(scope, "", 1)
	query := fmt.Sprintf(`
		SELECT pt.timestamp, pt.total_impact_gbp
		FROM portfolio_transactions pt
		WHERE pt.entry_type IN ('BUY', 'SELL', 'DIVIDEND')
		AND %s
		AND pt.instrument_id IN (
			SELECT instrument_id
			FROM portfolio_transactions
			WHERE %s
			GROUP BY instrument_id
			HAVING SUM(
				CASE entry_type
					WHEN 'BUY' THEN shares
					WHEN 'SELL' THEN -shares
					WHEN 'STOCK_SPLIT' THEN shares
					WHEN 'STOCK_DISTRIBUTION' THEN shares
					WHEN 'ACQUISITION' THEN shares
					ELSE 0
				END
			) > 0
		)
		ORDER BY pt.timestamp;`, outerFilter, innerFilter)

	rows, err := r.db.QueryContext(ctx, query, outerArg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	flows := []CashFlow{}
	for rows.Next() {
		var f CashFlow
		if err := rows.Scan(&f.Date, &f.AmountGBP); err != nil {
			return nil, err
		}
		flows = append(flows, f)
	}
	return flows, rows.Err()
}
