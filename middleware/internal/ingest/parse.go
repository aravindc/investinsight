package ingest

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/shopspring/decimal"
)

// ParsedRow is one CSV row's fields, prior to any DB interaction.
type ParsedRow struct {
	Action                string
	TransactionTime       time.Time
	ISIN                  string
	Ticker                string
	Name                  string
	Notes                 string
	InternalID            string
	NoOfShares            decimal.Decimal
	PricePerShare         decimal.Decimal
	CurrencyPricePerShare string
	ExchangeRate          decimal.Decimal
	Result                decimal.Decimal
	CurrencyResult        string
	Total                 decimal.Decimal
	CurrencyTotal         string
	WithholdingTax        decimal.Decimal
	CurrencyWithholding   string
	StampDuty             decimal.Decimal
	CurrencyStamp         string
	ConversionFee         decimal.Decimal
	CurrencyConversion    string
}

// RowError describes a single CSV row that failed to parse.
type RowError struct {
	Line int
	Err  error
}

func (e RowError) Error() string {
	return fmt.Sprintf("row %d: %v", e.Line, e.Err)
}

// ParseCSV reads broker CSV rows from r. Rows that fail to parse are
// skipped and reported in errs rather than aborting the whole file.
func ParseCSV(r io.Reader, hasHeader bool) (rows []ParsedRow, errs []RowError) {
	reader := csv.NewReader(r)

	lineOffset := 1
	if hasHeader {
		if _, err := reader.Read(); err != nil {
			errs = append(errs, RowError{Line: 1, Err: fmt.Errorf("reading header: %w", err)})
			return nil, errs
		}
		lineOffset = 2
	}

	parseDecimal := func(val string) decimal.Decimal {
		if val == "" {
			return decimal.Zero
		}
		d, err := decimal.NewFromString(val)
		if err != nil {
			return decimal.Zero
		}
		return d
	}

	line := lineOffset
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errs = append(errs, RowError{Line: line, Err: err})
			line++
			continue
		}

		if len(record) < 21 {
			errs = append(errs, RowError{Line: line, Err: fmt.Errorf("expected 21 columns, got %d", len(record))})
			line++
			continue
		}

		txTime, err := time.Parse("2006-01-02 15:04:05", record[1])
		if err != nil {
			errs = append(errs, RowError{Line: line, Err: fmt.Errorf("invalid time %q", record[1])})
			line++
			continue
		}

		rows = append(rows, ParsedRow{
			Action:                record[0],
			TransactionTime:       txTime,
			ISIN:                  record[2],
			Ticker:                record[3],
			Name:                  record[4],
			Notes:                 record[5],
			InternalID:            record[6],
			NoOfShares:            parseDecimal(record[7]),
			PricePerShare:         parseDecimal(record[8]),
			CurrencyPricePerShare: record[9],
			ExchangeRate:          parseDecimal(record[10]),
			Result:                parseDecimal(record[11]),
			CurrencyResult:        record[12],
			Total:                 parseDecimal(record[13]),
			CurrencyTotal:         record[14],
			WithholdingTax:        parseDecimal(record[15]),
			CurrencyWithholding:   record[16],
			StampDuty:             parseDecimal(record[17]),
			CurrencyStamp:         record[18],
			ConversionFee:         parseDecimal(record[19]),
			CurrencyConversion:    record[20],
		})
		line++
	}

	return rows, errs
}
