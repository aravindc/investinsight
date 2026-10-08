package cashisa

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

// stored is a Cash ISA as persisted: the anchor balance and its rates.
type stored struct {
	AccountID   uuid.UUID
	AccountName string
	Balance     decimal.Decimal
	BalanceDate time.Time
	Rates       []Rate
}

// List returns the Cash ISAs for one account, every account in a
// household, or (scope nil) all accounts, ordered by account name.
func (r *Repository) List(ctx context.Context, scope *Scope) ([]stored, error) {
	where, args := "TRUE", []any{}
	if scope != nil {
		if scope.Household {
			where, args = "a.household_id = $1", []any{scope.HouseholdID}
		} else {
			where, args = "a.id = $1", []any{scope.AccountID}
		}
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT c.account_id, a.name, c.balance, c.balance_date,
			r.effective_from, r.annual_rate_percent
		FROM cash_isas c
		JOIN accounts a ON a.id = c.account_id
		LEFT JOIN cash_isa_rates r ON r.account_id = c.account_id
		WHERE `+where+`
		ORDER BY a.name, c.account_id, r.effective_from;`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []stored
	for rows.Next() {
		var s stored
		var from sql.NullTime
		var rate decimal.NullDecimal
		if err := rows.Scan(&s.AccountID, &s.AccountName, &s.Balance, &s.BalanceDate, &from, &rate); err != nil {
			return nil, err
		}
		if n := len(out); n == 0 || out[n-1].AccountID != s.AccountID {
			out = append(out, s)
		}
		if from.Valid && rate.Valid {
			last := &out[len(out)-1]
			last.Rates = append(last.Rates, Rate{EffectiveFrom: from.Time, AnnualPercent: rate.Decimal})
		}
	}
	return out, rows.Err()
}

// Save sets the account's balance as of balanceDate and records ratePercent
// as applying from that date.
func (r *Repository) Save(ctx context.Context, accountID uuid.UUID, balance decimal.Decimal, balanceDate time.Time, ratePercent decimal.Decimal) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO cash_isas (account_id, balance, balance_date)
		VALUES ($1, $2, $3)
		ON CONFLICT (account_id) DO UPDATE SET
			balance = EXCLUDED.balance, balance_date = EXCLUDED.balance_date, updated_at = NOW();`,
		accountID, balance, balanceDate); err != nil {
		return err
	}
	if err := upsertRate(ctx, tx, accountID, balanceDate, ratePercent); err != nil {
		return err
	}
	return tx.Commit()
}

// SetRate records ratePercent as applying from effectiveFrom. It returns
// false if the account has no Cash ISA.
func (r *Repository) SetRate(ctx context.Context, accountID uuid.UUID, effectiveFrom time.Time, ratePercent decimal.Decimal) (bool, error) {
	var exists bool
	if err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM cash_isas WHERE account_id = $1)`, accountID).Scan(&exists); err != nil || !exists {
		return false, err
	}
	return true, upsertRate(ctx, r.db, accountID, effectiveFrom, ratePercent)
}

func upsertRate(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, accountID uuid.UUID, effectiveFrom time.Time, ratePercent decimal.Decimal) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO cash_isa_rates (account_id, effective_from, annual_rate_percent)
		VALUES ($1, $2, $3)
		ON CONFLICT (account_id, effective_from) DO UPDATE SET annual_rate_percent = EXCLUDED.annual_rate_percent;`,
		accountID, effectiveFrom, ratePercent)
	return err
}

func (r *Repository) Delete(ctx context.Context, accountID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM cash_isas WHERE account_id = $1`, accountID)
	return err
}
