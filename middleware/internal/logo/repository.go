package logo

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Instrument is the minimal shape needed to fetch/store a logo.
type Instrument struct {
	ID           uuid.UUID
	Ticker       string
	BaseCurrency string
}

// InstrumentsMissingLogo lists every instrument that doesn't yet have a
// stored logo row — the set to fetch on each daily backfill run.
func (r *Repository) InstrumentsMissingLogo(ctx context.Context) ([]Instrument, error) {
	query := `
		SELECT i.id, i.ticker, i.base_currency
		FROM instruments i
		LEFT JOIN instrument_logos l ON l.instrument_id = i.id
		WHERE l.instrument_id IS NULL;`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instruments []Instrument
	for rows.Next() {
		var inst Instrument
		if err := rows.Scan(&inst.ID, &inst.Ticker, &inst.BaseCurrency); err != nil {
			return nil, err
		}
		instruments = append(instruments, inst)
	}
	return instruments, rows.Err()
}

// UpsertLogo stores (or replaces) the logo image for instrumentID.
func (r *Repository) UpsertLogo(ctx context.Context, instrumentID uuid.UUID, data []byte, contentType string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO instrument_logos (instrument_id, image_data, content_type, fetched_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (instrument_id) DO UPDATE
		SET image_data = EXCLUDED.image_data, content_type = EXCLUDED.content_type, fetched_at = EXCLUDED.fetched_at;`,
		instrumentID, data, contentType, time.Now().UTC())
	return err
}

// StoredLogo is a previously-fetched logo image.
type StoredLogo struct {
	Data        []byte
	ContentType string
}

// LogoByInstrument returns the stored logo for instrumentID, or nil if none
// has been fetched yet.
func (r *Repository) LogoByInstrument(ctx context.Context, instrumentID uuid.UUID) (*StoredLogo, error) {
	var logo StoredLogo
	err := r.db.QueryRowContext(ctx, `
		SELECT image_data, content_type FROM instrument_logos WHERE instrument_id = $1;`,
		instrumentID,
	).Scan(&logo.Data, &logo.ContentType)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &logo, nil
}
