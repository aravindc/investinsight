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

// Instrument is one tradable instrument with its manually-assigned
// classification: a Category → Sub-Category pair (internal/category) and an
// independent Sector (internal/sector).
type Instrument struct {
	ID          uuid.UUID `json:"id"`
	Ticker      string    `json:"ticker"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	SubCategory string    `json:"subCategory"`
	Sector      string    `json:"sector"`
	SubSector   string    `json:"subSector"` // "" when not set
}

// Classification is the editable part of an Instrument. SubSector is
// optional ("" clears it); when set it must belong to Sector.
type Classification struct {
	Category    string `json:"category"`
	SubCategory string `json:"subCategory"`
	Sector      string `json:"sector"`
	SubSector   string `json:"subSector"`
}

// List returns every instrument, ordered by ticker.
func (r *Repository) List(ctx context.Context) ([]Instrument, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, ticker, name, category, sub_category, sector, COALESCE(sub_sector, '')
		FROM instruments ORDER BY ticker;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instruments []Instrument
	for rows.Next() {
		var inst Instrument
		if err := rows.Scan(&inst.ID, &inst.Ticker, &inst.Name, &inst.Category, &inst.SubCategory, &inst.Sector, &inst.SubSector); err != nil {
			return nil, err
		}
		instruments = append(instruments, inst)
	}
	return instruments, rows.Err()
}

// UpdateClassification sets instrumentID's category, sub-category and
// sector. Foreign keys reject unknown values or a sub-category from another
// category. Returns sql.ErrNoRows if no instrument with that id exists.
func (r *Repository) UpdateClassification(ctx context.Context, instrumentID uuid.UUID, cl Classification) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE instruments SET category = $1, sub_category = $2, sector = $3, sub_sector = NULLIF($4, '')
		WHERE id = $5;`,
		cl.Category, cl.SubCategory, cl.Sector, cl.SubSector, instrumentID)
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
