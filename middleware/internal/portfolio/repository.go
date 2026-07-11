package portfolio

import (
	"context"
	"database/sql"

	"github.com/shopspring/decimal"
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

// HoldingsSummary computes the invested (cost basis) value of instruments
// still held: net shares (BUY/SELL/STOCK_SPLIT/STOCK_DISTRIBUTION/ACQUISITION)
// greater than zero, summing net BUY/SELL cash flow for those instruments.
// Market value is not yet available (no price feed) and is reported as zero.
func (r *Repository) HoldingsSummary(ctx context.Context) (*HoldingsSummary, error) {
	query := `
		SELECT COALESCE(SUM(invested_gbp), 0)
		FROM (
			SELECT
				instrument_id,
				SUM(
					CASE entry_type
						WHEN 'BUY' THEN shares
						WHEN 'SELL' THEN -shares
						WHEN 'STOCK_SPLIT' THEN shares
						WHEN 'STOCK_DISTRIBUTION' THEN shares
						WHEN 'ACQUISITION' THEN shares
						ELSE 0
					END
				) AS net_shares,
				SUM(
					CASE entry_type
						WHEN 'BUY' THEN net_amount_gbp
						WHEN 'SELL' THEN -net_amount_gbp
						ELSE 0
					END
				) AS invested_gbp
			FROM portfolio_transactions
			WHERE instrument_id IS NOT NULL
			GROUP BY instrument_id
		) per_instrument
		WHERE net_shares > 0;`

	var invested decimal.Decimal
	if err := r.db.QueryRowContext(ctx, query).Scan(&invested); err != nil {
		return nil, err
	}

	return &HoldingsSummary{
		InvestedValueGBP: invested,
		MarketValueGBP:   decimal.Zero,
	}, nil
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
		var day string
		var value decimal.Decimal
		if err := rows.Scan(&day, &value); err != nil {
			return nil, err
		}
		series = append(series, ValuePoint{Date: day, ValueGBP: value})
	}

	return series, rows.Err()
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
