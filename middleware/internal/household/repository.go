package household

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

// Household groups one or more accounts (e.g. a family sharing a dashboard).
type Household struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// List returns every household, ordered by name.
func (r *Repository) List(ctx context.Context) ([]Household, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name FROM households ORDER BY name;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var households []Household
	for rows.Next() {
		var h Household
		if err := rows.Scan(&h.ID, &h.Name); err != nil {
			return nil, err
		}
		households = append(households, h)
	}
	return households, rows.Err()
}

// Create inserts a new household with the given name.
func (r *Repository) Create(ctx context.Context, name string) (*Household, error) {
	h := &Household{Name: name}
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO households (name) VALUES ($1) RETURNING id;`, name,
	).Scan(&h.ID)
	if err != nil {
		return nil, err
	}
	return h, nil
}
