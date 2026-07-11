package ingest

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

// Result summarizes the outcome of loading a CSV file.
type Result struct {
	Inserted  int
	Duplicate int
	Errors    []RowError
}

type Engine struct {
	db *sql.DB
}

func NewEngine(db *sql.DB) *Engine {
	return &Engine{db: db}
}

// LoadCSV parses r and ingests it inside a single transaction: new rows are
// inserted into source_transactions and normalized into
// portfolio_transactions; rows matching an existing
// (action, transaction_time, isin, internal_id) are counted as duplicates
// and skipped rather than re-inserted.
func (e *Engine) LoadCSV(ctx context.Context, r io.Reader, hasHeader bool) (*Result, error) {
	parsedRows, parseErrs := ParseCSV(r, hasHeader)

	result := &Result{Errors: parseErrs}
	if len(parsedRows) == 0 {
		return result, nil
	}

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	insertQuery := `
		INSERT INTO source_transactions (
			action, transaction_time, isin, ticker, name, notes, internal_id,
			no_of_shares, price_per_share, currency_price_per_share,
			exchange_rate, result, currency_result, total, currency_total,
			withholding_tax, currency_withholding_tax,
			stamp_duty_reserve_tax, currency_stamp_duty_reserve_tax,
			currency_conversion_fee, currency_conversion_fee_currency
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		ON CONFLICT (action, transaction_time, isin, internal_id) DO NOTHING
		RETURNING id;
	`
	stmt, err := tx.PrepareContext(ctx, insertQuery)
	if err != nil {
		return nil, fmt.Errorf("preparing insert statement: %w", err)
	}
	defer stmt.Close()

	var freshInsertedIDs []uuid.UUID
	for _, row := range parsedRows {
		var insertedID uuid.UUID
		err := stmt.QueryRowContext(ctx,
			row.Action, row.TransactionTime, row.ISIN, row.Ticker, row.Name, row.Notes, row.InternalID,
			row.NoOfShares, row.PricePerShare, row.CurrencyPricePerShare,
			row.ExchangeRate, row.Result, row.CurrencyResult, row.Total, row.CurrencyTotal,
			row.WithholdingTax, row.CurrencyWithholding,
			row.StampDuty, row.CurrencyStamp,
			row.ConversionFee, row.CurrencyConversion,
		).Scan(&insertedID)

		switch {
		case err == sql.ErrNoRows:
			// ON CONFLICT DO NOTHING suppressed the row: a duplicate.
			result.Duplicate++
		case err != nil:
			if pqErr, ok := err.(*pq.Error); ok {
				result.Errors = append(result.Errors, RowError{Err: fmt.Errorf("insert failed: %s", pqErr.Message)})
			} else {
				result.Errors = append(result.Errors, RowError{Err: fmt.Errorf("insert failed: %w", err)})
			}
		default:
			freshInsertedIDs = append(freshInsertedIDs, insertedID)
			result.Inserted++
		}
	}

	if len(freshInsertedIDs) > 0 {
		if err := e.normalizeWithinTx(ctx, tx, freshInsertedIDs); err != nil {
			return nil, fmt.Errorf("normalizing ledger entries: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return result, nil
}

// normalizeWithinTx mirrors the standalone csv_loader's ingestion engine:
// fetch the freshly inserted raw rows, resolve/create their instruments,
// and insert normalized ledger entries into portfolio_transactions.
func (e *Engine) normalizeWithinTx(ctx context.Context, tx *sql.Tx, targetIDs []uuid.UUID) error {
	selectQuery := `
		SELECT
			id, action, transaction_time, isin, ticker, name,
			no_of_shares, price_per_share, currency_price_per_share,
			exchange_rate, total, currency_total, withholding_tax,
			stamp_duty_reserve_tax, currency_conversion_fee
		FROM source_transactions
		WHERE id = ANY($1)
		ORDER BY transaction_time ASC;`

	rows, err := tx.QueryContext(ctx, selectQuery, pq.Array(targetIDs))
	if err != nil {
		return fmt.Errorf("fetching inserted rows: %w", err)
	}
	defer rows.Close()

	var pool []RawSourceTransaction
	for rows.Next() {
		var raw RawSourceTransaction
		if err := rows.Scan(
			&raw.ID, &raw.Action, &raw.TransactionTime, &raw.ISIN, &raw.Ticker, &raw.Name,
			&raw.NoOfShares, &raw.PricePerShare, &raw.CurrencyPricePerShare,
			&raw.ExchangeRate, &raw.Total, &raw.CurrencyTotal, &raw.WithholdingTax,
			&raw.StampDutyReserveTax, &raw.CurrencyConversionFee,
		); err != nil {
			return err
		}
		pool = append(pool, raw)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	insertQuery := `
		INSERT INTO portfolio_transactions (
			source_transaction_id, instrument_id, entry_type, timestamp,
			shares, price_per_share_local, fx_rate_to_gbp,
			net_amount_gbp, fees_gbp, taxes_gbp, total_impact_gbp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);`

	insertStmt, err := tx.PrepareContext(ctx, insertQuery)
	if err != nil {
		return err
	}
	defer insertStmt.Close()

	for _, raw := range pool {
		var instrumentID *uuid.UUID
		entryType := MapAction(raw.Action)

		if isAssetEntry(entryType) {
			id, err := e.getOrCreateInstrument(ctx, tx, raw.ISIN.String, raw.Ticker.String, raw.Name.String, raw.CurrencyPricePerShare)
			if err != nil {
				return err
			}
			instrumentID = &id
		}

		if strings.ToUpper(raw.CurrencyPricePerShare) == "GBX" {
			raw.PricePerShare = raw.PricePerShare.Div(decimal.NewFromInt(100))
		}

		normalized, err := NormalizeTransaction(raw, instrumentID)
		if err != nil {
			return err
		}

		_, err = insertStmt.ExecContext(ctx,
			normalized.SourceID, normalized.InstrumentID, normalized.EntryType, normalized.Timestamp,
			normalized.Shares, normalized.PricePerShareLocal, normalized.FXRateToGBP,
			normalized.NetAmountGBP, normalized.FeesGBP, normalized.TaxesGBP, normalized.TotalImpactGBP,
		)
		if err != nil {
			return fmt.Errorf("inserting ledger entry: %w", err)
		}
	}

	return nil
}

func (e *Engine) getOrCreateInstrument(ctx context.Context, tx *sql.Tx, isin, ticker, name, rawCurrency string) (uuid.UUID, error) {
	baseCurrency := normalizeCurrency(rawCurrency)

	var id uuid.UUID
	query := `
		INSERT INTO instruments (isin, ticker, name, base_currency)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (isin) DO UPDATE
		SET ticker = EXCLUDED.ticker, name = EXCLUDED.name
		RETURNING id;`

	err := tx.QueryRowContext(ctx, query, isin, ticker, name, baseCurrency).Scan(&id)
	return id, err
}
