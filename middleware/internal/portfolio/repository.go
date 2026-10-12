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
	Name         string
	BaseCurrency string
	ISIN         string
	Sector       string
	SubSector    *string // nil when the instrument has no sub-sector
	Category     string
	SubCategory  string
	NetShares    decimal.Decimal
	InvestedGBP  decimal.Decimal
}

// instrumentLedgerRow is one share/cost-affecting transaction, ordered for
// the average-cost pool walk in CurrentHoldings.
type instrumentLedgerRow struct {
	InstrumentID uuid.UUID
	Ticker       string
	Name         string
	BaseCurrency string
	ISIN         string
	Sector       string
	SubSector    *string
	Category     string
	SubCategory  string
	EntryType    string
	Timestamp    time.Time
	Shares       decimal.Decimal
	NetAmountGBP decimal.Decimal
}

// CurrentHoldings lists every instrument with net shares
// (BUY/SELL/STOCK_SPLIT_OPEN/STOCK_SPLIT_CLOSE/STOCK_DISTRIBUTION/ACQUISITION) greater than zero,
// i.e. still held today. InvestedGBP is the remaining cost basis under
// average-cost (UK Section 104 pool) accounting: each BUY adds its shares
// and cost to a running pool; each SELL removes shares from the pool at
// the pool's current average cost per share (not at the sale's proceeds,
// which is what a naive "total BUY cash minus total SELL cash" gets
// wrong for any instrument that's been partially sold at a different
// price than its average cost). Splits/distributions/acquisitions adjust
// share count only — pool cost is unchanged, since no cash moves.
func (r *Repository) CurrentHoldings(ctx context.Context, scope Scope, filter Filter) ([]CurrentHolding, error) {
	scopeFilter, scopeArg := buildScopeFilter(scope, "pt", 1)
	args := []any{scopeArg}
	where := scopeFilter
	if instFilter, instArg, ok := buildInstrumentFilter(filter, "pt", 2); ok {
		where += " AND " + instFilter
		args = append(args, instArg)
	}

	query := fmt.Sprintf(`
		SELECT
			i.id, i.ticker, i.name, i.base_currency, i.isin, i.sector,
			i.sub_sector, i.category, i.sub_category,
			pt.entry_type, pt.timestamp, pt.shares, pt.net_amount_gbp
		FROM portfolio_transactions pt
		JOIN instruments i ON i.id = pt.instrument_id
		WHERE %s
		AND pt.entry_type IN ('BUY', 'SELL', 'STOCK_SPLIT_OPEN', 'STOCK_SPLIT_CLOSE', 'STOCK_DISTRIBUTION', 'ACQUISITION')
		ORDER BY i.id, pt.timestamp;`, where)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ledger []instrumentLedgerRow
	for rows.Next() {
		var l instrumentLedgerRow
		if err := rows.Scan(&l.InstrumentID, &l.Ticker, &l.Name, &l.BaseCurrency, &l.ISIN, &l.Sector, &l.SubSector, &l.Category, &l.SubCategory, &l.EntryType, &l.Timestamp, &l.Shares, &l.NetAmountGBP); err != nil {
			return nil, err
		}
		ledger = append(ledger, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return poolCurrentHoldings(ledger), nil
}

// poolCurrentHoldings walks ledger (already ordered by instrument then
// timestamp) maintaining a running (shares, pool cost) pair per
// instrument, and returns one CurrentHolding for every instrument whose
// final share count is positive.
func poolCurrentHoldings(ledger []instrumentLedgerRow) []CurrentHolding {
	type pool struct {
		holding CurrentHolding
		shares  decimal.Decimal
		cost    decimal.Decimal
	}
	pools := make(map[uuid.UUID]*pool)
	var order []uuid.UUID

	for _, l := range ledger {
		p, ok := pools[l.InstrumentID]
		if !ok {
			p = &pool{holding: CurrentHolding{
				InstrumentID: l.InstrumentID,
				Ticker:       l.Ticker,
				Name:         l.Name,
				BaseCurrency: l.BaseCurrency,
				ISIN:         l.ISIN,
				Sector:       l.Sector,
				SubSector:    l.SubSector,
				Category:     l.Category,
				SubCategory:  l.SubCategory,
			}}
			pools[l.InstrumentID] = p
			order = append(order, l.InstrumentID)
		}

		switch l.EntryType {
		case "BUY":
			p.shares = p.shares.Add(l.Shares)
			p.cost = p.cost.Add(l.NetAmountGBP)
		case "SELL":
			if p.shares.IsPositive() {
				// Remove shares at the pool's current average cost, not
				// at the sale's proceeds. DivRound to a fixed precision —
				// shopspring/decimal's plain Div can otherwise expand to
				// an unbounded number of decimal places.
				soldFraction := l.Shares.DivRound(p.shares, 10)
				if soldFraction.GreaterThan(decimal.NewFromInt(1)) {
					soldFraction = decimal.NewFromInt(1)
				}
				p.cost = p.cost.Sub(p.cost.Mul(soldFraction)).Round(4)
			}
			p.shares = p.shares.Sub(l.Shares)
		case "STOCK_SPLIT_OPEN", "STOCK_DISTRIBUTION", "ACQUISITION":
			p.shares = p.shares.Add(l.Shares)
		case "STOCK_SPLIT_CLOSE":
			p.shares = p.shares.Sub(l.Shares)
		}
	}

	holdings := make([]CurrentHolding, 0, len(pools))
	for _, id := range order {
		p := pools[id]
		if !p.shares.IsPositive() {
			continue
		}
		p.holding.NetShares = p.shares
		p.holding.InvestedGBP = p.cost
		holdings = append(holdings, p.holding)
	}
	return holdings
}

// ShareDelta is one day's net change in shares held for an instrument.
type ShareDelta struct {
	InstrumentID uuid.UUID
	Date         time.Time
	SharesDelta  decimal.Decimal
}

// ShareDeltasForCurrentHoldings returns every day with a share-count change
// (BUY/SELL/STOCK_SPLIT_OPEN/STOCK_SPLIT_CLOSE/STOCK_DISTRIBUTION/ACQUISITION) for instruments
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
					WHEN 'STOCK_SPLIT_OPEN' THEN pt.shares
						WHEN 'STOCK_SPLIT_CLOSE' THEN -pt.shares
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
					WHEN 'STOCK_SPLIT_OPEN' THEN shares
					WHEN 'STOCK_SPLIT_CLOSE' THEN -shares
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
func (r *Repository) TrailingYearIncome(ctx context.Context, scope Scope, filter Filter) (trailing, prior decimal.Decimal, err error) {
	scopeFilter, scopeArg := buildScopeFilter(scope, "", 1)
	args := []any{scopeArg}

	// INTEREST has no instrument_id, so it can never match an instrument
	// allowlist — when a filter is active, drop it from the entry_type list
	// explicitly (rather than relying on "NULL = ANY(...)" being falsy) so
	// "cash income excluded once filtered" is visible in the SQL.
	where := "entry_type IN ('DIVIDEND', 'INTEREST') AND " + scopeFilter
	if instFilter, instArg, ok := buildInstrumentFilter(filter, "", 2); ok {
		where = "entry_type = 'DIVIDEND' AND " + scopeFilter + " AND " + instFilter
		args = append(args, instArg)
	}

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
		WHERE %s;`, where)

	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&trailing, &prior); err != nil {
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
func (r *Repository) InvestmentCashFlows(ctx context.Context, scope Scope, filter Filter) ([]CashFlow, error) {
	scopeFilter, scopeArg := buildScopeFilter(scope, "", 1)
	args := []any{scopeArg}

	// Same reasoning as TrailingYearIncome: INTEREST has no instrument_id
	// and is dropped explicitly once a filter is active.
	where := "entry_type IN ('BUY', 'SELL', 'DIVIDEND', 'INTEREST') AND " + scopeFilter
	if instFilter, instArg, ok := buildInstrumentFilter(filter, "", 2); ok {
		where = "entry_type IN ('BUY', 'SELL', 'DIVIDEND') AND " + scopeFilter + " AND " + instFilter
		args = append(args, instArg)
	}

	query := fmt.Sprintf(`
		SELECT timestamp, total_impact_gbp
		FROM portfolio_transactions
		WHERE %s
		ORDER BY timestamp;`, where)

	rows, err := r.db.QueryContext(ctx, query, args...)
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

// InstrumentDividends sums dividend income received per instrument, across
// all currently-held instruments matching scope. Instruments with no
// dividend history are simply absent from the returned map.
func (r *Repository) InstrumentDividends(ctx context.Context, scope Scope) (map[uuid.UUID]decimal.Decimal, error) {
	scopeFilter, scopeArg := buildScopeFilter(scope, "", 1)
	query := fmt.Sprintf(`
		SELECT instrument_id, SUM(net_amount_gbp)
		FROM portfolio_transactions
		WHERE entry_type = 'DIVIDEND' AND instrument_id IS NOT NULL AND %s
		GROUP BY instrument_id;`, scopeFilter)

	rows, err := r.db.QueryContext(ctx, query, scopeArg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	totals := make(map[uuid.UUID]decimal.Decimal)
	for rows.Next() {
		var id uuid.UUID
		var total decimal.Decimal
		if err := rows.Scan(&id, &total); err != nil {
			return nil, err
		}
		totals[id] = total
	}
	return totals, rows.Err()
}

// InstrumentTrailingYearDividends sums dividend income received in the last
// 365 days, per instrument. Used as the numerator for per-holding dividend
// yield.
func (r *Repository) InstrumentTrailingYearDividends(ctx context.Context, scope Scope) (map[uuid.UUID]decimal.Decimal, error) {
	scopeFilter, scopeArg := buildScopeFilter(scope, "", 1)
	query := fmt.Sprintf(`
		SELECT instrument_id, SUM(net_amount_gbp)
		FROM portfolio_transactions
		WHERE entry_type = 'DIVIDEND' AND instrument_id IS NOT NULL AND timestamp >= NOW() - INTERVAL '365 days' AND %s
		GROUP BY instrument_id;`, scopeFilter)

	rows, err := r.db.QueryContext(ctx, query, scopeArg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	totals := make(map[uuid.UUID]decimal.Decimal)
	for rows.Next() {
		var id uuid.UUID
		var total decimal.Decimal
		if err := rows.Scan(&id, &total); err != nil {
			return nil, err
		}
		totals[id] = total
	}
	return totals, rows.Err()
}

// InstrumentYearlyDividend is one instrument's total dividend income
// received in one calendar year.
type InstrumentYearlyDividend struct {
	InstrumentID uuid.UUID
	Year         int
	TotalGBP     decimal.Decimal
}

// InstrumentYearlyDividends returns total dividend income per instrument
// per calendar year, across the full ledger history, for computing
// per-holding dividend growth.
func (r *Repository) InstrumentYearlyDividends(ctx context.Context, scope Scope) ([]InstrumentYearlyDividend, error) {
	scopeFilter, scopeArg := buildScopeFilter(scope, "", 1)
	query := fmt.Sprintf(`
		SELECT instrument_id, EXTRACT(YEAR FROM timestamp)::int AS year, SUM(net_amount_gbp)
		FROM portfolio_transactions
		WHERE entry_type = 'DIVIDEND' AND instrument_id IS NOT NULL AND %s
		GROUP BY instrument_id, year
		ORDER BY instrument_id, year;`, scopeFilter)

	rows, err := r.db.QueryContext(ctx, query, scopeArg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var yearly []InstrumentYearlyDividend
	for rows.Next() {
		var y InstrumentYearlyDividend
		if err := rows.Scan(&y.InstrumentID, &y.Year, &y.TotalGBP); err != nil {
			return nil, err
		}
		yearly = append(yearly, y)
	}
	return yearly, rows.Err()
}

// InstrumentCashFlow is one dated external cash flow attributed to a single
// instrument (see CurrentHoldingsCashFlows for the flat, non-attributed
// version).
type InstrumentCashFlow struct {
	InstrumentID uuid.UUID
	Date         time.Time
	AmountGBP    decimal.Decimal
}

// InstrumentCashFlows returns BUY/SELL/DIVIDEND cash flows (excluding
// INTEREST, which has no instrument_id), attributed per instrument, for
// instruments still held today — the per-holding equivalent of
// CurrentHoldingsCashFlows, used to compute each holding's own IRR.
func (r *Repository) InstrumentCashFlows(ctx context.Context, scope Scope) ([]InstrumentCashFlow, error) {
	outerFilter, outerArg := buildScopeFilter(scope, "pt", 1)
	innerFilter, _ := buildScopeFilter(scope, "", 1)

	query := fmt.Sprintf(`
		SELECT pt.instrument_id, pt.timestamp, pt.total_impact_gbp
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
					WHEN 'STOCK_SPLIT_OPEN' THEN shares
					WHEN 'STOCK_SPLIT_CLOSE' THEN -shares
					WHEN 'STOCK_DISTRIBUTION' THEN shares
					WHEN 'ACQUISITION' THEN shares
					ELSE 0
				END
			) > 0
		)
		ORDER BY pt.instrument_id, pt.timestamp;`, outerFilter, innerFilter)

	rows, err := r.db.QueryContext(ctx, query, outerArg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flows []InstrumentCashFlow
	for rows.Next() {
		var f InstrumentCashFlow
		if err := rows.Scan(&f.InstrumentID, &f.Date, &f.AmountGBP); err != nil {
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
func (r *Repository) CurrentHoldingsCashFlows(ctx context.Context, scope Scope, filter Filter) ([]CashFlow, error) {
	outerFilter, outerArg := buildScopeFilter(scope, "pt", 1)
	innerFilter, _ := buildScopeFilter(scope, "", 1)
	args := []any{outerArg}

	where := outerFilter
	if instFilter, instArg, ok := buildInstrumentFilter(filter, "pt", 2); ok {
		where += " AND " + instFilter
		args = append(args, instArg)
	}

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
					WHEN 'STOCK_SPLIT_OPEN' THEN shares
					WHEN 'STOCK_SPLIT_CLOSE' THEN -shares
					WHEN 'STOCK_DISTRIBUTION' THEN shares
					WHEN 'ACQUISITION' THEN shares
					ELSE 0
				END
			) > 0
		)
		ORDER BY pt.timestamp;`, where, innerFilter)

	rows, err := r.db.QueryContext(ctx, query, args...)
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
