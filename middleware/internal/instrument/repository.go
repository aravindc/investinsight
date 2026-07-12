package instrument

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

// Instrument is one tradable instrument, with its manually-assigned sector
// (see internal/sector for the fixed set of valid sector names).
type Instrument struct {
	ID     uuid.UUID `json:"id"`
	Ticker string    `json:"ticker"`
	Name   string    `json:"name"`
	Sector string    `json:"sector"`
}

// List returns every instrument, ordered by ticker.
func (r *Repository) List(ctx context.Context) ([]Instrument, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, ticker, name, sector FROM instruments ORDER BY ticker;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instruments []Instrument
	for rows.Next() {
		var inst Instrument
		if err := rows.Scan(&inst.ID, &inst.Ticker, &inst.Name, &inst.Sector); err != nil {
			return nil, err
		}
		instruments = append(instruments, inst)
	}
	return instruments, rows.Err()
}

// UpdateSector sets instrumentID's sector. Returns sql.ErrNoRows if no
// instrument with that id exists.
func (r *Repository) UpdateSector(ctx context.Context, instrumentID uuid.UUID, sector string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE instruments SET sector = $1 WHERE id = $2;`, sector, instrumentID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
