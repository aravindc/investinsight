package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <input.csv> <output.csv>\n", os.Args[0])
		os.Exit(1)
	}

	if err := convert(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func convert(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	reader := csv.NewReader(src)
	header, err := reader.Read()
	if err != nil {
		return err
	}
	col := make(map[string]int, len(header))
	for i, name := range header {
		col[name] = i
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	writer := csv.NewWriter(dst)
	defer writer.Flush()

	if err := writer.Write([]string{"date", "action", "ticker", "name", "no.shares", "pricepershare", "currency"}); err != nil {
		return err
	}

	for {
		row, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}

		action := row[col["Action"]]
		if action != "Market buy" && action != "Market sell" && !strings.HasPrefix(action, "Dividend (") {
			continue
		}

		ticker := row[col["Ticker"]]
		if ticker == "" {
			continue
		}
		currency := row[col["Currency (Price / share)"]]
		price, err := strconv.ParseFloat(row[col["Price / share"]], 64)
		if err != nil {
			return fmt.Errorf("parsing price for %s: %w", ticker, err)
		}

		switch currency {
		case "GBX":
			price = price / 100.0
			currency = "GBP"
			ticker += ".L"
		case "GBP":
			ticker += ".L"
		case "EUR":
			ticker += ".DE"
		}

		out := []string{
			row[col["Time"]],
			action,
			ticker,
			row[col["Name"]],
			row[col["No. of shares"]],
			strconv.FormatFloat(price, 'f', 6, 64),
			currency,
		}
		if err := writer.Write(out); err != nil {
			return err
		}
	}

	return nil
}
