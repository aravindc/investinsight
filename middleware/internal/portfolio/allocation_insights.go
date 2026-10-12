package portfolio

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AllocationGroup identifies one group of a dimension.
type AllocationGroup struct {
	Name         string  `json:"name"`
	Parent       *string `json:"parent"`
	Unclassified bool    `json:"unclassified"`
}

func (k groupKey) group() AllocationGroup {
	g := AllocationGroup{Name: k.name, Unclassified: k.unclassified}
	if k.hasParent {
		parent := k.parent
		g.Parent = &parent
	}
	return g
}

// MatrixCell is the holdings in one row group and one column group.
type MatrixCell struct {
	Row int `json:"row"`
	Col int `json:"col"`
	AllocationFigures
	// SharePercent is the cell's share of total market value.
	SharePercent decimal.Decimal `json:"sharePercent"`
}

// AllocationMatrix cross-tabulates holdings by two dimensions. Rows and
// Cols are sorted by market value, descending; Cells hold only non-empty
// combinations and index into them.
type AllocationMatrix struct {
	RowsBy GroupBy           `json:"rowsBy"`
	ColsBy GroupBy           `json:"colsBy"`
	Rows   []AllocationGroup `json:"rows"`
	Cols   []AllocationGroup `json:"cols"`
	Cells  []MatrixCell      `json:"cells"`
	Totals AllocationFigures `json:"totals"`
}

func (s *SummaryService) AllocationMatrix(ctx context.Context, scope Scope, filter Filter, rowsBy, colsBy GroupBy) (*AllocationMatrix, error) {
	if rowsBy == colsBy {
		return nil, fmt.Errorf("rows and cols must be different dimensions")
	}
	holdings, err := s.pricedHoldings(ctx, scope, filter)
	if err != nil {
		return nil, err
	}
	return aggregateMatrix(rowsBy, colsBy, holdings), nil
}

// orderedKeys returns keys sorted by market value then name, with each
// key's index.
func orderedKeys(totals map[groupKey]*figureTotals) ([]groupKey, map[groupKey]int) {
	keys := make([]groupKey, 0, len(totals))
	for k := range totals {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := totals[keys[i]], totals[keys[j]]
		if !a.marketValue.Equal(b.marketValue) {
			return a.marketValue.GreaterThan(b.marketValue)
		}
		if keys[i].parent != keys[j].parent {
			return keys[i].parent < keys[j].parent
		}
		return keys[i].name < keys[j].name
	})
	index := make(map[groupKey]int, len(keys))
	for i, k := range keys {
		index[k] = i
	}
	return keys, index
}

func aggregateMatrix(rowsBy, colsBy GroupBy, holdings []pricedHolding) *AllocationMatrix {
	type cellKey struct{ row, col groupKey }
	var total figureTotals
	rows := make(map[groupKey]*figureTotals)
	cols := make(map[groupKey]*figureTotals)
	cells := make(map[cellKey]*figureTotals)
	add := func(m map[groupKey]*figureTotals, k groupKey, h pricedHolding) {
		if m[k] == nil {
			m[k] = &figureTotals{}
		}
		m[k].add(h, decimal.Zero)
	}

	for _, h := range holdings {
		rk, ck := keyFor(rowsBy, h.CurrentHolding), keyFor(colsBy, h.CurrentHolding)
		add(rows, rk, h)
		add(cols, ck, h)
		c := cellKey{rk, ck}
		if cells[c] == nil {
			cells[c] = &figureTotals{}
		}
		cells[c].add(h, decimal.Zero)
		total.add(h, decimal.Zero)
	}

	rowKeys, rowIndex := orderedKeys(rows)
	colKeys, colIndex := orderedKeys(cols)
	m := &AllocationMatrix{
		RowsBy: rowsBy,
		ColsBy: colsBy,
		Rows:   make([]AllocationGroup, len(rowKeys)),
		Cols:   make([]AllocationGroup, len(colKeys)),
		Cells:  make([]MatrixCell, 0, len(cells)),
		Totals: total.figures(),
	}
	for i, k := range rowKeys {
		m.Rows[i] = k.group()
	}
	for i, k := range colKeys {
		m.Cols[i] = k.group()
	}
	for k, t := range cells {
		cell := MatrixCell{Row: rowIndex[k.row], Col: colIndex[k.col], AllocationFigures: t.figures()}
		if total.marketValue.IsPositive() {
			cell.SharePercent = t.marketValue.Div(total.marketValue).Mul(decimal.NewFromInt(100)).Round(2)
		}
		m.Cells = append(m.Cells, cell)
	}
	sort.Slice(m.Cells, func(i, j int) bool {
		if m.Cells[i].Row != m.Cells[j].Row {
			return m.Cells[i].Row < m.Cells[j].Row
		}
		return m.Cells[i].Col < m.Cells[j].Col
	})
	return m
}

// AllocationSeriesPoint is each group's market value on one date, aligned
// with AllocationSeries.Groups.
type AllocationSeriesPoint struct {
	Date      string            `json:"date"`
	ValuesGBP []decimal.Decimal `json:"valuesGbp"`
}

// AllocationSeries is month-end market value per group, plus today, for
// instruments held today, classified as they are today.
type AllocationSeries struct {
	GroupBy GroupBy                 `json:"groupBy"`
	Groups  []AllocationGroup       `json:"groups"`
	Points  []AllocationSeriesPoint `json:"points"`
}

func (s *SummaryService) AllocationSeries(ctx context.Context, scope Scope, filter Filter, groupBy GroupBy) (*AllocationSeries, error) {
	holdings, err := s.repo.CurrentHoldings(ctx, scope, filter)
	if err != nil {
		return nil, err
	}

	keyIndex := make(map[groupKey]int)
	var keys []groupKey
	instrumentKey := make(map[uuid.UUID]string, len(holdings))
	for _, h := range holdings {
		k := keyFor(groupBy, h)
		i, ok := keyIndex[k]
		if !ok {
			i = len(keys)
			keyIndex[k] = i
			keys = append(keys, k)
		}
		instrumentKey[h.InstrumentID] = strconv.Itoa(i)
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	var points []AllocationSeriesPoint
	err = s.walkDailyValues(ctx, scope,
		func(id uuid.UUID) (string, bool) {
			key, ok := instrumentKey[id]
			return key, ok
		},
		func(day time.Time, totals map[string]decimal.Decimal) {
			monthEnd := day.AddDate(0, 0, 1).Day() == 1
			if !monthEnd && !day.Equal(today) {
				return
			}
			values := make([]decimal.Decimal, len(keys))
			for i := range keys {
				values[i] = totals[strconv.Itoa(i)].Round(2)
			}
			points = append(points, AllocationSeriesPoint{Date: day.Format("2006-01-02"), ValuesGBP: values})
		})
	if err != nil {
		return nil, err
	}
	return buildAllocationSeries(groupBy, keys, points), nil
}

// buildAllocationSeries orders groups by their latest value, descending,
// reordering each point's values to match.
func buildAllocationSeries(groupBy GroupBy, keys []groupKey, points []AllocationSeriesPoint) *AllocationSeries {
	order := make([]int, len(keys))
	for i := range order {
		order[i] = i
	}
	if len(points) > 0 {
		latest := points[len(points)-1].ValuesGBP
		sort.SliceStable(order, func(a, b int) bool {
			return latest[order[a]].GreaterThan(latest[order[b]])
		})
	}

	series := &AllocationSeries{
		GroupBy: groupBy,
		Groups:  make([]AllocationGroup, len(keys)),
		Points:  make([]AllocationSeriesPoint, len(points)),
	}
	for i, k := range order {
		series.Groups[i] = keys[k].group()
	}
	for p, point := range points {
		values := make([]decimal.Decimal, len(order))
		for i, k := range order {
			values[i] = point.ValuesGBP[k]
		}
		series.Points[p] = AllocationSeriesPoint{Date: point.Date, ValuesGBP: values}
	}
	return series
}
