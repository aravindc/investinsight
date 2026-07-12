package account

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Account is a single account within a household (e.g. an ISA or SIPP),
// the unit that CSV imports are scoped to.
type Account struct {
	ID          uuid.UUID `json:"id"`
	HouseholdID uuid.UUID `json:"householdId"`
	Name        string    `json:"name"`
}

// List returns every account, ordered by name.
func (r *Repository) List(ctx context.Context) ([]Account, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, household_id, name FROM accounts ORDER BY name;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.HouseholdID, &a.Name); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

// Create inserts a new account with the given name under householdID.
func (r *Repository) Create(ctx context.Context, householdID uuid.UUID, name string) (*Account, error) {
	a := &Account{HouseholdID: householdID, Name: name}
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO accounts (household_id, name) VALUES ($1, $2) RETURNING id;`,
		householdID, name,
	).Scan(&a.ID)
	if err != nil {
		return nil, err
	}
	return a, nil
}
