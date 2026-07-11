package marketprice

import (
	"context"
	"database/sql"
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

type StoredPrice struct {
	Date     time.Time
	Close    decimal.Decimal
	Currency string
}

// Instrument is the minimal shape needed to backfill/price an instrument.
type Instrument struct {
	ID           uuid.UUID
	Ticker       string
	BaseCurrency string
}

// InstrumentsNeedingPrices lists every instrument with at least one
// share-affecting event (BUY, SELL, STOCK_SPLIT, STOCK_DISTRIBUTION,
// ACQUISITION — matching how "currently held" is defined elsewhere), along
// with the date of its earliest such event (the backfill start date). Using
// only BUY here would miss instruments acquired purely via a corporate
// action (e.g. a spin-off distribution with no direct purchase).
func (r *Repository) InstrumentsNeedingPrices(ctx context.Context) ([]struct {
	Instrument
	FirstBuyDate time.Time
}, error) {
	query := `
		SELECT i.id, i.ticker, i.base_currency, MIN(pt.timestamp)
		FROM instruments i
		JOIN portfolio_transactions pt ON pt.instrument_id = i.id
		WHERE pt.entry_type IN ('BUY', 'SELL', 'STOCK_SPLIT', 'STOCK_DISTRIBUTION', 'ACQUISITION')
		GROUP BY i.id, i.ticker, i.base_currency;`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []struct {
		Instrument
		FirstBuyDate time.Time
	}
	for rows.Next() {
		var item struct {
			Instrument
			FirstBuyDate time.Time
		}
		if err := rows.Scan(&item.ID, &item.Ticker, &item.BaseCurrency, &item.FirstBuyDate); err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

// LatestPriceDate returns the most recent price_date stored for an
// instrument, or a zero time if none exist yet.
func (r *Repository) LatestPriceDate(ctx context.Context, instrumentID uuid.UUID) (time.Time, error) {
	var date sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT MAX(price_date) FROM market_prices WHERE instrument_id = $1`,
		instrumentID,
	).Scan(&date)
	if err != nil {
		return time.Time{}, err
	}
	if !date.Valid {
		return time.Time{}, nil
	}
	return date.Time, nil
}

// UpsertPrices stores daily closes, replacing any existing row for the same
// (instrument, date).
func (r *Repository) UpsertPrices(ctx context.Context, instrumentID uuid.UUID, closes []DailyClose) error {
	if len(closes) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO market_prices (instrument_id, price_date, close_price, currency, source)
		VALUES ($1, $2, $3, $4, 'yahoo')
		ON CONFLICT (instrument_id, price_date)
		DO UPDATE SET close_price = EXCLUDED.close_price, currency = EXCLUDED.currency, fetched_at = NOW();`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range closes {
		if _, err := stmt.ExecContext(ctx, instrumentID, c.Date, c.Close, c.Currency); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// PriceOnOrBefore returns the most recent stored close on or before the
// given date (the standard "last known price" fallback for non-trading
// days), or nil if no price exists on/before that date.
func (r *Repository) PriceOnOrBefore(ctx context.Context, instrumentID uuid.UUID, date time.Time) (*StoredPrice, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT price_date, close_price, currency
		FROM market_prices
		WHERE instrument_id = $1 AND price_date <= $2
		ORDER BY price_date DESC
		LIMIT 1;`, instrumentID, date)

	var price StoredPrice
	if err := row.Scan(&price.Date, &price.Close, &price.Currency); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &price, nil
}
