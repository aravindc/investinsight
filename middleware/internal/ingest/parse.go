package ingest

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
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

	// Column positions default to the legacy layout; when a header is present they
	// are resolved by name, since newer exports rename "Time" to "Time (UTC)" and
	// insert extra columns such as "Total (Account CCY)".
	cols := legacyColumns
	lineOffset := 1
	if hasHeader {
		header, err := reader.Read()
		if err == io.EOF {
			// An empty file (e.g. an export of a period with no activity)
			// simply has no transactions.
			return nil, nil
		}
		if err != nil {
			errs = append(errs, RowError{Line: 1, Err: fmt.Errorf("reading header: %w", err)})
			return nil, errs
		}
		if cols, err = resolveColumns(header); err != nil {
			errs = append(errs, RowError{Line: 1, Err: err})
			return nil, errs
		}
		lineOffset = 2
	}
	minColumns := 0
	for _, idx := range cols {
		if idx+1 > minColumns {
			minColumns = idx + 1
		}
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

		if len(record) < minColumns {
			errs = append(errs, RowError{Line: line, Err: fmt.Errorf("expected at least %d columns, got %d", minColumns, len(record))})
			line++
			continue
		}

		field := func(col string) string {
			idx, ok := cols[col]
			if !ok {
				return ""
			}
			return record[idx]
		}

		// Prefer the account-currency total when the export provides one.
		totalStr, totalCcy := field(colTotal), field(colTotalCcy)
		if v := strings.TrimSpace(field(colTotalAccountCcy)); v != "" {
			totalStr, totalCcy = v, accountCurrency
		}

		txTime, err := parseTransactionTime(field(colTime))
		if err != nil {
			errs = append(errs, RowError{Line: line, Err: fmt.Errorf("invalid time %q", field(colTime))})
			line++
			continue
		}

		rows = append(rows, ParsedRow{
			Action:                field(colAction),
			TransactionTime:       txTime,
			ISIN:                  field(colISIN),
			Ticker:                field(colTicker),
			Name:                  field(colName),
			Notes:                 field(colNotes),
			InternalID:            field(colID),
			NoOfShares:            parseDecimal(field(colShares)),
			PricePerShare:         parseDecimal(field(colPrice)),
			CurrencyPricePerShare: field(colPriceCcy),
			ExchangeRate:          parseDecimal(field(colExchangeRate)),
			Result:                parseDecimal(field(colResult)),
			CurrencyResult:        field(colResultCcy),
			Total:                 parseDecimal(totalStr),
			CurrencyTotal:         totalCcy,
			WithholdingTax:        parseDecimal(field(colWithholding)),
			CurrencyWithholding:   field(colWithholdingCcy),
			StampDuty:             parseDecimal(field(colStampDuty)),
			CurrencyStamp:         field(colStampDutyCcy),
			ConversionFee:         parseDecimal(field(colConversionFee)),
			CurrencyConversion:    field(colConversionFeeCcy),
		})
		line++
	}

	return rows, errs
}

const (
	colAction           = "Action"
	colTime             = "Time"
	colISIN             = "ISIN"
	colTicker           = "Ticker"
	colName             = "Name"
	colNotes            = "Notes"
	colID               = "ID"
	colShares           = "No. of shares"
	colPrice            = "Price / share"
	colPriceCcy         = "Currency (Price / share)"
	colExchangeRate     = "Exchange rate"
	colResult           = "Result"
	colResultCcy        = "Currency (Result)"
	colTotal            = "Total"
	colTotalCcy         = "Currency (Total)"
	colWithholding      = "Withholding tax"
	colWithholdingCcy   = "Currency (Withholding tax)"
	colStampDuty        = "Stamp duty reserve tax"
	colStampDutyCcy     = "Currency (Stamp duty reserve tax)"
	colConversionFee    = "Currency conversion fee"
	colConversionFeeCcy = "Currency (Currency conversion fee)"

	// colTotalAccountCcy is optional: newer exports carry it alongside "Total",
	// which is then in the instrument's currency for some actions (e.g. ADR Fee).
	colTotalAccountCcy = "Total (Account CCY)"
)

// accountCurrency is the currency "Total (Account CCY)" is denominated in.
const accountCurrency = "GBP"

// legacyColumns is the positional layout of the original 21-column export,
// used when the CSV has no header row.
var legacyColumns = map[string]int{
	colAction: 0, colTime: 1, colISIN: 2, colTicker: 3, colName: 4, colNotes: 5, colID: 6,
	colShares: 7, colPrice: 8, colPriceCcy: 9, colExchangeRate: 10, colResult: 11,
	colResultCcy: 12, colTotal: 13, colTotalCcy: 14, colWithholding: 15,
	colWithholdingCcy: 16, colStampDuty: 17, colStampDutyCcy: 18,
	colConversionFee: 19, colConversionFeeCcy: 20,
}

// headerAliases maps alternative header names onto the canonical column name.
var headerAliases = map[string]string{
	"Time (UTC)": colTime,
}

// resolveColumns maps each required column to its index in the header row.
func resolveColumns(header []string) (map[string]int, error) {
	found := make(map[string]int, len(header))
	for i, h := range header {
		h = strings.TrimSpace(strings.TrimPrefix(h, "\uFEFF"))
		if alias, ok := headerAliases[h]; ok {
			h = alias
		}
		if _, dup := found[h]; !dup {
			found[h] = i
		}
	}

	var missing []string
	for _, col := range requiredColumns {
		if _, ok := found[col]; !ok {
			missing = append(missing, col)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required columns: %s", strings.Join(missing, ", "))
	}

	// Every other known column is optional: exports omit columns with no
	// data in the period (e.g. no sells means no "Result"), and a missing
	// column reads as empty.
	cols := make(map[string]int, len(legacyColumns)+1)
	for col := range legacyColumns {
		if idx, ok := found[col]; ok {
			cols[col] = idx
		}
	}
	if idx, ok := found[colTotalAccountCcy]; ok {
		cols[colTotalAccountCcy] = idx
	}
	return cols, nil
}

// requiredColumns must be present in a CSV header; without them a row can't
// be placed in the ledger at all.
var requiredColumns = []string{colAction, colTime}

// transactionTimeLayouts covers the timestamp formats seen across export versions.
// Layouts without an offset are interpreted as UTC.
var transactionTimeLayouts = []string{
	"2006-01-02 15:04:05",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02 15:04:05.999999999Z07:00",
	time.RFC3339Nano,
}

func parseTransactionTime(val string) (time.Time, error) {
	val = strings.TrimSpace(val)
	var lastErr error
	for _, layout := range transactionTimeLayouts {
		t, err := time.Parse(layout, val)
		if err == nil {
			return t.UTC(), nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}
