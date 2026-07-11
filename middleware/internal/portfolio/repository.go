package portfolio

import (
	"context"
	"database/sql"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"investinsight/middleware/internal/sector"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListTransactions(ctx context.Context, limit int) ([]Transaction, error) {
	query := `
		SELECT
			pt.id, pt.entry_type, pt.timestamp, i.name,
			pt.shares, pt.net_amount_gbp, pt.total_impact_gbp
		FROM portfolio_transactions pt
		LEFT JOIN instruments i ON i.id = pt.instrument_id
		ORDER BY pt.timestamp DESC
		LIMIT $1;`

	rows, err := r.db.QueryContext(ctx, query, limit)
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
func (r *Repository) CurrentHoldings(ctx context.Context) ([]CurrentHolding, error) {
	query := `
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
		) > 0;`

	rows, err := r.db.QueryContext(ctx, query)
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
func (r *Repository) ShareDeltasForCurrentHoldings(ctx context.Context) ([]ShareDelta, error) {
	query := `
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
		WHERE pt.instrument_id IN (
			SELECT instrument_id
			FROM portfolio_transactions
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
		ORDER BY pt.instrument_id, day;`

	rows, err := r.db.QueryContext(ctx, query)
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
func (r *Repository) InvestedValueSeries(ctx context.Context) ([]ValuePoint, error) {
	query := `
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
			WHERE entry_type IN ('BUY', 'SELL')
			GROUP BY DATE(timestamp)
		) daily
		ORDER BY day;`

	rows, err := r.db.QueryContext(ctx, query)
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
func (r *Repository) SectorAllocations(ctx context.Context) ([]SectorAllocation, error) {
	holdings, err := r.CurrentHoldings(ctx)
	if err != nil {
		return nil, err
	}

	byS := make(map[string]decimal.Decimal)
	for _, h := range holdings {
		s := sector.ForTicker(h.Ticker)
		byS[s] = byS[s].Add(h.InvestedGBP)
	}

	allocations := make([]SectorAllocation, 0, len(byS))
	for s, v := range byS {
		allocations = append(allocations, SectorAllocation{Sector: s, InvestedGBP: v.Round(2)})
	}
	sort.Slice(allocations, func(i, j int) bool {
		return allocations[i].InvestedGBP.GreaterThan(allocations[j].InvestedGBP)
	})

	return allocations, nil
}

// DividendsSummary sums net dividend income received across all holdings.
func (r *Repository) DividendsSummary(ctx context.Context) (*DividendsSummary, error) {
	query := `
		SELECT COALESCE(SUM(net_amount_gbp), 0)
		FROM portfolio_transactions
		WHERE entry_type = 'DIVIDEND';`

	var total decimal.Decimal
	if err := r.db.QueryRowContext(ctx, query).Scan(&total); err != nil {
		return nil, err
	}

	return &DividendsSummary{TotalDividendsGBP: total}, nil
}

// MonthlyDividends returns total dividend income received in each calendar
// month that had any, across the full ledger history, ordered
// chronologically. Used to derive both the cumulative dividends-received
// series and the year-over-year dividend growth comparison.
func (r *Repository) MonthlyDividends(ctx context.Context) ([]MonthlyDividend, error) {
	query := `
		SELECT
			EXTRACT(YEAR FROM timestamp)::int AS year,
			EXTRACT(MONTH FROM timestamp)::int AS month,
			SUM(net_amount_gbp) AS total
		FROM portfolio_transactions
		WHERE entry_type = 'DIVIDEND'
		GROUP BY year, month
		ORDER BY year, month;`

	rows, err := r.db.QueryContext(ctx, query)
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
