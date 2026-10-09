package instrument

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"

	"investinsight/middleware/internal/sector"
)

// ClassificationRow is one parsed line of a classification CSV:
// Ticker,Name,Category,Sub-Category,Sector[,Sub-Sector].
type ClassificationRow struct {
	Line        int
	Ticker      string
	Name        string
	Category    string
	SubCategory string
	Sector      string
	// SubSector is only applied when the file has a Sub-Sector column
	// (HasSubSector); a blank cell then clears it.
	SubSector    string
	HasSubSector bool
}

// RowIssue reports a CSV row that wasn't applied.
type RowIssue struct {
	Line   int    `json:"line"`
	Ticker string `json:"ticker"`
	Reason string `json:"reason"`
}

// ImportResult summarizes a classification import.
type ImportResult struct {
	Updated              int        `json:"updated"`
	Unchanged            int        `json:"unchanged"`
	CreatedCategories    []string   `json:"createdCategories"`
	CreatedSubCategories []string   `json:"createdSubCategories"` // "Category / Sub-Category"
	CreatedSectors       []string   `json:"createdSectors"`
	CreatedSubSectors    []string   `json:"createdSubSectors"` // "Sector / Sub-Sector"
	Skipped              []RowIssue `json:"skipped"`
}

var classificationColumns = []string{"ticker", "name", "category", "sub-category", "sector"}

// headerKey normalizes a header cell: case, BOM, spacing and the common
// spellings of "Sub-Category".
func headerKey(h string) string {
	h = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(h, "\uFEFF")))
	switch h {
	case "sub category", "subcategory", "sub_category":
		return "sub-category"
	case "sub sector", "subsector", "sub_sector":
		return "sub-sector"
	}
	return h
}

// ParseClassificationCSV reads a CSV with the header
// Ticker,Name,Category,Sub-Category,Sector and an optional Sub-Sector (any
// order, extra columns ignored). Rows with problems are returned as issues, not errors; only an
// unreadable file or missing header columns fail the whole parse.
func ParseClassificationCSV(r io.Reader) ([]ClassificationRow, []RowIssue, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err == io.EOF {
		return nil, nil, errors.New("the file is empty")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("reading header: %w", err)
	}
	idx := map[string]int{}
	for i, h := range header {
		if _, seen := idx[headerKey(h)]; !seen {
			idx[headerKey(h)] = i
		}
	}
	var missing []string
	for _, col := range classificationColumns {
		if _, ok := idx[col]; !ok {
			missing = append(missing, col)
		}
	}
	if len(missing) > 0 {
		return nil, nil, fmt.Errorf("missing column(s): %s (expected Ticker,Name,Category,Sub-Category,Sector[,Sub-Sector])", strings.Join(missing, ", "))
	}
	_, hasSubSector := idx["sub-sector"]

	var rows []ClassificationRow
	var issues []RowIssue
	seen := map[string]int{} // ticker+name → first line
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		// The reader skips blank lines, so take the line number from it.
		line, _ := reader.FieldPos(0)
		if err != nil {
			var perr *csv.ParseError
			if errors.As(err, &perr) {
				line = perr.Line
			}
			issues = append(issues, RowIssue{Line: line, Reason: err.Error()})
			continue
		}
		get := func(col string) string {
			if i := idx[col]; i < len(rec) {
				return strings.TrimSpace(rec[i])
			}
			return ""
		}
		row := ClassificationRow{Line: line, Ticker: get("ticker"), Name: get("name")}
		if row.Ticker == "" && row.Name == "" && get("category") == "" && get("sub-category") == "" && get("sector") == "" {
			continue // a row of empty cells, e.g. ",,,,"
		}
		if row.Ticker == "" && row.Name == "" {
			// A blank ticker is allowed (some instruments have none) but then
			// the name must identify the instrument.
			issues = append(issues, RowIssue{Line: line, Reason: "ticker and name are both empty"})
			continue
		}

		var bad []string
		for _, f := range []struct {
			label string
			dst   *string
			raw   string
		}{
			{"category", &row.Category, get("category")},
			{"sub-category", &row.SubCategory, get("sub-category")},
			{"sector", &row.Sector, get("sector")},
		} {
			v, err := sector.NormalizeName(f.raw)
			if err != nil {
				bad = append(bad, f.label+": "+strings.TrimPrefix(err.Error(), "sector name "))
				continue
			}
			*f.dst = v
		}
		if hasSubSector {
			row.HasSubSector = true
			if raw := get("sub-sector"); raw != "" {
				v, err := sector.NormalizeName(raw)
				if err != nil {
					bad = append(bad, "sub-sector: "+strings.TrimPrefix(err.Error(), "sector name "))
				}
				row.SubSector = v
			}
		}
		if len(bad) > 0 {
			issues = append(issues, RowIssue{Line: line, Ticker: row.Ticker, Reason: strings.Join(bad, "; ")})
			continue
		}

		key := strings.ToUpper(row.Ticker) + "\x00" + strings.ToLower(row.Name)
		if first, dup := seen[key]; dup {
			issues = append(issues, RowIssue{Line: line, Ticker: row.Ticker, Reason: fmt.Sprintf("duplicate of line %d", first)})
			continue
		}
		seen[key] = line
		rows = append(rows, row)
	}
	return rows, issues, nil
}

// ImportClassifications applies parsed rows in one transaction: each row's
// instrument (matched by ticker, and by name when the ticker is shared) gets
// the row's category, sub-category and sector, creating any of those that
// don't exist yet (matching existing ones case-insensitively).
func (r *Repository) ImportClassifications(ctx context.Context, rows []ClassificationRow, parseIssues []RowIssue) (*ImportResult, error) {
	res := &ImportResult{
		CreatedCategories:    []string{},
		CreatedSubCategories: []string{},
		CreatedSectors:       []string{},
		CreatedSubSectors:    []string{},
		Skipped:              append([]RowIssue{}, parseIssues...),
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	type inst struct {
		id                                         uuid.UUID
		name, category, subCat, sectorN, subSector string
	}
	byTicker := map[string][]inst{}
	instRows, err := tx.QueryContext(ctx, `
		SELECT id, ticker, name, category, sub_category, sector, COALESCE(sub_sector, '') FROM instruments;`)
	if err != nil {
		return nil, err
	}
	for instRows.Next() {
		var i inst
		var ticker string
		if err := instRows.Scan(&i.id, &ticker, &i.name, &i.category, &i.subCat, &i.sectorN, &i.subSector); err != nil {
			instRows.Close()
			return nil, err
		}
		byTicker[strings.ToUpper(strings.TrimSpace(ticker))] = append(byTicker[strings.ToUpper(strings.TrimSpace(ticker))], i)
	}
	instRows.Close()
	if err := instRows.Err(); err != nil {
		return nil, err
	}

	updated := map[uuid.UUID]int{} // instrument → CSV line that set it
	for _, row := range rows {
		candidates := byTicker[strings.ToUpper(row.Ticker)]
		// Shared tickers, and instruments without one, are told apart by name.
		if len(candidates) > 1 || row.Ticker == "" {
			var named []inst
			for _, c := range candidates {
				if strings.EqualFold(strings.TrimSpace(c.name), row.Name) {
					named = append(named, c)
				}
			}
			candidates = named
			if len(candidates) == 0 {
				reason := "ticker matches several instruments; set Name to the exact instrument name to choose one"
				if row.Ticker == "" {
					reason = "no instrument without a ticker has this exact name"
				}
				res.Skipped = append(res.Skipped, RowIssue{row.Line, row.Ticker, reason})
				continue
			}
		}
		if len(candidates) == 0 {
			res.Skipped = append(res.Skipped, RowIssue{row.Line, row.Ticker, "no instrument with this ticker"})
			continue
		}
		target := candidates[0]
		if first, dup := updated[target.id]; dup {
			res.Skipped = append(res.Skipped, RowIssue{row.Line, row.Ticker, fmt.Sprintf("same instrument as line %d", first)})
			continue
		}

		category, created, err := ensureCategory(ctx, tx, row.Category)
		if err != nil {
			return nil, err
		}
		if created {
			res.CreatedCategories = append(res.CreatedCategories, category)
		}
		subCategory, created, err := ensureSubCategory(ctx, tx, category, row.SubCategory)
		if err != nil {
			return nil, err
		}
		if created {
			res.CreatedSubCategories = append(res.CreatedSubCategories, category+" / "+subCategory)
		}
		sectorName, created, err := ensureSector(ctx, tx, row.Sector)
		if err != nil {
			return nil, err
		}
		if created {
			res.CreatedSectors = append(res.CreatedSectors, sectorName)
		}

		// Without a Sub-Sector column the existing sub-sector is kept, unless
		// the sector changed (it can't belong to the new sector).
		subSector := target.subSector
		switch {
		case row.HasSubSector && row.SubSector != "":
			subSector, created, err = ensureSubSector(ctx, tx, sectorName, row.SubSector)
			if err != nil {
				return nil, err
			}
			if created {
				res.CreatedSubSectors = append(res.CreatedSubSectors, sectorName+" / "+subSector)
			}
		case row.HasSubSector || sectorName != target.sectorN:
			subSector = ""
		}

		updated[target.id] = row.Line
		if target.category == category && target.subCat == subCategory && target.sectorN == sectorName && target.subSector == subSector {
			res.Unchanged++
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE instruments SET category = $1, sub_category = $2, sector = $3, sub_sector = NULLIF($4, '')
			WHERE id = $5;`,
			category, subCategory, sectorName, subSector, target.id); err != nil {
			return nil, fmt.Errorf("line %d: %w", row.Line, err)
		}
		res.Updated++
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return res, nil
}

// The ensure* helpers return the stored spelling of a value, creating it
// (ordered after existing entries but before "Other") when missing.

func ensureCategory(ctx context.Context, tx *sql.Tx, name string) (string, bool, error) {
	return ensureName(ctx, tx,
		`SELECT name FROM categories WHERE lower(name) = lower($1)`, []any{name},
		`INSERT INTO categories (name, sort_order)
		 SELECT $1::varchar, COALESCE(MAX(sort_order), 0) + 10 FROM categories WHERE name <> 'Other'`, []any{name})
}

func ensureSubCategory(ctx context.Context, tx *sql.Tx, category, name string) (string, bool, error) {
	return ensureName(ctx, tx,
		`SELECT name FROM sub_categories WHERE category = $1 AND lower(name) = lower($2)`, []any{category, name},
		`INSERT INTO sub_categories (category, name, sort_order)
		 SELECT $1::varchar, $2::varchar, COALESCE(MAX(sort_order), 0) + 10 FROM sub_categories
		 WHERE category = $1::varchar AND name <> 'Other'`, []any{category, name})
}

func ensureSubSector(ctx context.Context, tx *sql.Tx, sectorName, name string) (string, bool, error) {
	return ensureName(ctx, tx,
		`SELECT name FROM sub_sectors WHERE sector = $1 AND lower(name) = lower($2)`, []any{sectorName, name},
		`INSERT INTO sub_sectors (sector, name, sort_order)
		 SELECT $1::varchar, $2::varchar, COALESCE(MAX(sort_order), 0) + 10 FROM sub_sectors WHERE sector = $1::varchar`,
		[]any{sectorName, name})
}

func ensureSector(ctx context.Context, tx *sql.Tx, name string) (string, bool, error) {
	return ensureName(ctx, tx,
		`SELECT name FROM sectors WHERE lower(name) = lower($1)`, []any{name},
		`INSERT INTO sectors (name, sort_order)
		 SELECT $1::varchar, COALESCE(MAX(sort_order), 0) + 10 FROM sectors WHERE name <> 'Other'`, []any{name})
}

func ensureName(ctx context.Context, tx *sql.Tx, lookup string, lookupArgs []any, insert string, insertArgs []any) (string, bool, error) {
	var existing string
	err := tx.QueryRowContext(ctx, lookup, lookupArgs...).Scan(&existing)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", false, err
	}
	if _, err := tx.ExecContext(ctx, insert, insertArgs...); err != nil {
		return "", false, err
	}
	return insertArgs[len(insertArgs)-1].(string), true, nil
}
