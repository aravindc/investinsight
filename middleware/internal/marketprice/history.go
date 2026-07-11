package marketprice

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// PricesByDate returns every stored close for instrumentID between from and
// to (inclusive), keyed by date (midnight UTC). Used to reconstruct
// historical value without one query per day.
func (r *Repository) PricesByDate(ctx context.Context, instrumentID uuid.UUID, from, to time.Time) (map[time.Time]StoredPrice, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT price_date, close_price, currency
		FROM market_prices
		WHERE instrument_id = $1 AND price_date BETWEEN $2 AND $3
		ORDER BY price_date;`, instrumentID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[time.Time]StoredPrice)
	for rows.Next() {
		var p StoredPrice
		if err := rows.Scan(&p.Date, &p.Close, &p.Currency); err != nil {
			return nil, err
		}
		result[p.Date.UTC()] = p
	}
	return result, rows.Err()
}

// EarliestPriceDate returns the earliest stored price_date across all of
// the given instruments — i.e. the earliest day any of them has any price
// data at all. Used as a best-effort series start date: the chart begins
// as soon as something is priceable, without requiring every instrument to
// have full history from that point (a given day's total simply omits
// instruments not yet priced on that day). Returns a zero time if none of
// the instruments have any price data.
func (r *Repository) EarliestPriceDate(ctx context.Context, instrumentIDs []uuid.UUID) (time.Time, error) {
	if len(instrumentIDs) == 0 {
		return time.Time{}, nil
	}

	var earliest sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT MIN(price_date) FROM market_prices WHERE instrument_id = ANY($1)`,
		pq.Array(instrumentIDs),
	).Scan(&earliest)
	if err != nil {
		return time.Time{}, err
	}
	if !earliest.Valid {
		return time.Time{}, nil
	}
	return earliest.Time, nil
}
